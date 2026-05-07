app_name := "qr2g"
bin_dir := "bin"

# Default: list available commands
default:
    @just --list

# Tidy go modules
tidy:
    go mod tidy

# Build the backend binary into ./bin/qr2g
build:
    mkdir -p {{bin_dir}}
    go build -trimpath -o {{bin_dir}}/{{app_name}} ./

# Run the backend (uses local config.json + .env)
run:
    go run ./

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
