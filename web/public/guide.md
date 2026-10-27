# qr2g.it usage guide

qr2g.it gives each public GitHub repository one permanent short link, such as
`https://qr2g.it/2`, and renders QR codes and share cards for it. The link
follows the repository through renames and transfers. Generating a link does
not make anyone the owner of the repository or of its settings.

The machine-readable contract is <https://qr2g.it/api/v1/openapi.yaml>.

## For chats: prefilled design links

A chat can answer "make me a blue QR code for smegg99/Goptivum" with a
normal link that opens qr2g.it with the repository and QR design filled in:

```text
https://qr2g.it/?repo=smegg99%2FGoptivum&fg=%230b3d91
```

Opening the link creates nothing. The person presses **Get link**; qr2g.it then
verifies the repository and opens the QR editor with these choices.

Rules:

- A design link is not a short link. Never invent a slug such as
  `https://qr2g.it/XYZ` and never say a link exists until qr2g.it shows it.
- URL-encode every value. `repo` is `owner/name`, a `github.com/owner/name`
  path or a full `https://github.com/owner/name` URL.
- Unknown or invalid parameters are ignored and listed on the page.
- Never put tokens, email addresses or secrets in a design link.

| Parameter | Values | Default |
| --- | --- | --- |
| `repo` | repository | none |
| `fg`, `bg` | QR module and tile colour, `#rrggbb`; contrast of at least 4.5 | `#111111`, `#ffffff` |
| `logo` | `true` draws the qr2g.it fox in the centre, in the QR colour | `true` |
| `ec` | `L`, `M`, `Q`, `H`; the logo needs `H` | `H` with the logo, else `M` |
| `margin` | quiet zone in modules, 4 to 16 | 4 |
| `radius` | tile corner radius, 0 to 100 percent of the safe maximum | 0 |
| `size` | width in px, 128 to 4000 | 1024 |
| `transparent` | `true` leaves out the tile | `false` |
| `format` | `svg`, `png`, `jpg` | `png` |

The same names, except `repo`, are query parameters of the asset operation
below.

## For scripts and coding agents: HTTP API

Base URL: `https://qr2g.it/api/v1`. Every error answers
`{"code": "...", "message": "errors...."}`; branch on `code`.

### Read without creating

```sh
curl --fail-with-body -sS "https://qr2g.it/api/v1/links/lookup?repository=smegg99/qr2g.it"
```

A pure read: no link is created and GitHub is not called. `404 link_not_found`
means the repository has no link yet. `GET /links/{slug}` reads by slug and
`GET /links/{slug}/resolve` re-verifies a stale link with GitHub.

### Create or reuse

Browsers create links with a Turnstile token. Automation uses a token issued
by the operator:

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"url":"https://github.com/smegg99/qr2g.it"}'
```

`201` means this request created the link; `200` means it already existed.
Repeating the request returns the same canonical link, so it is safe to retry.

Up to 25 repositories at once, each with its own outcome or error:

```sh
curl --fail-with-body -sS https://qr2g.it/api/v1/links/batch \
  -H "Authorization: Bearer $QR2GIT_TOKEN" \
  -H 'Content-Type: application/json' \
  --data '{"repositories":["smegg99/qr2g.it","smegg99/Goptivum"]}'
```

The shorthand `curl -fsS https://qr2g.it/smegg99/qr2g.it` prints the short
link; `curl -fsS "https://qr2g.it/smegg99/qr2g.it?qr"` also draws its QR code
with Unicode blocks for a dark terminal (`?qr=invert` for a light one). It is
not a pure read: for an unknown repository it verifies it on GitHub
and creates the link (5 per hour per IP, and the operator can require a
browser check).

### Render files

```sh
curl --fail-with-body -sS -OJ "https://qr2g.it/api/v1/links/2/asset?format=png&download=true"
```

The website makes QR codes only. The API also renders other presets with
`preset`: `presentation`, `poster`, `label` and `badge`, with `theme`,
`caption`, `accent`, `labelWidth`, `labelHeight` and `dpi`; the contract
describes each. Formats: `svg`, `png`, `jpg` (`badge` and transparent codes
cannot be JPG). The response sets `Content-Disposition` with a name such as
`Goptivum-qr.png`, and
`X-Asset-Width` and `X-Asset-Height`. Text is drawn as outlines, so files need
no fonts. Codes keep a four-module quiet zone and need a contrast of at least
4.5 between modules and tile. Light-on-dark codes are allowed, but some older
scanners read only dark-on-light ones; test printed output, because no preview
proves every device scans it.

### Listed repositories and statistics

```sh
curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories?q=timetable&sort=popular30&page=1&limit=24"
curl --fail-with-body -sS https://qr2g.it/api/v1/links/2/stats
curl --fail-with-body -sS "https://qr2g.it/api/v1/repositories/top?limit=3"
```

`/repositories/top` is what the home page shows: the most visited listed
repositories over the last 30 days, each with daily counts, its total and the
time of its latest counted redirect. Only repositories submitted to qr2g.it appear, and only while they are public,
passed an owner policy check and are not opted out or removed. Statistics are
filtered redirects: successful final redirects to GitHub excluding HEAD
requests, identifiable bots, link previews, prefetches and authenticated API
traffic. They are not unique people or QR scans, and filtering is imperfect.

### Limits and retries

- `429` carries `Retry-After` in seconds. Wait that long.
- Token requests carry `X-RateLimit-Limit`, `X-RateLimit-Remaining` and
  `X-RateLimit-Reset` (Unix seconds) for the daily quota.
  `GET /automation/usage` reads it without spending it.
- `503` (`github_unavailable`, `github_rate_limited`, `verification_failed`)
  is temporary; retry with backoff. `github_rate_limited` carries
  `Retry-After` until GitHub's allowance resets.
- A batch answers within about 10 seconds; an entry still waiting on GitHub
  fails with `github_unavailable` and can be retried on its own.
- A claimed AI User-Agent is not authorization.

## Owner opt-out

On the default branch, an empty `.qr2git-noindex` file or a standalone
`<!-- qr2git: noindex -->` README comment hides the repository from the home
page, search, rankings and statistics, and direct link reads and cards
omit its description and language; the short link keeps working.
`.qr2git-noanalytics` or `<!-- qr2git: noanalytics -->` stops future usage
collection. Markers inside fenced code examples are ignored. Reports and
removal requests: <https://qr2g.it/contact>.
