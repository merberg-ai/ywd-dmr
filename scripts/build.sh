#!/usr/bin/env bash
set -Eeuo pipefail
ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUTPUT="${YWD_DMR_BUILD_OUTPUT:-$ROOT/dist/ywd-dmrd}"
OUTPUT_DIR="$(dirname -- "$OUTPUT")"
mkdir -p "$OUTPUT_DIR"

# The development appliance installer currently invokes this script under sudo.
# Never leave its generated checkout artifact owned by root, otherwise the
# developer's next normal-user build cannot replace dist/ywd-dmrd. Run this on
# EXIT so ownership is repaired even when a later build/test step fails.
repair_sudo_checkout_ownership() {
  local group
  if [ "$(id -u)" -eq 0 ] \
     && [ -n "${SUDO_USER:-}" ] \
     && [ "${SUDO_USER:-root}" != root ] \
     && [ "$OUTPUT_DIR" = "$ROOT/dist" ]; then
    group="$(id -gn "$SUDO_USER" 2>/dev/null || printf '%s' "$SUDO_USER")"
    chown "$SUDO_USER:$group" "$OUTPUT_DIR" 2>/dev/null || true
    if [ -e "$OUTPUT" ] || [ -L "$OUTPUT" ]; then
      chown "$SUDO_USER:$group" "$OUTPUT" 2>/dev/null || true
    fi
  fi
}
trap repair_sudo_checkout_ownership EXIT

VERSION="${YWD_DMR_VERSION:-0.0.0-dev}"
COMMIT="${YWD_DMR_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || printf unknown)}"
BRANCH="${YWD_DMR_BRANCH:-$(git branch --show-current 2>/dev/null || printf unknown)}"
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.branch=${BRANCH}"

echo "Building YWD-DMR ${VERSION} (${BRANCH}@${COMMIT})"
go test ./...
go build -trimpath -ldflags "$LDFLAGS" -o "$OUTPUT" ./cmd/ywd-dmrd

echo "Built: $OUTPUT"
