#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
export YWD_DMR_WEB_ROOT="${YWD_DMR_WEB_ROOT:-$ROOT/web}"
export YWD_DMR_DOCS_ROOT="${YWD_DMR_DOCS_ROOT:-$ROOT/docs}"
export YWD_DMR_LISTEN="${YWD_DMR_LISTEN:-127.0.0.1:8090}"
exec go run ./cmd/ywd-dmrd
