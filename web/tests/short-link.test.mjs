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
