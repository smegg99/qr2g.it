# qr2g.it

Permanent short links and QR codes for public GitHub repositories. A link is tied to the repository's GitHub identity, so its slug survives a rename or transfer. The Go service serves the Nuxt/Vuetify interface and the HTTP API from one listener, with SQLite for persistent data.

- Paste `owner/repo`, `github.com/owner/repo`, or a full HTTPS GitHub URL. The result has two steps: the short link with its filtered clicks (total, 30 and 7 days, last click, a 30 day chart), GitHub stars and language stack, then the QR code. The page takes the stack's GitHub colours until the result is closed.
- Design a QR code, with the qr2g.it fox in the centre by default, and export it as SVG, PNG, or JPG. The server renders previews and downloads from one Go renderer, which the API also uses for presentation, poster, label, and badge presets.
- See how many public GitHub repositories have a link, and the three most visited listed repositories, each with a 30 day sparkline of filtered redirects, the total, and the latest click. The API also searches all listed repositories.
- The contact dialog reveals the address only after a browser check; an unverified visitor gets the catch-all address. Use it for questions and removal requests.
- Owners opt out with `.qr2git-noindex` or `.qr2git-noanalytics` files, or `<!-- qr2git: noindex -->` and `<!-- qr2git: noanalytics -->` README comments.
- Chats can link to a prefilled design, for example `https://qr2g.it/?repo=smegg99%2FGoptivum&fg=%230b3d91`. Opening it creates nothing.
- Use English or Polish and light, dark, or system appearance.
- Read the [API page](https://qr2g.it/docs), the [usage guide](https://qr2g.it/guide.md), the [OpenAPI contract](https://qr2g.it/api/v1/openapi.yaml), or the [interactive reference](https://qr2g.it/swagger/index.html).
- Author: [smegg99 on LinkedIn](https://linkedin.smegg.me).

## Requirements

- Go 1.27, including CUE and oapi-codegen pinned in `go.mod` and S99Locale pinned in `tools/locales/go.mod`.
- Node.js 22.19+, 24.11+, or 26+ and pnpm 10.31.0, selected by `web/package.json`.
- A writable directory for SQLite. No PostgreSQL server is required.
- Python 3 for the generated-file checks and PostgreSQL dump importer.
- `just` for the command recipes. `sqlite3` is optional for `just db-shell`.

The authored API contract lives in `contracts/backend/`. Redocly (pinned in `contracts/package.json`) bundles it; oapi-codegen generates the Gin server boundary in `api/v1/gen/` and Hey API (pinned in `web/package.json`) generates the TypeScript client in `web/app/generated/api/`. `just generate` regenerates them; `just check-generated` fails when any output is stale. Shared error and pagination fragments are copied from the pinned S99AB module by `just shared-contracts`.

S99AB generates the frontend theme from an accent seed saved in each browser. Its package archive is bundled under `web/vendor/` for deployment builds.

## Local development

```sh
cp .env.example .env
just install
just dev
```

The API listens at `127.0.0.1:9245` and Nuxt at `http://127.0.0.1:5173`. Nuxt proxies `/api` to the backend. `config.json.dev` stores data at `data/qr2git.sqlite`. The example environment and Nuxt development fallback use Cloudflare testing keys. Use real keys in production.

The binary has commands: `qr2git serve` runs the service, `qr2git token create --name NAME --quota N` issues an automation token and prints its secret once, `qr2git token list` and `qr2git token revoke ID` manage tokens, and `qr2git moderate unlist|remove|clear SLUG` records removal decisions (`moderate list` shows them). `unlist` hides a repository from public surfaces; `remove` also stops its redirect. The slug is never reused.

`CONFIG_PATH` selects a different configuration file. The CUE schema in `schema/config.cue` owns its shape and defaults; `just gen-types` regenerates the Go types. The default `config.json` binds to loopback port 8080, while the S99Deploy config uses port 9260.

## Verification and builds

```sh
just check
just build
just release
```

`just check` runs Go formatting, vet, backend tests, frontend tests, lint, typecheck, locale drift and source-key checks, config generation checks, shared-package verification, deployment checks, and the complete build. `just test-race` runs backend tests with the race detector.

`VERSION` sets the release version. `just build` builds the frontend from the committed locale bundles, embeds it, and produces `bin/qr2git`. `just release` packages that binary, configuration examples, environment example, version, README, and license into `qr2git-v<VERSION>-linux-amd64.tar.gz`.

Language colours come from a pinned GitHub Linguist release; `just language-colors` regenerates `web/app/generated/languageColors.ts` and needs network access and PyYAML.

English and Polish copy is authored in `i18n/locales/`. S99Locale generates the committed bundles under `web/i18n/generated/`. Change YAML, run `just locales`, and check with `just check-locales`. The private S99Locale module is a development dependency: a cold tool cache needs authenticated access to its Git repository. Production builds consume the committed generated bundles and do not run or download that compiler. Do not edit generated bundles or config types directly.

## curl API

### Get or create a short link

Replace `smegg99/qr2g.it` with a public `owner/repo`:

```sh
curl -fsS https://qr2g.it/smegg99/qr2g.it
```

Example response:

```text
https://qr2g.it/2
```

Add `?qr` to draw the QR code above the link with Unicode blocks, for a dark terminal; `?qr=invert` suits a light one. Quote the URL so the shell does not expand `?`:

```sh
curl -fsS "https://qr2g.it/smegg99/qr2g.it?qr"
```

This shorthand is not a pure read. Known repositories return their existing address; an unknown public repository is checked on GitHub and receives a new link. The response is a plain-text URL with a trailing newline. curl is recognized by its user agent; other clients can explicitly request text:

```sh
curl -fsS -H "Accept: text/plain" https://qr2g.it/smegg99/qr2g.it
```

`?format=text` and `?text=1` also request text. A browser normally receives a redirect to the short link instead. Public slugs are uppercase base36, case-sensitive, and never recycled.

### Read without creating

```sh
curl --fail-with-body -sS "https://qr2g.it/api/v1/links/lookup?repository=smegg99/qr2g.it"
curl -fsS https://qr2g.it/api/v1/links/2
curl -fsS https://qr2g.it/2+
```

None of these creates a link or calls GitHub. A link keeps the shipped fields `url`, `slug`, `id`, `state`, `createdAt` (now the start of its UTC day), and, while the repository is publicly available, `initialFullName`, `currentFullName`, `currentUrl`, `verifiedAt`, `verifyAfter`, `githubNodeId`, and `githubDatabaseId`. New fields are `description`, `language`, `archived`, and the shared `policy`; description and language are withheld while the owner or operator hides the repository from discovery. States are `active`, `unavailable` (GitHub no longer shows it publicly; deleted and private look the same, and stored `deleted` rows became `unavailable`), `private`, and `removed`.

### Verify and resolve

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/resolve
```

The resolve endpoint re-verifies stale repositories before answering. If verification fails, it returns an error instead of trusting an outdated destination. Links that no longer redirect return HTTP 410; an unavailable repository that becomes public again recovers the same slug. To inspect the normal redirect without following it:

```sh
curl -sS -D - -o /dev/null https://qr2g.it/2
```

Add `-L` when curl should follow redirects.

### Submit JSON

Browsers send a fresh Turnstile token. Set `TURNSTILE_TOKEN` to that single-use token.

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H 'Content-Type: application/json' \
  --data "{\"url\":\"https://github.com/smegg99/qr2g.it\",\"turnstileToken\":\"$TURNSTILE_TOKEN\"}"
```

The repository URL must be HTTPS on `github.com`, with no credentials, query, fragment, port, or extra path components. HTTP 201 means this request created the link; 200 means it already existed. Repeating the request returns the same link.

### Automation

Scripts and coding agents use an operator-issued token instead of Turnstile:

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"url":"https://github.com/smegg99/qr2g.it"}'
```

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links/batch \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"repositories":["smegg99/qr2g.it","smegg99/Goptivum"]}'
```

```sh
curl --fail-with-body -sS -H "Authorization: Bearer $QR2GIT_TOKEN" https://qr2g.it/api/v1/automation/usage
```

A batch takes up to 25 repositories and answers each with `created`, `existing`, or `failed` plus its error. Token responses carry `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset`; a used-up quota answers 429 `quota_exceeded` with `Retry-After`.

### Render files

```sh
curl --fail-with-body -sS -OJ "https://qr2g.it/api/v1/links/2/asset?format=png&download=true"
```

The default `qr` preset draws the qr2g.it logo in the centre (`logo=false` leaves it out; the logo needs error correction `H`). The API also accepts `presentation`, `poster`, `label`, and `badge`; the [usage guide](https://qr2g.it/guide.md) lists the options. Files are named after the repository and preset, for example `Goptivum-qr.png`. Every code has a four-module quiet zone unless `margin` sets 0 to 16, and needs a contrast ratio of at least 4.5 between modules and tile; light-on-dark codes are allowed. A rendered file does not prove that every printer or camera reads it.

### Listed repositories and statistics

```sh
curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories?q=timetable&sort=popular30&page=1&limit=24"
```

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/stats
```

```sh
curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories/top?limit=3"
```

The home page shows the three most visited listed repositories from `/repositories/top`, with 30 days of daily counts, the total, and the time of the latest counted redirect.

Only listed repositories appear: publicly available, checked for owner markers, not opted out, and not removed. Statistics count filtered redirects per UTC day: successful final redirects excluding HEAD requests, identifiable bots, link previews, prefetches, and authenticated API traffic. They are not unique people or QR scans, and days before collection began are unknown.

### Limits and errors

The deployment defaults apply per IP:

- Creating an unknown repository through shorthand: 5 per hour. Known-repository lookups do not consume this allowance.
- POST: 3 requests per minute and 50 submissions per day after URL validation.
- Slug resolution: 600 requests per minute. Reads: 120 per minute. Assets: 600 per minute, cacheable for an hour. Refresh repository settings: 20 per hour per IP and once per 5 minutes per link.

Errors answer the S99AB envelope `{"code": "...", "message": "errors...."}` (earlier releases used `error` for the code): branch on `code`; `message` is an i18n key. Rate-limited responses, including 503 `github_rate_limited`, carry `Retry-After`. `--fail-with-body` requires curl 7.76.0 or newer.

| HTTP | Meaning |
| --- | --- |
| 200 | Existing link, metadata, or successful read |
| 201 | New link created |
| 302 | Redirect |
| 400 | Invalid JSON, repository, or design option |
| 401 | Automation token required, unknown, or revoked |
| 403 | Private repository or missing/failed Turnstile verification |
| 404 | Repository, link, or listing not found |
| 410 | Link is no longer redirecting |
| 429 | Request limit or quota; see `Retry-After` |
| 503 | GitHub, Turnstile, contact configuration, or identity verification unavailable |

`GET /health` is liveness and `GET /version` reports the build. `GET /ready` checks SQLite connectivity. `GET /{slug}+` reads metadata; `GET /{slug}` follows the verified repository destination.

## VPS deployment

Install the current S99Deploy CLI as `/usr/local/bin/depl` on the target host. To install the service:

```sh
sudo depl install https://github.com/smegg99/qr2g.it.git
sudoedit /opt/qr2git/.env
sudo depl up qr2git
```

Set real `TURNSTILE_SECRET_KEY` and `TURNSTILE_SITE_KEY` values in the root-owned, mode-0600 environment file. Optional: `GITHUB_TOKEN` (a token without repository scopes) raises GitHub's hourly allowance for availability and owner-policy checks; anonymous reads allow 60 requests per hour, and background checks pause when fewer than `github.background_reserve` remain. `CONTACT_EMAIL` and `CONTACT_CATCHALL_EMAIL` are the addresses the contact dialog reveals; neither reaches the site bundle. A Managed Turnstile widget lets a visitor complete a visible challenge when needed. The backend injects the public site key into embedded HTML at runtime.

`deploy.json` builds from the checkout, runs as the dedicated `qr2git` identity, loads `config.json.deploy`, and checks `/ready` on `127.0.0.1:9260`. A TLS reverse proxy must replace forwarded client-IP headers with the verified visitor address. Direct origin clients must not be allowed to supply trusted visitor addresses.

The production database is `/opt/qr2git/home/data/qr2git.sqlite`, outside the checkout. Restart with `sudo systemctl restart qr2git` and read logs with `sudo journalctl -u qr2git`. Updating code must preserve the database and service environment.

## Import and backup

Download the full production database through the `ovh` SSH alias:

```sh
just db-download
sqlitebrowser data/qr2git-production.sqlite
```

This uses SQLite's online backup API, includes committed WAL data, and checks integrity before saving. It requires Python 3 locally and on the VPS, plus passwordless sudo through SSH. To choose another destination or host, use `just db-download /path/to/snapshot.sqlite ovh`. Close the database preview before refreshing the snapshot.

Stop the destination service before importing. Python 3 and a `pg_restore` version capable of reading the dump are required:

```sh
python3 scripts/import-postgres.py /path/to/kopia_bazy.dump /path/to/new-qr2git.sqlite
```

The importer refuses to overwrite an existing file. It preserves link IDs, aliases, nullable fields, states, timestamp instants (normalized to UTC), and the PostgreSQL sequence high-water mark. It compares every imported field and checks foreign keys and SQLite integrity. Give the service account ownership of the imported file and point `db.path` at it before starting.

For a live SQLite backup, use the backup API rather than copying a database without its WAL:

```sh
sqlite3 /path/to/qr2git.sqlite ".backup '/path/to/backup.sqlite'"
```

## License

[MIT](LICENSE).
