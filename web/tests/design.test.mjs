// web/tests/design.test.mjs
import assert from 'node:assert/strict'
import { test } from 'node:test'
import { DEFAULT_DESIGN, assetFilename, assetQuery, designLink, formatsFor, importDesign } from '../app/utils/design.ts'

test('a prefilled link round-trips repository and every QR choice', () => {
  const design = { ...DEFAULT_DESIGN, fg: '#0b3d91', bg: '#fffaf0', logo: false, ec: 'L', margin: 6, radius: 40, size: 2048, format: 'svg' }
  const url = designLink('https://qr2g.it/', 'smegg99/Goptivum', design)
  assert.match(url, /^https:\/\/qr2g\.it\/\?repo=smegg99%2FGoptivum&/)
  const back = importDesign(new URL(url).searchParams)
  assert.equal(back.repo, 'smegg99/Goptivum')
  assert.deepEqual(back.problems, [])
  assert.deepEqual(back.design, design)
  assert.equal(designLink('https://qr2g.it', 'a/b', DEFAULT_DESIGN), 'https://qr2g.it/?repo=a%2Fb')
})

test('invalid, oversized and unknown parameters are reported and ignored', () => {
  const params = new URLSearchParams({
    preset: 'presentation', fg: 'red', margin: '17', size: '99999', token: 'secret',
    redirect: 'https://evil.example', ec: 'h', logo: 'yes', repo: 'a/b',
  })
  const back = importDesign(params)
  assert.equal(back.design.fg, DEFAULT_DESIGN.fg)
  assert.equal(back.design.margin, 4)
  assert.equal(back.design.size, 1024)
  assert.equal(back.design.ec, 'H')
  assert.equal(back.design.logo, true)
  const reasons = Object.fromEntries(back.problems.map(p => [p.param, p.reason]))
  assert.deepEqual(reasons, {
    preset: 'unknown', fg: 'invalid', margin: 'out_of_range', size: 'out_of_range', logo: 'invalid',
    token: 'unknown', redirect: 'unknown',
  })
})

test('a quiet zone can go down to zero', () => {
  const back = importDesign(new URLSearchParams('margin=0'))
  assert.deepEqual(back.problems, [])
  assert.equal(back.design.margin, 0)
  assert.deepEqual(assetQuery(back.design), { format: 'png', margin: '0' })
})

test('the logo keeps error correction at H', () => {
  const back = importDesign(new URLSearchParams('ec=M'))
  assert.equal(back.design.ec, 'H')
  assert.deepEqual(back.problems, [{ param: 'ec', reason: 'invalid' }])
  assert.equal(importDesign(new URLSearchParams('ec=Q')).design.ec, 'H')
  assert.deepEqual(importDesign(new URLSearchParams('ec=L&logo=false')).problems, [])
})

test('asset queries drop server defaults', () => {
  assert.deepEqual(assetQuery(DEFAULT_DESIGN), { format: 'png' })
  assert.deepEqual(assetQuery({ ...DEFAULT_DESIGN, logo: false, ec: 'M' }, 'svg'), { format: 'svg', logo: 'false' })
  assert.deepEqual(assetQuery({ ...DEFAULT_DESIGN, logo: false }), { format: 'png', logo: 'false', ec: 'H' })
  assert.deepEqual(assetQuery({ ...DEFAULT_DESIGN, transparent: true, radius: 50 }), { format: 'png', transparent: 'true' })
  assert.deepEqual(formatsFor({ ...DEFAULT_DESIGN, transparent: true }), ['svg', 'png'])
})

test('filenames match the server', () => {
  assert.equal(assetFilename('Goptivum', 'png'), 'Goptivum-qr.png')
  assert.equal(assetFilename('../ąę name', 'svg'), 'name-qr.svg')
})
