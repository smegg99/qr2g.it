// web/tests/curl.test.mjs
import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { CURL } from '../app/content/curl.ts'

test('the documented POST command sends valid JSON and expands the verification token', () => {
  const output = execFileSync('bash', ['-s'], {
    input: `curl() { printf '%s\\0' "$@"; }\n${CURL.create}\n`,
    env: { ...process.env, TURNSTILE_TOKEN: 'example-verification-token' },
  })
  const args = output.toString().split('\0')
  assert.equal(args[0], '--fail-with-body')
  assert.equal(args[2], 'https://qr2g.it/api/v1/links')
  assert.deepEqual(JSON.parse(args[args.indexOf('--data') + 1]), {
    url: 'https://github.com/smegg99/qr2g.it',
    turnstileToken: 'example-verification-token',
  })
})

test('README and website publish the same curl commands', () => {
  const readme = readFileSync(new URL('../../README.md', import.meta.url), 'utf8')
  for (const [name, command] of Object.entries(CURL)) {
    assert.ok(readme.includes(command), `README example is stale: ${name}`)
  }
})
