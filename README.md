<p align="center">
  <img src="assets/qr2g.it.svg" alt="qr2g.it" width="500">
</p>

qr2g.it is a small site for permanent, QR-friendly short links to public GitHub repositories. Paste `owner/repo`, `github.com/owner/repo`, or a full GitHub repository URL, and the app returns a short link plus a customizable QR code.

Links are bound to GitHub's stable repository identity (`node_id` and database id), not just the current `owner/repo` text. If a repository is renamed or transferred later, qr2g.it can verify the identity again and keep the same slug pointing at the same repo.

## What It Does

- Runs a Nuxt/Vuetify single-page site for creating short links and exporting QR codes.
- Accepts `owner/repo`, `github.com/owner/repo`, and `https://github.com/owner/repo` in the web UI.
- Creates or reuses a permanent short link for public GitHub repositories through `POST /api/v1/links`.
- Generates QR codes in the browser with foreground/background colors, error correction, size, margin, and corner radius controls.
- Exports QR codes as SVG, PNG, or JPG using filenames based on the short link.
- Supports English and Polish UI text, plus persisted light/dark/auto theme preferences.
- Keeps the CLI shortcut: `curl qr2g.it/owner/repo` prints the short link as plain text.
- Redirects `/{SLUG}` to the current verified GitHub repository URL.
- Exposes JSON metadata at `/{SLUG}+` and `/api/v1/links/:slug`.
- Tracks GitHub `node_id` aliases so migrated GitHub identities still map back to the original link.
- Runs a background verifier that re-checks stale links and tombstones deleted or private repositories after repeated failures.
- Applies per-IP create/resolve limits and GitHub-facing rate limits, with a small circuit breaker around GitHub calls.

## Usage

### Website

Start from the homepage, paste a public GitHub repository, and generate the link. The result screen shows the short URL, the current repository state, open/copy actions, and the QR editor.

The web app accepts:

- `smegg99/ThugHunter`
- `github.com/smegg99/ThugHunter`
- `https://github.com/smegg99/ThugHunter`

### CLI

The shorthand route still works nicely from a terminal:

```sh
curl https://qr2g.it/smegg99/ThugHunter
# https://qr2g.it/AB12
```

`curl`, `wget`, `HTTPie`, `python-requests`, Go's default HTTP client, and PowerShell get a `text/plain` response with a trailing newline. Browsers get a redirect chain instead:

```txt
https://qr2g.it/smegg99/ThugHunter
https://qr2g.it/AB12
https://github.com/smegg99/ThugHunter
```

Force plain text from any client with `?format=text`, `?text=1`, or `Accept: text/plain`.

## Prerequisites

### Backend

- Go matching the directive in `go.mod` (currently `1.26.2`)
- PostgreSQL reachable through `DATABASE_URL`
- Optional: `just` for the recipes in `justfile`
- Optional but recommended: a GitHub App installation for authenticated REST requests and strict GraphQL verification by `node_id`
- Cloudflare Turnstile keys for public link creation, unless `turnstile.enabled` is set to `false`

### Frontend

- Node.js
- pnpm

### Optional Development Tools

- `cue` for regenerating Go config types
- `sqlc` for generating repository query code if needed
- `swag` for regenerating the OpenAPI files under `docs/`
- `psql` for the database recipes

## Setup

### 1. Clone the repository

```sh
git clone https://github.com/smegg99/qr2g.it
cd qr2g.it
```

### 2. Configure the backend

```sh
cp .env.example .env
```

At minimum, set a local Postgres URL:

```sh
DATABASE_URL=postgres://qr2g:qr2g@localhost:5432/qr2g?sslmode=disable
```

The checked-in `config.json` is production-oriented: it binds the backend to `127.0.0.1:8080`, assumes nginx or another reverse proxy handles public HTTPS, and enforces the production Turnstile hostname. For local testing, use `config.json.dev`, which listens on `:9245` and matches the Nuxt dev proxy.

No GitHub credentials are required. Repository verification uses GitHub's public REST API for public repositories.

For Turnstile-protected link creation, set:

```sh
TURNSTILE_SECRET_KEY=...
TURNSTILE_SITE_KEY=...
```

The example file uses Cloudflare's always-passing invisible-widget test keys for local development. Dummy testing-key responses bypass action and hostname checks; real Turnstile secrets still enforce the configured checks. Do not build or deploy production releases with those keys. When `turnstile.enabled` is `true`, the backend refuses to start without `TURNSTILE_SECRET_KEY`; set `turnstile.enabled` to `false` only for an intentionally unprotected deployment.

### 3. Start the backend

```sh
just run-dev
# or
CONFIG_PATH=config.json.dev go run ./
```

On startup the backend loads `.env`, validates `config.json` against `schema/config.cue`, opens the Postgres pool, applies embedded migrations when `db.auto_migrate` is enabled, initializes GitHub and Turnstile clients, starts the verifier, and serves the API.

### 4. Start the site

```sh
cd web
pnpm install
pnpm dev
```

Open `http://localhost:5173`. In development, Nuxt proxies `/api` to `QR2G_BACKEND_URL` or `http://127.0.0.1:9245`.

## Development

Backend:

```sh
just test
just test-race
just vet
just fmt
```

Frontend:

```sh
cd web
pnpm lint
pnpm lint:fix
pnpm build
pnpm preview
```

Generated code and specs:

```sh
just gen-types    # CUE schema -> Go config types
just gen-sqlc     # SQL query package, if needed
just gen-swagger  # OpenAPI docs under ./docs
```

Database helpers:

```sh
just db-shell
just migrate
just db-reset     # destructive, local dev only
```

## Building And Release

Build the backend binary into `bin/qr2g`:

```sh
just build
```

Build the static frontend during development:

```sh
cd web
pnpm build
```

Create the release archive:

```sh
just release
```

That command installs frontend dependencies with the lockfile, generates the Nuxt static output, copies it into the Go embed directory, builds a Linux x86-64 binary, and writes:

```txt
qr2git-V1.0.0-x86-64.tar.gz
```

The archive contains `qr2git`, `config.json`, `config.json.dev`, `.env.example`, `README.md`, `LICENSE`, `qr2git.service.template`, and `install-systemd.sh`. The frontend is embedded inside the Go binary, so running `./qr2git` serves both the SPA and `/api/v1` from the same listener. The backend injects `turnstile.site_key` into the embedded HTML at runtime, so set `TURNSTILE_SITE_KEY` on the deployed host rather than baking it into the release.

Install the systemd service after extracting the archive:

```sh
cp .env.example .env
$EDITOR .env
$EDITOR config.json
sudo ./install-systemd.sh
sudo systemctl start qr2git
```

The installer only writes `/etc/systemd/system/qr2git.service`, reloads systemd, and enables the service. It does not move files or create users. The service runs `qr2git` from the extracted release directory, uses `config.json` from that directory, and loads environment variables from `.env` in that same directory. Override the unit name if needed:

```sh
sudo SERVICE_NAME=qr2git-prod ./install-systemd.sh
```

## API

Swagger UI is served at `/swagger/index.html` while the backend is running. The OpenAPI spec is committed under `docs/`.

### Create a Link

```sh
curl -s -X POST http://localhost:9245/api/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/owner/repo","turnstileToken":"TOKEN"}'
```

Example response:

```json
{
  "url": "HTTP://LOCALHOST:9245/ABC123",
  "slug": "ABC123",
  "id": 623698779,
  "initialFullName": "owner/repo",
  "currentFullName": "owner/repo",
  "currentUrl": "https://github.com/owner/repo",
  "state": "active",
  "verifiedAt": "2026-05-05T19:22:00Z",
  "verifyAfter": "2026-05-05T20:22:00Z",
  "createdAt": "2026-05-05T19:22:00Z",
  "githubNodeId": "R_kgDOExample",
  "githubDatabaseId": 123456789
}
```

Accepted backend URL forms are strict HTTPS GitHub repository URLs:

- `https://github.com/owner/repo`
- `https://github.com/owner/repo/`
- `https://github.com/owner/repo.git`

The backend rejects non-HTTPS URLs, non-`github.com` hosts, credentials, ports, query strings, fragments, extra path components, and reserved owner-level paths.

### Routes

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/` | Embedded frontend when present; otherwise JSON service metadata. |
| `GET` | `/health` | Liveness check. |
| `GET` | `/ready` | Readiness check; pings Postgres. |
| `POST` | `/api/v1/links` | Creates or reuses a link for a public GitHub repository URL. |
| `GET` | `/api/v1/links/:slug` | Returns stored link metadata without verifying. |
| `GET` | `/api/v1/links/:slug/resolve` | Resolves as JSON, verifying stale active links first. |
| `GET` | `/{SLUG}` | Redirects to the current GitHub URL, verifying stale links first. |
| `GET` | `/{SLUG}+` | Returns metadata for a slug without redirecting. |
| `GET` | `/{owner}/{repo}` | Finds or creates a link. CLI clients get text; browsers redirect to the slug. |
| `GET` | `/swagger/index.html` | Swagger UI for the JSON API. |

Slug rules live in `core/slug`: uppercase base36 only, one to twelve characters, no leading zero, no lowercase canonical form.

## Configuration

`config.json` is validated against `schema/config.cue`. String values can reference environment variables, files, keyring entries, the config directory, or the data directory with `@{...}` references. Private references (`@{env:...}`, `@{file:...}`, `@{keyring:...}`) are excluded from exported public config. Public references (`@{pubenv:...}`, `@{pubfile:...}`, `@{cfgdir:...}`, `@{datadir:...}`) are kept.

Add `?` after the source name to make a reference optional:

```json
{
  "db": {
    "url": "@{env?:DATABASE_URL}"
  }
}
```

Important options:

| Option | Schema default | Description |
|---|---|---|
| `server.listen_addr` | `:8080` | Backend listen address. The local `config.json` uses `:9245`. |
| `server.public_base_url` | `https://qr2g.it` | Lowercase public URL used for redirects and CLI text output. |
| `server.canonical_base_url` | `HTTPS://QR2G.IT` | Canonical short URL used in API responses and QR payloads. |
| `server.trusted_proxy_mode` | `cloudflare` | Client IP extraction mode: `cloudflare`, `none`, or `all`. |
| `server.require_shorthand_turnstile` | `false` | Requires Turnstile for cold `/{owner}/{repo}` creation. |
| `db.url` | `@{env?:DATABASE_URL}` | Postgres connection string. The service refuses to start if this resolves empty. |
| `db.auto_migrate` | `true` | Applies embedded migrations at startup. |
| `github.rest_base_url` | `https://api.github.com` | GitHub REST API base URL used for public repository lookups. |
| `turnstile.enabled` | `true` | Enables Turnstile checks on `POST /api/v1/links`. |
| `turnstile.expected_action` | `create_link` | Requires Siteverify responses to match the frontend action. |
| `turnstile.expected_hostname` | `""` | Optional exact hostname check for Siteverify responses; set to `qr2g.it` in production. |
| `verifier.enabled` | `true` | Runs the background stale-link verifier. |
| `verifier.fresh_ttl_seconds` | `3600` | How long an active link stays fresh before re-verification. |
| `verifier.max_fail_count` | `10` | Consecutive failures before a private/deleted repo is tombstoned. |
| `ratelimit.create_per_min` | `3` | Per-IP create rate limit. |
| `ratelimit.create_per_day` | `50` | Per-IP daily create cap. |
| `ratelimit.shorthand_per_hour` | `5` | Per-IP cold-create cap on `/{owner}/{repo}`. |
| `ratelimit.resolve_per_min` | `600` | Per-IP slug resolve limit. |

## Verification Behavior

Fresh active links redirect immediately to the cached GitHub URL. Stale active links are verified before redirecting. When GraphQL is available, verification resolves the repository by `node_id`; otherwise it falls back to REST lookup by GitHub database id.

Verification fails closed: a stale link is not redirected if GitHub verification fails, rate limits, or is unavailable. Private or deleted repositories are only tombstoned after `verifier.max_fail_count` consecutive failures. Tombstoned slugs are never recycled.

The background verifier polls in batches with `FOR UPDATE SKIP LOCKED`, so multiple backend instances can share the work safely.

## Screenshots

<div align="center">
  <table>
    <tr>
      <td align="center">
        <img src="assets/screenshots/screenshot_home_page.png" alt="qr2g.it home page" width="700"><br>
        <em>Home page with the repository input and quick actions.</em>
      </td>
    </tr>
    <tr>
      <td align="center">
        <img src="assets/screenshots/screenshot_qr_page.png" alt="qr2g.it qr page" width="700"><br>
        <em>QR page with the generated QR code and short url.</em>
      </td>
    </tr>
  </table>
</div>

## License

See [LICENSE](LICENSE).
