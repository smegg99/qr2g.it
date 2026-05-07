app_name := "qr2git"
bin_dir := "bin"
release_name := "qr2git-V1.0.0-x86-64"
release_root := "release"
frontend_embed_dir := "internal/frontend/dist"

# Default: list available commands
default:
    @just --list

# Tidy go modules
tidy:
    go mod tidy

# Build the backend binary into ./bin/qr2git
build:
    mkdir -p {{bin_dir}}
    go build -trimpath -o {{bin_dir}}/{{app_name}} ./

# Build Nuxt, embed it into the Go binary, and package a Linux x86-64 release tarball
release:
    rm -rf {{release_root}}/{{release_name}} {{release_name}}.tar.gz
    mkdir -p {{frontend_embed_dir}} {{release_root}}/{{release_name}}
    find {{frontend_embed_dir}} -mindepth 1 -maxdepth 1 ! -name placeholder.txt -exec rm -rf {} +
    pnpm --dir web install --frozen-lockfile
    pnpm --dir web build
    cp -a web/.output/public/. {{frontend_embed_dir}}/
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o {{release_root}}/{{release_name}}/{{app_name}} ./
    cp README.md LICENSE config.json config.json.dev .env.example {{release_root}}/{{release_name}}/
    install -m 0644 packaging/systemd/{{app_name}}.service.template {{release_root}}/{{release_name}}/{{app_name}}.service.template
    install -m 0755 packaging/install-systemd.sh {{release_root}}/{{release_name}}/install-systemd.sh
    tar -C {{release_root}} -czf {{release_name}}.tar.gz {{release_name}}
    @printf 'wrote %s\n' "{{release_name}}.tar.gz"

# Run the backend (uses local config.json + .env)
run:
    go run ./

# Run the backend with the local development config
run-dev:
    CONFIG_PATH=config.json.dev go run ./

# Run unit tests
test:
    go test ./...

# Run unit tests with race detector and verbose output
test-race:
    go test -race -v ./...

# Run go vet
vet:
    go vet ./...

# Format the codebase
fmt:
    gofmt -s -w .

# Regenerate Go config types from the CUE schema (requires `cue` >= v0.16)
gen-types:
    bash scripts/gen-types.sh

# Inspect the database schema (requires psql)
db-shell:
    psql $DATABASE_URL

# Reset the local Postgres schema (DESTRUCTIVE - local dev only)
db-reset:
    psql $DATABASE_URL -c "DROP TABLE IF EXISTS schema_migrations, github_node_id_aliases, links CASCADE; DROP TYPE IF EXISTS link_state;"

# Generate sqlc code from core/datastore/queries (requires sqlc)
gen-sqlc:
    sqlc generate

# Regenerate the OpenAPI spec under ./docs (requires `swag` >= v1.16)
gen-swagger:
    swag init -g main.go --output docs --parseInternal

# Run a one-shot migration apply via the binary's startup path
migrate:
    go run ./
