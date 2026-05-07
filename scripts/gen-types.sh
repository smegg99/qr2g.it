#!/usr/bin/env bash
# Regenerate common/config/cue_types_gen.go from schema/config.cue.
# Requires `cue` >= v0.16 (the exp gengotypes subcommand).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if ! command -v cue >/dev/null 2>&1; then
    echo "cue not found in PATH. Install: https://cuelang.org/docs/install/" >&2
    exit 1
fi

cue exp gengotypes ./schema -o ./common/config/cue_types_gen.go
gofmt -s -w ./common/config/cue_types_gen.go
echo "regenerated common/config/cue_types_gen.go"
