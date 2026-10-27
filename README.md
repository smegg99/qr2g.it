# qr2g.it

Permanent short links and QR codes for public GitHub repositories. A link is tied to the repository's GitHub identity, so its slug survives a rename or transfer. The Go service serves the Nuxt/Vuetify interface and the HTTP API from one listener, with SQLite for persistent data.

- Paste `owner/repo`, `github.com/owner/repo`, or a full HTTPS GitHub URL.
- Copy the short link, or customize and export a QR code as SVG, PNG, or JPG.
- Use English or Polish and light, dark, or system appearance.
- Read the [curl reference](https://qr2g.it/docs) or the [interactive JSON reference](https://qr2g.it/swagger/index.html).
- Author: [smegg99 on LinkedIn](https://linkedin.smegg.me).

## Requirements

- Go 1.27, including CUE and Swagger pinned in `go.mod` and S99Locale pinned in `tools/locales/go.mod`.
- Node.js 22.19+, 24.11+, or 26+ and pnpm 10.31.0, selected by `web/package.json`.
- A writable directory for SQLite. No PostgreSQL server is required.
- Python 3 for the generated-file checks and PostgreSQL dump importer.
- `just` for the command recipes. `sqlite3` is optional for `just db-shell`.

S99AB generates the frontend theme from an accent seed saved in each browser. Its package archive is bundled under `web/vendor/` for deployment builds.

## Local development

```sh
cp .env.example .env
just install
just dev
```

The API listens at `127.0.0.1:9245` and Nuxt at `http://127.0.0.1:5173`. Nuxt proxies `/api` to the backend. `config.json.dev` stores data at `data/qr2git.sqlite`. The example environment and Nuxt development fallback use Cloudflare testing keys. Use real keys in production.

`CONFIG_PATH` selects a different configuration file. The CUE schema in `schema/config.cue` owns its shape and defaults; `just gen-types` regenerates the Go types. The default `config.json` binds to loopback port 8080, while the S99Deploy config uses port 9260.

## Verification and builds

```sh
just check
just build
just release
```

`just check` runs Go formatting, vet, backend tests, frontend tests, lint, typecheck, locale drift and source-key checks, config generation checks, shared-package verification, deployment checks, and the complete build. `just test-race` runs backend tests with the race detector.

`VERSION` sets the release version. `just build` builds the frontend from the committed locale bundles, embeds it, and produces `bin/qr2git`. `just release` packages that binary, configuration examples, environment example, version, README, and license into `qr2git-v<VERSION>-linux-amd64.tar.gz`.

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

Known repositories return their existing address. New public repositories are checked in GitHub and receive a new slug. The response is a plain-text URL with a trailing newline. curl is recognized by its user agent; other clients can explicitly request text:

```sh
curl -fsS -H "Accept: text/plain" https://qr2g.it/smegg99/qr2g.it
```

`?format=text` and `?text=1` also request text. A browser normally receives a redirect to the short link instead. Public slugs are uppercase base36, case-sensitive, and never recycled.

### Read stored metadata

```sh
curl -fsS https://qr2g.it/api/v1/links/2
curl -fsS https://qr2g.it/2+
```

Both return JSON without re-verifying in GitHub. Fields include `url`, `slug`, `id`, `initialFullName`, `currentFullName`, `currentUrl`, `state`, verification times, and the GitHub node/database IDs. States are `active`, `private`, `deleted`, and `unavailable`.

### Verify and resolve

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/resolve
```

The resolve endpoint re-verifies stale active repositories before returning their destination. If verification fails, the endpoint returns an error. Non-active links return HTTP 410. To inspect the normal redirect without following it:

```sh
curl -sS -D - -o /dev/null https://qr2g.it/2
```

Add `-L` when curl should follow redirects.

### Submit JSON

POST requests require a fresh Turnstile token from a browser challenge. Set `TURNSTILE_TOKEN` to that single-use token.

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H 'Content-Type: application/json' \
  --data "{\"url\":\"https://github.com/smegg99/qr2g.it\",\"turnstileToken\":\"$TURNSTILE_TOKEN\"}"
```

The repository URL must be HTTPS on `github.com`, with no credentials, query, fragment, port, or extra path components. Successful submissions return JSON and HTTP 201 for a new link or 200 for an existing one.

### Limits and errors

The deployment defaults apply per IP:

- Creating an unknown repository through shorthand: 5 per hour. Known-repository lookups do not consume this allowance.
- POST: 3 requests per minute and 50 submissions per day after URL validation. Failed verification consumes that daily allowance.
- Slug resolution: 600 requests per minute.

Errors contain a stable `error` code and a `message`. `--fail-with-body` requires curl 7.76.0 or newer and retains the error response while returning a failing exit code.

| HTTP | Meaning |
| --- | --- |
| 200 | Existing link, metadata, or successful resolution |
| 201 | New link created |
| 302 | Redirect |
| 400 | Invalid JSON or repository URL |
| 403 | Private repository or missing/failed Turnstile verification |
| 404 | Repository or slug not found |
| 410 | Link is no longer active |
| 429 | Per-IP request limit |
| 503 | GitHub, Turnstile, or identity verification unavailable |

`GET /health` is liveness. `GET /ready` checks SQLite connectivity. `GET /{slug}+` reads metadata; `GET /{slug}` follows the verified repository destination.

## VPS deployment

Install the current S99Deploy CLI as `/usr/local/bin/depl` on the target host. To install the service:

```sh
sudo depl install https://github.com/smegg99/qr2g.it.git
sudoedit /opt/qr2git/.env
sudo depl up qr2git
```

Set real `TURNSTILE_SECRET_KEY` and `TURNSTILE_SITE_KEY` values in the root-owned, mode-0600 environment file. A Managed Turnstile widget lets a visitor complete a visible challenge when needed. The backend injects the public site key into embedded HTML at runtime.

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
