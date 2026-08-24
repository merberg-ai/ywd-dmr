#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
mkdir -p dist
VERSION="${YWD_DMR_VERSION:-0.0.0-dev}"
COMMIT="${YWD_DMR_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || printf unknown)}"
BRANCH="${YWD_DMR_BRANCH:-$(git branch --show-current 2>/dev/null || printf unknown)}"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.branch=${BRANCH}"

echo "Building YWD-DMR ${VERSION} (${BRANCH}@${COMMIT})"
go test ./...
go build -trimpath -ldflags "$LDFLAGS" -o dist/ywd-dmrd ./cmd/ywd-dmrd

echo "Built: $ROOT/dist/ywd-dmrd"
