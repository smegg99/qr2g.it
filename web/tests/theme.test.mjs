// web/tests/theme.test.mjs
import assert from 'node:assert/strict'
import { test, mock } from 'node:test'
import { ACCENT_KEY, visitorSeed } from '../app/theme/visitor.ts'

function storage() {
  const values = new Map()
  return { getItem: key => values.get(key) ?? null, setItem: (key, value) => values.set(key, value) }
}

test('a visitor keeps the same accent across visits', () => {
  globalThis.localStorage = storage()
  const first = visitorSeed()
  assert.match(first, /^[0-9a-f]{8}$/)
  assert.equal(visitorSeed(), first)
})

test('corrupted stored seeds are replaced', () => {
  globalThis.localStorage = storage()
  localStorage.setItem(ACCENT_KEY, 'broken')
  const seed = visitorSeed()
  assert.match(seed, /^[0-9a-f]{8}$/)
  assert.equal(localStorage.getItem(ACCENT_KEY), seed)
})

test('blocked storage still produces a usable accent', () => {
  globalThis.localStorage = {
    getItem() { throw new Error('blocked') },
    setItem() { throw new Error('blocked') },
  }
  assert.match(visitorSeed(), /^[0-9a-f]{8}$/)
})

test('separate browser profiles receive independent seeds', () => {
  let value = 0x12345677
  const random = mock.method(crypto, 'getRandomValues', (values) => {
    values[0] = ++value
    return values
  })
  try {
    globalThis.localStorage = storage()
    const first = visitorSeed()
    globalThis.localStorage = storage()
    assert.notEqual(visitorSeed(), first)
  }
  finally {
    random.mock.restore()
  }
})
