<p align="center">
  <img src="assets/qr2g.it.svg" alt="qr2g.it" width="500">
</p>

qr2g.it is a backend API for permanent, QR-friendly short links to public GitHub repositories. A link is bound to the GitHub repository identity recorded by GitHub (`node_id` and database id), not just the current `owner/repo` text, so the slug keeps resolving after renames or transfers when verification succeeds.

This repository is backend-only. There is no frontend, dashboard, deployment manifest, or QR image endpoint.

Built with Go 1.26.2, Gin, PostgreSQL, pgx, CUE-backed JSON configuration, GitHub REST/GraphQL API clients, and Cloudflare Turnstile verification.

## Use It

The fastest way to get a short link is to point any HTTP client at `qr2g.it/{owner}/{repo}`.

From a terminal:

```sh
curl qr2g.it/smegg99/ThugHunter
# https://qr2g.it/AB12
```

`curl`, `wget`, `HTTPie`, `python-requests`, `Go-http-client`, and PowerShell get a `text/plain` body with the canonical short URL (terminated by a newline so it pipes cleanly into `xclip`, `pbcopy`, or another command).

In a browser, the same URL is a 302 chain: `qr2g.it/smegg99/ThugHunter` → `qr2g.it/AB12` → `https://github.com/smegg99/ThugHunter`.

Force the plain-text shape from any client with `?format=text`, `?text=1`, or `Accept: text/plain`. Force a redirect with `curl -L`.

## What It Does

- Creates or reuses a permanent short link for any public GitHub repository URL via `POST /api/v1/links`.
- Accepts shorthand `GET /{owner}/{repo}`, with content negotiation between a 302 redirect for browsers and a plain-text body for CLI tools.
- Encodes slugs as uppercase base36 of the Postgres `links.id`, between one and twelve characters, with no leading zero.
- Redirects `GET /{SLUG}` with `302 Found` to the current verified GitHub repository URL; verifies stale links before redirecting.
- Exposes `GET /{SLUG}+` and `/api/v1/links/:slug` for metadata without redirecting.
- Stores GitHub `node_id` aliases so migrated GitHub identities still map back to the original link.
- Runs a background verifier that re-checks active links against GitHub and tombstones private or deleted repositories after repeated failures.
- Applies per-IP and GitHub-facing rate limits in process, with a small circuit breaker around GitHub.

## Prerequisites

- Go matching the directive in `go.mod` (currently `1.26.2`).
- PostgreSQL reachable by `DATABASE_URL`.
- Optional: `just` for the recipes in `justfile`.
- Optional: a GitHub App installation for authenticated REST and strict GraphQL verification by node id. Without it the service falls back to unauthenticated REST lookup by GitHub database id.
- Optional: Cloudflare Turnstile keys for `POST /api/v1/links` abuse protection. Off by default in the shipped `config.json`.
- Optional development tools: `cue` for `just gen-types`, `sqlc` for `just gen-sqlc`, `psql` for the database recipes.

## Setup

Copy the example environment file and fill in a Postgres connection string:

```sh
cp .env.example .env
```

```sh
DATABASE_URL=postgres://qr2g:qr2g@localhost:5432/qr2g?sslmode=disable
```

For GitHub App authentication and GraphQL verification, also set:

```sh
GITHUB_APP_ID=12345
GITHUB_APP_INSTALLATION_ID=67890
GITHUB_APP_PRIVATE_KEY_PATH=/path/to/github-app-private-key.pem
```

For Turnstile-protected link creation (disabled out of the box):

```sh
TURNSTILE_SECRET_KEY=...
TURNSTILE_SITE_KEY=...
```

Configuration is read from `config.json`, `config/config.json`, or the path in `CONFIG_PATH`. If no config exists, the binary writes a default `config.json` derived from `schema/config.cue` and exits so the operator can review it.

## Running

```sh
go run ./
# or
just run
```

By default the service listens on `:8080`, opens a Postgres pool, applies embedded SQL migrations when `db.auto_migrate` is true, initializes the GitHub and Turnstile clients, starts the background verifier, and serves the HTTP API. SIGINT or SIGTERM trigger a graceful shutdown bounded by `server.shutdown_timeout_seconds`.

Build a release binary into `bin/qr2g`:

```sh
just build
```

## API

Interactive API docs are served at `/swagger/index.html` once the binary is running. The OpenAPI spec is committed under `docs/` and regenerated with `just gen-swagger`.

### Create a link

```sh
curl -s -X POST http://localhost:8080/api/v1/links \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://github.com/owner/repo","turnstileToken":"TOKEN"}'
```

Accepted GitHub URL forms:

- `https://github.com/owner/repo`
- `https://github.com/owner/repo/`
- `https://github.com/owner/repo.git`

The parser rejects non-HTTPS URLs, non-`github.com` hosts, credentials, ports, query strings, fragments, extra path components, and reserved owner-level paths such as `api` or `settings`.

Example response:

```json
{
  "url": "https://qr2g.it/ABC123",
  "slug": "ABC123",
  "id": 623698779,
  "initialFullName": "owner/repo",
  "currentFullName": "owner/repo",
  "currentUrl": "https://github.com/owner/repo",
  "state": "active",
  "createdAt": "2026-05-05T19:22:00Z",
  "githubNodeId": "R_kgDOExample",
  "githubDatabaseId": 123456789
}
```

### Routes

| Method | Path | Behavior |
|---|---|---|
| `GET` | `/` | JSON service metadata. |
| `GET` | `/health` | Liveness check. |
| `GET` | `/ready` | Readiness check; pings Postgres. |
| `GET` | `/robots.txt` | Returns `Disallow: /`. |
| `GET` | `/favicon.ico` | Empty 204. |
| `POST` | `/api/v1/links` | Creates or reuses a link for a public GitHub repository URL. |
| `GET` | `/api/v1/links/:slug` | Returns stored link metadata without verifying. |
| `GET` | `/api/v1/links/:slug/resolve` | Resolves as JSON, verifying stale active links first. |
| `GET` | `/{SLUG}` | Redirects to the current GitHub URL, verifying first when stale. |
| `GET` | `/{SLUG}+` | Returns metadata for a slug without redirecting. |
| `GET` | `/{owner}/{repo}` | Finds or creates a link. CLI clients get plain text; browsers get a 302 to the slug. |
| `GET` | `/swagger/index.html` | Swagger UI for the JSON API. |

Slug rules live in `core/slug`: uppercase base36 only, one to twelve characters, no leading zero, no lowercase canonical form.

## Configuration

`config.json` is validated against `schema/config.cue`. String values may reference environment variables, files, the OS keyring, or config/data directories with `@{...}` references. Private references (`@{env:...}`, `@{file:...}`, `@{keyring:...}`) are excluded from the exported public config map. Public references (`@{pubenv:...}`) are kept.

| Option | Default | Description |
|---|---|---|
| `server.listen_addr` | `:8080` | HTTP listen address. |
| `server.public_base_url` | `http://localhost:8080` | Base URL used for slug redirects. |
| `server.canonical_base_url` | `HTTP://LOCALHOST:8080` | Base URL used in API responses and CLI plain-text output. |
| `server.trusted_proxy_mode` | `cloudflare` | Client IP extraction mode: `cloudflare`, `none`, or `all`. |
| `server.require_shorthand_turnstile` | `false` | Requires Turnstile for cold `/{owner}/{repo}` creation. Leaving this off is what makes the curl UX work. |
| `db.url` | `@{env?:DATABASE_URL}` | Postgres connection string. The service refuses to start if this resolves empty. |
| `db.auto_migrate` | `true` | Applies embedded migrations at startup. |
| `github.use_app_auth` | `true` | Sends GitHub App installation tokens when App credentials are configured. |
| `github.use_graphql` | `true` | Enables GraphQL verification only when App auth is usable. |
| `turnstile.enabled` | `false` | Enables Turnstile checks on `POST /api/v1/links`. |
| `verifier.enabled` | `true` | Runs the background stale-link verifier. |
| `verifier.fresh_ttl_seconds` | `3600` | How long a verified link stays fresh before re-verification. |
| `verifier.max_fail_count` | `10` | Consecutive failures before a private/deleted repo is tombstoned. |
| `ratelimit.create_per_min` | `3` | Per-IP create burst/rate limit. |
| `ratelimit.create_per_day` | `50` | Per-IP daily create cap. |
| `ratelimit.shorthand_per_hour` | `5` | Per-IP cold-create cap on the `/{owner}/{repo}` shorthand. |
| `ratelimit.resolve_per_min` | `600` | Per-IP slug resolve limit. |

## Verification Behavior

Fresh active links redirect immediately to the cached GitHub URL. Stale active links are verified before redirecting. When GraphQL is available, verification resolves the GitHub repository by `node_id`; otherwise it falls back to REST lookup by GitHub database id.

Verification fails closed: a stale link is not redirected if GitHub verification fails, rate limits, or is unavailable. Private or deleted repositories are only tombstoned after `verifier.max_fail_count` consecutive failures; tombstoned slugs return state information instead of being recycled.

The background verifier polls in batches with `FOR UPDATE SKIP LOCKED` so multiple instances can share the load safely. A small circuit breaker (`common/breaker`) guards GitHub.

## Development

```sh
just test
just test-race
just vet
just fmt
```

Regenerate Go config types from the CUE schema:

```sh
just gen-types
```

Generate sqlc code from the query files (the active repository layer in `core/repositories` is hand-written pgx; `sqlc.yaml` is present for generating compatible code under `core/repositories/sqlcgen` if needed):

```sh
just gen-sqlc
```

Database recipes:

```sh
just db-shell
just migrate
just db-reset   # DESTRUCTIVE; drops the local schema, dev only
```

## License

MIT. See [LICENSE](LICENSE).
