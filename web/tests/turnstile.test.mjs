// web/tests/turnstile.test.mjs
import assert from 'node:assert/strict'
import { test } from 'node:test'

let sequence = 0

async function setup() {
  const scripts = []
  const unmount = []
  globalThis.onBeforeUnmount = callback => unmount.push(callback)
  const container = {}
  globalThis.window = {}
  globalThis.document = {
    querySelector: () => scripts[0] ?? null,
    createElement: () => {
      const script = new EventTarget()
      script.remove = () => scripts.splice(scripts.indexOf(script), 1)
      return script
    },
    head: { appendChild: script => scripts.push(script) },
    getElementById: () => container,
  }
  globalThis.useRuntimeConfig = () => ({ public: { turnstileSiteKey: 'test-site-key' } })
  const { useTurnstile } = await import(`../app/composables/useTurnstile.ts?test=${sequence++}`)
  let options
  let removed = 0
  const api = {
    render: (el, opts) => {
      assert.equal(el, container)
      options = opts
      return 'widget'
    },
    execute: () => {},
    reset: () => {},
    remove: () => { removed++ },
  }
  return { scripts, api, unmount, createVerifier: useTurnstile, verifier: useTurnstile(), options: () => options, removed: () => removed }
}

const flush = () => new Promise(resolve => setImmediate(resolve))

test('a failed script download is retried with a fresh script', async () => {
  const state = await setup()
  const token = state.verifier.getToken()
  state.scripts[0].dispatchEvent(new Event('error'))
  await flush()
  assert.equal(state.scripts.length, 1)
  globalThis.window.turnstile = state.api
  state.scripts[0].dispatchEvent(new Event('load'))
  await flush()
  state.options().callback('fresh-token')
  assert.equal(await token, 'fresh-token')
})

test('temporary challenge errors retry before failing, without submitting an empty token', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const token = state.verifier.getToken()
  await flush()
  const options = state.options()
  assert.equal(options.retry, 'auto')
  options['error-callback']('600010')
  assert.equal(state.removed(), 0)
  options.callback('recovered-token')
  assert.equal(await token, 'recovered-token')
  assert.equal(state.removed(), 1)
})

test('repeated temporary challenge failures stop after three attempts', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const token = state.verifier.getToken()
  const rejected = assert.rejects(token, /600010/)
  await flush()
  for (let attempt = 0; attempt < 3; attempt++) state.options()['error-callback']('600010')
  await rejected
  assert.equal(state.removed(), 1)
})

test('invalid widget configuration fails immediately', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const token = state.verifier.getToken()
  const rejected = assert.rejects(token, /110200/)
  await flush()
  state.options()['error-callback']('110200')
  await rejected
  assert.equal(state.removed(), 1)
})

test('unsupported browsers settle verification with an actionable diagnostic', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const token = state.verifier.getToken()
  const rejected = assert.rejects(token, /browser unsupported/)
  await flush()
  state.options()['unsupported-callback']()
  await rejected
})

test('a new verification request removes the previous widget and rejects the old request', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const previous = state.verifier.getToken()
  const rejected = assert.rejects(previous, /superseded/)
  await flush()
  const current = state.verifier.getToken()
  await flush()
  await rejected
  assert.equal(state.removed(), 1)
  state.options().callback('current-token')
  assert.equal(await current, 'current-token')
  assert.equal(state.removed(), 2)
})

test('verification state belongs to each composable instance', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const first = state.verifier.getToken()
  await flush()
  const firstOptions = state.options()
  const second = state.createVerifier().getToken()
  await flush()
  firstOptions.callback('first-token')
  assert.equal(await first, 'first-token')
  state.options().callback('second-token')
  assert.equal(await second, 'second-token')
})

test('callbacks from a replaced challenge cannot settle its replacement', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const previous = state.verifier.getToken()
  const rejected = assert.rejects(previous, /superseded/)
  await flush()
  const stale = state.options()
  const current = state.verifier.getToken()
  await flush()
  await rejected
  stale.callback('stale-token')
  stale['error-callback']('110200')
  state.options().callback('current-token')
  assert.equal(await current, 'current-token')
})

test('cancelling a script download prevents a late widget mount', async () => {
  const state = await setup()
  const controller = new AbortController()
  const token = state.verifier.getToken(controller.signal)
  const rejected = assert.rejects(token, { name: 'AbortError' })
  controller.abort()
  await rejected
  globalThis.window.turnstile = state.api
  state.scripts[0].dispatchEvent(new Event('load'))
  await flush()
  assert.equal(state.options(), undefined)
})

test('unmount cancels the challenge and removes its widget', async () => {
  const state = await setup()
  globalThis.window.turnstile = state.api
  const token = state.verifier.getToken()
  const rejected = assert.rejects(token, { name: 'AbortError' })
  await flush()
  state.unmount[0]()
  await rejected
  assert.equal(state.removed(), 1)
  state.options().callback('late-token')
  assert.equal(state.removed(), 1)
})
