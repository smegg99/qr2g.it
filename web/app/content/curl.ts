// app/content/curl.ts
export const CURL = {
  short: 'curl -fsS https://qr2g.it/smegg99/qr2g.it',
  qr: 'curl -fsS "https://qr2g.it/smegg99/qr2g.it?qr"',
  text: 'curl -fsS -H "Accept: text/plain" https://qr2g.it/smegg99/qr2g.it',
  lookup: 'curl --fail-with-body -sS "https://qr2g.it/api/v1/links/lookup?repository=smegg99/qr2g.it"',
  metadata: 'curl -fsS https://qr2g.it/api/v1/links/2',
  info: 'curl -fsS https://qr2g.it/2+',
  resolve: 'curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/resolve',
  redirect: 'curl -sS -D - -o /dev/null https://qr2g.it/2',
  create: String.raw`curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H 'Content-Type: application/json' \
  --data "{\"url\":\"https://github.com/smegg99/qr2g.it\",\"turnstileToken\":\"$TURNSTILE_TOKEN\"}"`,
  automation: String.raw`curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"url":"https://github.com/smegg99/qr2g.it"}'`,
  batch: String.raw`curl --fail-with-body -sS https://qr2g.it/api/v1/links/batch \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"repositories":["smegg99/qr2g.it","smegg99/Goptivum"]}'`,
  asset: 'curl --fail-with-body -sS -OJ "https://qr2g.it/api/v1/links/2/asset?format=png&download=true"',
  search: 'curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories?q=timetable&sort=popular30&page=1&limit=24"',
  stats: 'curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/stats',
  top: 'curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories/top?limit=3"',
  usage: 'curl --fail-with-body -sS -H "Authorization: Bearer $QR2GIT_TOKEN" https://qr2g.it/api/v1/automation/usage',
} as const

export const DESIGN_LINK_EXAMPLE = 'https://qr2g.it/?repo=smegg99%2FGoptivum&fg=%230b3d91'
