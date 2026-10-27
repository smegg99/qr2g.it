// web/tests/guide.test.mjs
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { CURL, DESIGN_LINK_EXAMPLE } from '../app/content/curl.ts'
import { importDesign } from '../app/utils/design.ts'

const read = path => readFileSync(new URL(path, import.meta.url), 'utf8')
const bundle = read('../../contracts/backend/dist/openapi.yaml')
const guide = read('../public/guide.md')
const llms = read('../public/llms.txt')

// Contract paths as patterns, e.g. /links/{slug}/asset -> ^/links/[^/?]+/asset$.
const paths = [...bundle.matchAll(/^ {2}(\/[^:\s]*):$/gm)].map(m => new RegExp(`^${m[1].replace(/\{[^}]+\}/g, '[^/?]+')}$`))

function apiPaths(text) {
  return [...text.matchAll(/qr2g\.it\/api\/v1(\/[A-Za-z0-9/_{}.-]*)/g)].map(m => m[1])
}

test('every documented API path exists in the contract', () => {
  const documented = [...apiPaths(guide), ...apiPaths(Object.values(CURL).join('\n'))]
    .filter(p => p !== '/openapi.yaml')
  assert.ok(documented.length > 8)
  for (const path of documented) {
    assert.ok(paths.some(re => re.test(path)), `not in the contract: ${path}`)
  }
})

test('the guide names every asset parameter, and every design-link parameter imports', () => {
  const asset = bundle.slice(bundle.indexOf('operationId: renderLinkAsset'))
  const section = asset.slice(0, asset.indexOf('responses:'))
  const contract = new Set([...section.matchAll(/- name: (\w+)/g)].map(m => m[1]))
  contract.delete('slug')
  contract.delete('download')
  const named = new Set([...guide.matchAll(/`([a-z][A-Za-z]+)`/g)].map(m => m[1]))
  for (const name of contract) assert.ok(named.has(name), `guide misses ${name}`)
  const table = guide.slice(guide.indexOf('| Parameter |'), guide.indexOf('The same names'))
  const linkParams = [...table.matchAll(/^\| ((?:`\w+`(?:, )?)+) \|/gm)].flatMap(m => [...m[1].matchAll(/`(\w+)`/g)].map(n => n[1]))
  assert.ok(linkParams.includes('logo') && linkParams.includes('repo'))
  for (const name of linkParams.filter(n => n !== 'repo')) {
    assert.ok(contract.has(name), `not an asset parameter: ${name}`)
    assert.deepEqual(importDesign(new URLSearchParams({ [name]: 'x' })).problems.map(p => p.reason).filter(r => r === 'unknown'), [], name)
  }
})

test('the documented design links import cleanly and llms.txt points at the guide', () => {
  for (const link of [DESIGN_LINK_EXAMPLE, ...guide.matchAll(/https:\/\/qr2g\.it\/\?[^\s`]+/g)].map(String)) {
    const imported = importDesign(new URL(link).searchParams)
    assert.deepEqual(imported.problems, [], link)
    assert.equal(imported.repo, 'smegg99/Goptivum')
  }
  assert.match(llms, /\(https:\/\/qr2g\.it\/guide\.md\)/)
})
