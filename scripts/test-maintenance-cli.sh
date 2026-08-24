#!/usr/bin/env bash
set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

PUBLIC="$TMP/runtime.conf"
PROTECTED="$TMP/ywd-dmr.env"

cat > "$PUBLIC" <<'EOF'
YWD_DMR_LISTEN=127.0.0.1:8997
EOF

actual="$(YWD_DMR_PUBLIC_RUNTIME_FILE="$PUBLIC" YWD_DMR_ENV_FILE="$PROTECTED" bash "$ROOT/scripts/ywd-dmr" url)"
expected='http://127.0.0.1:8997/'
[ "$actual" = "$expected" ] || {
  echo "FAIL: expected $expected, got $actual" >&2
  exit 1
}

echo "OK: maintenance CLI reads public runtime listener"

rm -f "$PUBLIC" "$PROTECTED"
actual="$(YWD_DMR_PUBLIC_RUNTIME_FILE="$PUBLIC" YWD_DMR_ENV_FILE="$PROTECTED" bash "$ROOT/scripts/ywd-dmr" url)"
expected='http://127.0.0.1:8989/'
[ "$actual" = "$expected" ] || {
  echo "FAIL: expected fallback $expected, got $actual" >&2
  exit 1
}

echo "OK: maintenance CLI falls back to safe default listener"
