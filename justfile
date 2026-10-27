# justfile
app_name := "qr2git"
bin_dir := "bin"
version := `tr -d '\r\n' < VERSION`
release_name := app_name + "-v" + version + "-linux-amd64"

# List available commands.
default:
    @just --list

# Install frontend and Go dependencies.
install:
    pnpm --dir web install --frozen-lockfile
    go mod download
    go -C tools/locales mod download

# Run every source, generated-file, test, and build check.
check: check-fmt vet test test-web lint typecheck check-locales check-generated build

# Fail when maintained Go source is not formatted.
check-fmt:
    #!/usr/bin/env bash
    set -euo pipefail
    unformatted="$(gofmt -l main.go api common core internal schema docs)"
    if [[ -n "$unformatted" ]]; then printf '%s\n' "$unformatted" >&2; exit 1; fi

# Format Go and frontend source.
fmt:
    gofmt -w main.go api common core internal schema docs
    pnpm --dir web exec eslint . --fix

# Check Go source statically.
vet:
    go vet ./...

# Run backend unit and SQLite integration tests.
test:
    go test ./...

# Run backend tests with the race detector.
test-race:
    go test -race ./...

# Run frontend behavior and documented-command tests.
test-web:
    pnpm --dir web test

# Check frontend code and templates without rewriting them.
lint:
    pnpm --dir web exec eslint . --max-warnings 0

# Check frontend TypeScript using the pinned checker.
typecheck:
    pnpm --dir web typecheck

# Generate both languages from the canonical locale sources.
locales:
    go -C tools/locales tool loc build -c ../../i18n/config.cue

# Check generated locales and every source key reference.
check-locales:
    go -C tools/locales tool loc check -c ../../i18n/config.cue
    go -C tools/locales tool loc lint keys -c ../../i18n/config.cue --strict

# Regenerate Go configuration types with the pinned CUE tool.
gen-types:
    bash scripts/gen-types.sh

# Check generated config, the shared package, and deployment settings.
check-generated:
    python3 scripts/check-generated.py

# Build the static site and embed it in the Go binary.
build: build-web build-api

# Build the production frontend.
build-web:
    pnpm --dir web build
    mkdir -p internal/frontend/dist
    cp -a web/.output/public/. internal/frontend/dist/

# Build the Go backend with the currently embedded frontend.
build-api:
    mkdir -p {{bin_dir}}
    CGO_ENABLED=0 go build -trimpath -ldflags '-s -w' -o {{bin_dir}}/{{app_name}} .

# Run the API and Nuxt dev server.
dev: build-api
    #!/usr/bin/env bash
    set -euo pipefail
    api_pid=""
    web_pid=""
    cleanup() {
        [[ -z "$api_pid" ]] || kill -- "-$api_pid" 2>/dev/null || true
        [[ -z "$web_pid" ]] || kill -- "-$web_pid" 2>/dev/null || true
        [[ -z "$api_pid" ]] || wait "$api_pid" 2>/dev/null || true
        [[ -z "$web_pid" ]] || wait "$web_pid" 2>/dev/null || true
    }
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    CONFIG_PATH=config.json.dev setsid ./bin/{{app_name}} &
    api_pid=$!
    setsid pnpm --dir web dev &
    web_pid=$!
    wait -n "$api_pid" "$web_pid"

# Run only the backend with the production-oriented local config.
run:
    go run .

# Run only the backend with development settings.
run-dev:
    CONFIG_PATH=config.json.dev go run .

# Package a portable Linux release.
release: build-web
    #!/usr/bin/env bash
    set -euo pipefail
    stage="$(mktemp -d -t qr2git-release-XXXXXXXX)"
    trap 'rm -rf -- "$stage"' EXIT
    mkdir -p "$stage/{{release_name}}"
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags '-s -w' -o "$stage/{{release_name}}/{{app_name}}" .
    cp README.md LICENSE VERSION config.json config.json.dev .env.example "$stage/{{release_name}}/"
    tar -C "$stage" -czf "{{release_name}}.tar.gz" "{{release_name}}"
    printf 'wrote %s\n' '{{release_name}}.tar.gz'

# Open the local SQLite database; sqlite3 is optional outside this recipe.
db-shell:
    sqlite3 data/qr2git.sqlite

# Download a complete production SQLite snapshot over SSH.
[positional-arguments]
db-download destination="data/qr2git-production.sqlite" host="ovh":
    python3 scripts/download-db.py "$1" --host "$2"

# Apply migrations by starting the backend.
migrate:
    go run .

# Regenerate the existing Swagger reference from Go annotations.
gen-swagger:
    go tool swag init -g main.go --output docs --parseInternal

# Resolve the Go module graph.
tidy:
    go mod tidy
