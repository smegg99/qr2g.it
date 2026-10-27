// web/tests/generator.test.mjs
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { useGenerator } from '../app/composables/useGenerator.ts'

globalThis.ref = value => ({ value })
globalThis.computed = get => ({
  get value() {
    return get()
  },
})
globalThis.onBeforeUnmount = () => {}

const pending = slug => ({ id: 2, slug, url: `https://qr2g.it/${slug}`, policy: { listing: 'pending' } })
const listed = slug => ({ ...pending(slug), policy: { listing: 'listed' } })

function api(overrides = {}) {
  return {
    createLink: async () => assert.fail('unexpected create'),
    getLink: async () => assert.fail('unexpected read'),
    ...overrides,
  }
}

function deferred() {
  let resolve
  const promise = new Promise(done => resolve = done)
  return { promise, resolve }
}

test('failed or empty verification never submits a link without a token', async () => {
  const rejected = async () => {
    throw new Error('challenge rejected')
  }
  for (const getToken of [rejected, async () => null]) {
    globalThis.useTurnstile = () => ({ isConfigured: true, getToken })
    const form = useGenerator(api())
    assert.equal(await form.create('smegg99/qr2g.it'), null)
    assert.equal(form.error.value.code, 'turnstile_unavailable')
    assert.equal(form.loading.value, false)
  }
})

test('the request outcome follows 201 and 200 and the token is sent', async () => {
  globalThis.useTurnstile = () => ({ isConfigured: true, getToken: async () => 'widget-token' })
  for (const [status, kind] of [[201, 'created'], [200, 'existing']]) {
    const form = useGenerator(api({
      createLink: async (body) => {
        assert.deepEqual(body, { url: 'https://github.com/smegg99/qr2g.it', turnstileToken: 'widget-token' })
        return { status, data: listed('2') }
      },
    }))
    assert.equal((await form.create('smegg99/qr2g.it')).slug, '2')
    assert.equal(form.outcome.value, kind)
    assert.equal(form.loading.value, false)
  }
})

test('server errors keep their stable code and i18n key', async () => {
  globalThis.useTurnstile = () => ({ isConfigured: false })
  const form = useGenerator(api({ createLink: async () => ({ status: 404, error: { code: 'repo_not_found', message: 'errors.repo_not_found' } }) }))
  assert.equal(await form.create('a/b'), null)
  assert.deepEqual(form.error.value, { code: 'repo_not_found', message: 'errors.repo_not_found' })
  assert.equal(await form.create('not a repo'), null)
  assert.equal(form.error.value.code, 'invalid_url')
})

test('reset during verification prevents a late submission', async () => {
  const token = deferred()
  globalThis.useTurnstile = () => ({ isConfigured: true, getToken: () => token.promise })
  const form = useGenerator(api())
  const running = form.create('smegg99/qr2g.it')
  assert.equal(form.stage.value, 'verifying')
  form.reset()
  token.resolve('late-token')
  assert.equal(await running, null)
  assert.equal(form.error.value, null)
  assert.equal(form.loading.value, false)
  assert.equal(form.link.value, null)
})

test('an old request cannot clear the loading state or replace the result of a new one', async () => {
  const oldBody = deferred()
  let attempts = 0
  globalThis.useTurnstile = () => ({ isConfigured: false })
  const form = useGenerator(api({
    createLink: async () => (++attempts === 1 ? oldBody.promise : { status: 201, data: listed('3') }),
  }))
  const previous = form.create('a/old')
  const current = form.create('a/new')
  assert.equal((await current).slug, '3')
  oldBody.resolve({ status: 200, data: listed('1') })
  assert.equal(await previous, null)
  assert.equal(form.link.value.slug, '3')
  assert.equal(form.outcome.value, 'created')
  assert.equal(form.loading.value, false)
})

test('a pending policy is polled only while the same link is current', async (t) => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  globalThis.useTurnstile = () => ({ isConfigured: false })
  let reads = 0
  const form = useGenerator(api({
    createLink: async () => ({ status: 201, data: pending('2') }),
    getLink: async () => ({ status: 200, data: ++reads === 2 ? listed('2') : pending('2') }),
  }))
  await form.create('a/b')
  for (let i = 0; i < 2; i++) {
    t.mock.timers.tick(2500)
    await new Promise(setImmediate)
  }
  assert.equal(reads, 2)
  assert.equal(form.link.value.policy.listing, 'listed')
  t.mock.timers.tick(10000)
  await new Promise(setImmediate)
  assert.equal(reads, 2)
  form.reset()
})
