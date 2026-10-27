// web/tests/bootstrap.test.mjs
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'
import { restoreTheme } from '../app/theme/bootstrap.ts'

const grounds = { light: '#eeeeee', dark: '#111111' }
const keys = { settings: 'qr2g.settings', accent: 'qr2g.accent', paint: 'qr2g.paint' }
const script = `(${restoreTheme.toString()})(${JSON.stringify(grounds)}, ${JSON.stringify(keys)})`

function paint({ cookie = '', dark = false, storage = {} } = {}) {
  const style = {}
  runInNewContext(script, {
    document: { cookie, documentElement: { style } },
    matchMedia: () => ({ matches: dark }),
    localStorage: { getItem: key => storage[key] ?? null },
  })
  return style
}

test('first paint follows the system until the visitor chooses a theme', () => {
  assert.deepEqual(paint({ dark: true }), { backgroundColor: grounds.dark, colorScheme: 'dark' })
  const cookie = `qr2g.settings=${encodeURIComponent(JSON.stringify({ themeMode: 'light' }))}`
  assert.deepEqual(paint({ dark: true, cookie }), { backgroundColor: grounds.light, colorScheme: 'light' })
})

test('saved paint is used only for its matching visitor seed', () => {
  const saved = { seed: '12345678', light: '#abcdef', dark: '#123456' }
  const storage = { [keys.paint]: JSON.stringify(saved), [keys.accent]: saved.seed }
  assert.equal(paint({ dark: true, storage }).backgroundColor, saved.dark)
  storage[keys.accent] = '87654321'
  assert.equal(paint({ dark: true, storage }).backgroundColor, grounds.dark)
})

test('invalid preferences and paint do not prevent the initial background', () => {
  const cookie = 'qr2g.settings=broken'
  assert.equal(paint({ cookie, dark: true }).backgroundColor, grounds.dark)
  assert.equal(paint({ storage: { [keys.paint]: 'broken' } }).backgroundColor, grounds.light)
})

test('blocked storage still applies the system theme', () => {
  const storage = new Proxy({}, {
    get() {
      throw new Error('blocked')
    },
  })
  assert.deepEqual(paint({ dark: true, storage }), { backgroundColor: grounds.dark, colorScheme: 'dark' })
})
