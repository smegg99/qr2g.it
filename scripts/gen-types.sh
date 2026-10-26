#!/usr/bin/env bash
# Regenerate common/config/cue_types_gen.go from schema/config.cue.
# Uses the CUE version pinned in go.mod.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

GENERATED=./schema/cue_types_schema_gen.go
rm -f "$GENERATED"
go tool cue exp gengotypes ./schema
mv "$GENERATED" ./common/config/cue_types_gen.go
gofmt -s -w ./common/config/cue_types_gen.go
echo "regenerated common/config/cue_types_gen.go"
