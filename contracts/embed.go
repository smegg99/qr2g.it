// contracts/embed.go

// Package contracts embeds the bundled API contract that the server publishes.
package contracts

import _ "embed"

// Bundle is contracts/backend/dist/openapi.yaml, written by `just contracts`.
//
//go:embed backend/dist/openapi.yaml
var Bundle []byte
