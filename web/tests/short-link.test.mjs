// web/tests/short-link.test.mjs
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { useShortLink } from '../app/composables/useShortLink.ts'

globalThis.ref = value => ({ value })
globalThis.onBeforeUnmount = () => {}
globalThis.useRuntimeConfig = () => ({ public: { apiBase: '/api/v1' } })

test('failed or empty verification never submits a link without a token', async () => {
  const rejected = async () => {
    throw new Error('challenge rejected')
  }
  for (const getToken of [rejected, async () => null]) {
    globalThis.useTurnstile = () => ({ isConfigured: true, getToken })
    globalThis.fetch = () => {
      throw new Error('unexpected API request')
    }
    const form = useShortLink()
    assert.equal(await form.create({ url: 'smegg99/qr2g.it' }), null)
    assert.equal(form.error.value.code, 'turnstile_unavailable')
    assert.equal(form.loading.value, false)
  }
})

test('successful verification submits its token and preserves the returned slug', async () => {
  globalThis.useTurnstile = () => ({ isConfigured: true, getToken: async () => 'test-widget-token' })
  const link = { id: 2, slug: '2', url: 'https://qr2g.it/2' }
  globalThis.fetch = async (url, options) => {
    assert.equal(url, '/api/v1/links')
    assert.deepEqual(JSON.parse(options.body), { url: 'https://github.com/smegg99/qr2g.it', turnstileToken: 'test-widget-token' })
    return { ok: true, text: async () => JSON.stringify(link) }
  }
  const form = useShortLink()
  assert.deepEqual(await form.create({ url: 'smegg99/qr2g.it' }), link)
  assert.equal(form.error.value, null)
  assert.equal(form.loading.value, false)
})

function deferred() {
  let resolve
  const promise = new Promise(done => resolve = done)
  return { promise, resolve }
}

test('reset during verification prevents a late submission', async () => {
  const token = deferred()
  globalThis.useTurnstile = () => ({ isConfigured: true, getToken: () => token.promise })
  globalThis.fetch = () => assert.fail('cancelled verification submitted a request')
  const form = useShortLink()
  const pending = form.create({ url: 'smegg99/qr2g.it' })
  form.reset()
  token.resolve('late-token')
  assert.equal(await pending, null)
  assert.equal(form.error.value, null)
  assert.equal(form.loading.value, false)
  assert.equal(form.link.value, null)
})

test('an old request cannot clear the loading state of a new one', async () => {
  const oldToken = deferred()
  const newToken = deferred()
  let attempts = 0
  globalThis.useTurnstile = () => ({
    isConfigured: true,
    getToken: () => (++attempts === 1 ? oldToken.promise : newToken.promise),
  })
  globalThis.fetch = async () => ({ ok: true, text: async () => '{"id":2}' })
  const form = useShortLink()
  const previous = form.create({ url: 'smegg99/qr2g.it' })
  form.reset()
  const current = form.create({ url: 'smegg99/qr2g.it' })
  oldToken.resolve('old-token')
  assert.equal(await previous, null)
  assert.equal(form.loading.value, true)
  newToken.resolve('current-token')
  assert.deepEqual(await current, { id: 2 })
  assert.equal(form.loading.value, false)
})

test('a cancelled response cannot replace a newer link', async () => {
  const oldBody = deferred()
  const bodyStarted = deferred()
  let attempts = 0
  globalThis.useTurnstile = () => ({ isConfigured: false })
  globalThis.fetch = async () => ({
    ok: true,
    text: ++attempts === 1
      ? () => {
          bodyStarted.resolve()
          return oldBody.promise
        }
      : async () => '{"id":2}',
  })
  const form = useShortLink()
  const previous = form.create({ url: 'smegg99/qr2g.it' })
  await bodyStarted.promise
  form.reset()
  assert.deepEqual(await form.create({ url: 'smegg99/qr2g.it' }), { id: 2 })
  oldBody.resolve('{"id":1}')
  assert.equal(await previous, null)
  assert.deepEqual(form.link.value, { id: 2 })
})
