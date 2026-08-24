#!/usr/bin/env bash
set -u

PURGE_DATA=0
FAILED=0

usage() {
  cat <<'EOF'
YWD-DMR uninstall verification

Usage:
  bash scripts/verify-uninstall.sh [--purge-data]

Without --purge-data, verifies a normal software-only uninstall: software,
service, installer-owned firewall integration, and maintenance helpers must be
gone while persistent configuration/data and the service account remain.

With --purge-data, verifies a full purge: persistent configuration/data and the
installer-created service account must also be gone.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --purge-data) PURGE_DATA=1 ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown option: $1" >&2; usage; exit 2 ;;
  esac
  shift
done

ok()   { printf 'OK    %s\n' "$*"; }
fail() { printf 'FAIL  %s\n' "$*"; FAILED=1; }
info() { printf 'INFO  %s\n' "$*"; }

printf 'YWD-DMR uninstall verification\n'
printf '==============================\n'

check_absent_path() {
  local path="$1" label="$2"
  if [ ! -e "$path" ] && [ ! -L "$path" ]; then
    ok "$label removed"
  else
    fail "$label still exists at $path"
  fi
}

check_present_dir() {
  local path="$1" label="$2"
  if [ -d "$path" ]; then
    ok "$label preserved"
  else
    fail "$label missing at $path"
  fi
}

check_absent_path /opt/ywd-dmr "application tree"
check_absent_path /etc/systemd/system/ywd-dmrd.service "systemd unit"
check_absent_path /usr/local/bin/ywd-dmr "maintenance CLI"
check_absent_path /usr/local/sbin/ywd-dmr-uninstall "installed uninstaller"
check_absent_path /etc/ywd-dmr/firewall.conf "firewall ownership metadata"

if command -v systemctl >/dev/null 2>&1; then
  if systemctl cat ywd-dmrd.service >/dev/null 2>&1; then
    fail "systemd still resolves ywd-dmrd.service"
  else
    ok "systemd no longer resolves ywd-dmrd.service"
  fi
fi

if command -v ufw >/dev/null 2>&1; then
  if LC_ALL=C ufw show added 2>/dev/null | grep -F "YWD-DMR managed LAN" >/dev/null 2>&1; then
    fail "installer-owned UFW rule with YWD-DMR managed LAN tag still exists"
  else
    ok "no installer-owned YWD-DMR UFW rule remains"
  fi
fi

if [ "$PURGE_DATA" -eq 1 ]; then
  check_absent_path /etc/ywd-dmr "configuration directory"
  check_absent_path /var/lib/ywd-dmr "data/plugins directory"
  check_absent_path /var/log/ywd-dmr "log directory"
  check_absent_path /var/backups/ywd-dmr "managed backup directory"

  if getent passwd ywd-dmr >/dev/null 2>&1; then
    fail "ywd-dmr service account still exists after full purge"
  else
    ok "ywd-dmr service account removed"
  fi
else
  check_present_dir /etc/ywd-dmr "configuration directory"
  check_present_dir /var/lib/ywd-dmr "data/plugins directory"
  check_present_dir /var/log/ywd-dmr "log directory"
  check_present_dir /var/backups/ywd-dmr "managed backup directory"

  if getent passwd ywd-dmr >/dev/null 2>&1; then
    ok "ywd-dmr service account preserved"
  else
    fail "ywd-dmr service account missing after software-only uninstall"
  fi
fi

# Do not use `command -v ywd-dmr` here. Bash may retain a removed command in
# its per-shell hash table. The physical /usr/local/bin path above is the
# authoritative uninstall check. Clearing the parent shell's hash with
# `hash -r` is harmless but cannot be done by the child uninstaller process.
if hash -t ywd-dmr >/dev/null 2>&1; then
  info "current shell may still have a cached ywd-dmr command path; run: hash -r"
fi

if [ "$FAILED" -ne 0 ]; then
  printf '\nVerification FAILED.\n'
  exit 1
fi

printf '\nVerification PASSED.\n'
