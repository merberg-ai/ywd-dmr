#!/usr/bin/env bash
set -u

ENV_FILE=/etc/ywd-dmr/ywd-dmr.env
PUBLIC_RUNTIME_FILE=/etc/ywd-dmr/runtime.conf
FIREWALL_META_FILE=/etc/ywd-dmr/firewall.conf
FAILED=0

ok()   { printf 'OK    %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILED=1; }
info() { printf 'INFO  %s\n' "$*"; }

meta_value() {
  local key="$1" file="$2" value
  [ -f "$file" ] || return 1
  value="$(sed -n "s/^${key}=//p" "$file" 2>/dev/null | tail -1)"
  [ -n "$value" ] || return 1
  printf '%s\n' "$value"
}

ufw_managed_rule_present() {
  local source="$1" port="$2" comment="$3"
  command -v ufw >/dev/null 2>&1 || return 1
  LC_ALL=C ufw show added 2>/dev/null \
    | grep -F "$source" \
    | grep -F "port $port" \
    | grep -F "$comment" >/dev/null 2>&1
}

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
HOST="${LISTEN%:*}"
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

if [ -f "$FIREWALL_META_FILE" ]; then
  FW_PROVIDER="$(meta_value YWD_DMR_FIREWALL_PROVIDER "$FIREWALL_META_FILE" 2>/dev/null || true)"
  FW_MANAGED="$(meta_value YWD_DMR_FIREWALL_MANAGED "$FIREWALL_META_FILE" 2>/dev/null || true)"
  FW_STATUS="$(meta_value YWD_DMR_FIREWALL_STATUS "$FIREWALL_META_FILE" 2>/dev/null || printf ok)"
  FW_SOURCE="$(meta_value YWD_DMR_FIREWALL_SOURCE "$FIREWALL_META_FILE" 2>/dev/null || true)"
  FW_PORT="$(meta_value YWD_DMR_FIREWALL_PORT "$FIREWALL_META_FILE" 2>/dev/null || true)"
  FW_COMMENT="$(meta_value YWD_DMR_FIREWALL_COMMENT "$FIREWALL_META_FILE" 2>/dev/null || true)"

  [ "$(stat -c '%a' "$FIREWALL_META_FILE" 2>/dev/null)" = 644 ] && ok "firewall metadata mode is 0644" || fail "firewall metadata permissions are not 0644"

  if [ "$FW_STATUS" = failed ]; then
    fail "YWD-DMR firewall setup failed for $FW_SOURCE -> $FW_PORT/tcp"
  elif [ "$FW_MANAGED" = 1 ]; then
    if [ "$FW_PROVIDER" = ufw ] && ufw_managed_rule_present "$FW_SOURCE" "$FW_PORT" "$FW_COMMENT"; then
      ok "YWD-DMR-managed UFW rule is present for $FW_SOURCE -> $FW_PORT/tcp"
    else
      fail "firewall metadata says YWD-DMR owns a rule, but the tagged UFW rule was not found"
    fi
  else
    info "firewall rule is existing/user-owned and will not be removed by YWD-DMR"
  fi
elif [ "$HOST" = 0.0.0.0 ]; then
  info "no YWD-DMR firewall metadata; LAN access may rely on a user-managed firewall rule"
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