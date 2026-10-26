// app/content/curl.ts
export const CURL = {
  short: 'curl -fsS https://qr2g.it/smegg99/qr2g.it',
  text: 'curl -fsS -H "Accept: text/plain" https://qr2g.it/smegg99/qr2g.it',
  metadata: 'curl -fsS https://qr2g.it/api/v1/links/2',
  info: 'curl -fsS https://qr2g.it/2+',
  resolve: 'curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/resolve',
  redirect: 'curl -sS -D - -o /dev/null https://qr2g.it/2',
  create: String.raw`curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H 'Content-Type: application/json' \
  --data "{\"url\":\"https://github.com/smegg99/qr2g.it\",\"turnstileToken\":\"$TURNSTILE_TOKEN\"}"`,
} as const
