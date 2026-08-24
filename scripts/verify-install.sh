#!/usr/bin/env bash
set -u

ENV_FILE=/etc/ywd-dmr/ywd-dmr.env
PUBLIC_RUNTIME_FILE=/etc/ywd-dmr/runtime.conf
FAILED=0

ok()   { printf 'OK    %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILED=1; }
info() { printf 'INFO  %s\n' "$*"; }

printf 'YWD-DMR installed appliance verification\n'
printf '========================================\n'

[ -x /opt/ywd-dmr/current/bin/ywd-dmrd ] && ok "daemon binary installed" || fail "daemon binary missing"
[ -f /opt/ywd-dmr/current/web/index.html ] && ok "WebUI installed" || fail "WebUI missing"
[ -f /opt/ywd-dmr/current/docs/README.md ] && ok "versioned docs installed" || fail "docs missing"
[ -f "$ENV_FILE" ] && ok "protected daemon configuration installed" || fail "protected daemon configuration missing"
[ -f "$PUBLIC_RUNTIME_FILE" ] && ok "public runtime metadata installed" || fail "public runtime metadata missing"
[ -r "$PUBLIC_RUNTIME_FILE" ] && ok "public runtime metadata readable" || fail "public runtime metadata is not readable"
[ "$(stat -c '%a' "$ENV_FILE" 2>/dev/null)" = 640 ] && ok "protected daemon configuration mode is 0640" || fail "protected daemon configuration permissions are not 0640"
[ "$(stat -c '%a' "$PUBLIC_RUNTIME_FILE" 2>/dev/null)" = 644 ] && ok "public runtime metadata mode is 0644" || fail "public runtime metadata permissions are not 0644"
[ -f /etc/systemd/system/ywd-dmrd.service ] && ok "systemd unit installed" || fail "systemd unit missing"
[ -x /usr/local/bin/ywd-dmr ] && ok "maintenance CLI installed" || fail "maintenance CLI missing"
[ -x /usr/local/sbin/ywd-dmr-uninstall ] && ok "safe uninstaller installed" || fail "safe uninstaller missing"
[ -f /etc/ywd-dmr/install-owned-user ] && ok "installer ownership marker present" || fail "ownership marker missing"

if systemctl is-enabled --quiet ywd-dmrd.service 2>/dev/null; then ok "service enabled"; else fail "service not enabled"; fi
if systemctl is-active --quiet ywd-dmrd.service 2>/dev/null; then ok "service active"; else fail "service not active"; fi

LISTEN="$(sed -n 's/^YWD_DMR_LISTEN=//p' "$PUBLIC_RUNTIME_FILE" 2>/dev/null | tail -1)"
PROTECTED_LISTEN="$(sed -n 's/^YWD_DMR_LISTEN=//p' "$ENV_FILE" 2>/dev/null | tail -1)"
PORT="${LISTEN##*:}"
info "configured listener: ${LISTEN:-unknown}"

if [ -n "$LISTEN" ] && [ "$LISTEN" = "$PROTECTED_LISTEN" ]; then
  ok "public and protected listener settings match"
else
  fail "public and protected listener settings do not match"
fi

if command -v curl >/dev/null 2>&1 && [ -n "$PORT" ]; then
  if curl -fsS --max-time 3 "http://127.0.0.1:$PORT/api/v1/health" >/dev/null 2>&1; then
    ok "health API responded on port $PORT"
  else
    fail "health API did not respond on port $PORT"
  fi
else
  fail "curl or configured port unavailable for health test"
fi

if [ -L /opt/ywd-dmr/current ]; then
  info "current release: $(readlink -f /opt/ywd-dmr/current)"
fi
if [ -L /opt/ywd-dmr/previous ]; then
  info "previous release: $(readlink -f /opt/ywd-dmr/previous)"
fi

if [ "$FAILED" -ne 0 ]; then
  printf '\nVerification FAILED. Run: ywd-dmr logs\n'
  exit 1
fi
printf '\nVerification PASSED.\n'
