#!/usr/bin/env bash
# YWD-DMR development appliance installer
#
# This is the first testable install path. Production releases will install a
# verified GitHub release payload rather than compiling from a working tree.

set -Eeuo pipefail

ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
DEFAULT_PORT=8989
PORT=""
BIND_ADDR=""
LAN_TEST=0
ASSUME_YES=0
MANAGE_FIREWALL=1
UFW_SOURCE_OVERRIDE=""

APP_DIR=/opt/ywd-dmr
CONFIG_DIR=/etc/ywd-dmr
DATA_DIR=/var/lib/ywd-dmr
LOG_DIR=/var/log/ywd-dmr
BACKUP_DIR=/var/backups/ywd-dmr
ENV_FILE="$CONFIG_DIR/ywd-dmr.env"
PUBLIC_RUNTIME_FILE="$CONFIG_DIR/runtime.conf"
FIREWALL_META_FILE="$CONFIG_DIR/firewall.conf"
OWNED_USER_MARKER="$CONFIG_DIR/install-owned-user"
UNIT_FILE=/etc/systemd/system/ywd-dmrd.service
CLI_FILE=/usr/local/bin/ywd-dmr
UNINSTALL_FILE=/usr/local/sbin/ywd-dmr-uninstall
SERVICE_USER=ywd-dmr
FIREWALL_COMMENT="YWD-DMR managed LAN"

usage() {
  cat <<'EOF'
YWD-DMR development installer

Usage:
  sudo ./scripts/install.sh [options]

Options:
  --port PORT       Frontend/API TCP port (default: 8989)
  --local           Listen on this computer only (127.0.0.1)
  --lan-test        Listen on the local network (0.0.0.0). WARNING: the current
                    development dashboard does not have authentication yet.
  --no-firewall     Do not add or change a firewall rule.
  --ufw-source CIDR Override the automatically detected LAN subnet used for a
                    UFW rule, for example 192.168.1.0/24.
  --yes             Accept normal non-destructive prompts.
  -h, --help        Show this help.

Examples:
  sudo ./scripts/install.sh
  sudo ./scripts/install.sh --port 8989 --local
  sudo ./scripts/install.sh --port 8989 --lan-test
  sudo ./scripts/install.sh --lan-test --ufw-source 192.168.1.0/24
EOF
}

log()  { printf '%s\n' "$*"; }
warn() { printf 'WARNING: %s\n' "$*" >&2; }
die()  { printf 'ERROR: %s\n' "$*" >&2; exit 1; }

while [ "$#" -gt 0 ]; do
  case "$1" in
    --port) shift; [ "$#" -gt 0 ] || die "--port needs a value"; PORT="$1" ;;
    --local) BIND_ADDR=127.0.0.1 ;;
    --lan-test) BIND_ADDR=0.0.0.0; LAN_TEST=1 ;;
    --no-firewall) MANAGE_FIREWALL=0 ;;
    --ufw-source) shift; [ "$#" -gt 0 ] || die "--ufw-source needs a CIDR value"; UFW_SOURCE_OVERRIDE="$1" ;;
    --yes) ASSUME_YES=1 ;;
    -h|--help) usage; exit 0 ;;
    *) die "Unknown option: $1" ;;
  esac
  shift
done

[ "$(id -u)" -eq 0 ] || die "Please run the installer with sudo/root privileges."
[ "$(uname -s)" = Linux ] || die "This installer currently supports Linux only."
command -v systemctl >/dev/null 2>&1 || die "systemd/systemctl is required for this appliance installer."

install_pkg_if_missing() {
  local cmd="$1" pkg="$2"
  command -v "$cmd" >/dev/null 2>&1 && return 0
  if command -v apt-get >/dev/null 2>&1; then
    log "Installing required package: $pkg"
    apt-get update
    DEBIAN_FRONTEND=noninteractive apt-get install -y "$pkg"
  else
    die "Required command '$cmd' is missing. Install package '$pkg' and run the installer again."
  fi
  command -v "$cmd" >/dev/null 2>&1 || die "Could not install required command '$cmd'."
}

install_pkg_if_missing ss iproute2
install_pkg_if_missing curl curl
install_pkg_if_missing go golang-go

meta_value() {
  local key="$1" file="$2" value
  [ -f "$file" ] || return 1
  value="$(sed -n "s/^${key}=//p" "$file" 2>/dev/null | tail -1)"
  [ -n "$value" ] || return 1
  printf '%s\n' "$value"
}

valid_port() {
  [[ "$1" =~ ^[0-9]+$ ]] && (( 10#$1 >= 1024 && 10#$1 <= 65535 ))
}

valid_ipv4_cidr() {
  [[ "$1" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}/([0-9]|[12][0-9]|3[0-2])$ ]]
}

port_lines() {
  ss -H -ltnp 2>/dev/null | awk -v port=":$1" '$4 ~ port"$" {print}'
}

port_busy() {
  [ -n "$(port_lines "$1")" ]
}

suggest_port() {
  local p
  for p in $(seq 8990 8999); do
    if ! port_busy "$p"; then printf '%s\n' "$p"; return 0; fi
  done
  return 1
}

ufw_active() {
  command -v ufw >/dev/null 2>&1 || return 1
  LC_ALL=C ufw status 2>/dev/null | grep -q '^Status: active'
}

detect_lan_cidr() {
  local iface
  iface="$(ip -4 route show default 2>/dev/null | awk '{for (i=1;i<=NF;i++) if ($i=="dev") {print $(i+1); exit}}')"
  [ -n "$iface" ] || return 1
  ip -4 route show dev "$iface" proto kernel scope link 2>/dev/null \
    | awk '$1 ~ /^[0-9]+\./ && $1 ~ /\// && $1 !~ /^169\.254\./ {print $1; exit}'
}

ufw_rule_allows() {
  local source="$1" port="$2"
  LC_ALL=C ufw status 2>/dev/null \
    | grep -F "$port/tcp" \
    | grep -F "$source" >/dev/null 2>&1
}

ufw_managed_rule_present() {
  local source="$1" port="$2" comment="$3"
  command -v ufw >/dev/null 2>&1 || return 1
  LC_ALL=C ufw show added 2>/dev/null \
    | grep -F "$source" \
    | grep -F "port $port" \
    | grep -F "$comment" >/dev/null 2>&1
}

delete_managed_ufw_rule() {
  local source="$1" port="$2" comment="$3"
  [ -n "$source" ] && [ -n "$port" ] && [ -n "$comment" ] || return 0
  command -v ufw >/dev/null 2>&1 || { warn "UFW is unavailable; could not remove the old YWD-DMR-managed firewall rule."; return 1; }
  if ! ufw_managed_rule_present "$source" "$port" "$comment"; then
    return 0
  fi
  log "Removing old YWD-DMR-managed UFW rule: $source -> $port/tcp"
  if ufw --force delete allow proto tcp from "$source" to any port "$port" comment "$comment" >/dev/null; then
    return 0
  fi
  warn "Could not remove the old YWD-DMR-managed UFW rule. It was left in place."
  return 1
}

write_firewall_meta() {
  local managed="$1" source="$2" port="$3" status="${4:-ok}"
  cat > "$FIREWALL_META_FILE" <<EOF
# Managed by the YWD-DMR installer. Contains no credentials.
YWD_DMR_FIREWALL_PROVIDER=ufw
YWD_DMR_FIREWALL_MANAGED=$managed
YWD_DMR_FIREWALL_STATUS=$status
YWD_DMR_FIREWALL_SOURCE=$source
YWD_DMR_FIREWALL_PORT=$port
YWD_DMR_FIREWALL_COMMENT=$FIREWALL_COMMENT
EOF
  chown root:root "$FIREWALL_META_FILE"
  chmod 0644 "$FIREWALL_META_FILE"
}

# Existing installations keep their listener unless the user explicitly asks
# for a new port/access mode.
EXISTING_LISTEN=""
if [ -f "$ENV_FILE" ]; then
  EXISTING_LISTEN="$(sed -n 's/^YWD_DMR_LISTEN=//p' "$ENV_FILE" | tail -1)"
fi

if [ -z "$PORT" ] && [ -n "$EXISTING_LISTEN" ]; then
  PORT="${EXISTING_LISTEN##*:}"
fi
if [ -z "$BIND_ADDR" ] && [ -n "$EXISTING_LISTEN" ]; then
  BIND_ADDR="${EXISTING_LISTEN%:*}"
  [ "$BIND_ADDR" = 0.0.0.0 ] && LAN_TEST=1 || true
fi

if [ -n "$UFW_SOURCE_OVERRIDE" ] && ! valid_ipv4_cidr "$UFW_SOURCE_OVERRIDE"; then
  die "Invalid --ufw-source '$UFW_SOURCE_OVERRIDE'. Use IPv4 CIDR form such as 192.168.1.0/24."
fi

if [ -z "$PORT" ]; then PORT=$DEFAULT_PORT; fi
valid_port "$PORT" || die "Port '$PORT' is invalid. Choose a TCP port from 1024 through 65535."

# If our own installed service is active on the selected existing port, that is
# not a conflict. It will be stopped only after the replacement build is ready.
OWN_SERVICE_ON_PORT=0
if [ -n "$EXISTING_LISTEN" ] && [ "$PORT" = "${EXISTING_LISTEN##*:}" ] && systemctl is-active --quiet ywd-dmrd.service 2>/dev/null; then
  OWN_SERVICE_ON_PORT=1
fi

if port_busy "$PORT" && [ "$OWN_SERVICE_ON_PORT" -ne 1 ]; then
  warn "Port $PORT is already in use:"
  port_lines "$PORT" || true
  SUGGESTED="$(suggest_port || true)"
  if [ -n "$SUGGESTED" ] && [ -z "$EXISTING_LISTEN" ]; then
    if [ "$ASSUME_YES" -eq 1 ]; then
      log "Using free alternative port $SUGGESTED."
      PORT="$SUGGESTED"
    else
      printf 'Use suggested free port %s instead? [Y/n] ' "$SUGGESTED"
      IFS= read -r answer
      case "$answer" in n|N|no|NO) die "Choose another port and rerun with --port PORT." ;; *) PORT="$SUGGESTED" ;; esac
    fi
  else
    die "YWD-DMR will not stop or reconfigure the program using port $PORT. Choose another port."
  fi
fi

if [ -z "$BIND_ADDR" ]; then
  # Until authentication lands, safe default is loopback only.
  BIND_ADDR=127.0.0.1
fi

if [ "$BIND_ADDR" = 0.0.0.0 ]; then
  warn "LAN TEST MODE exposes the current unauthenticated development dashboard to your local network."
  warn "Do not forward port $PORT from your router and do not expose this service to the public internet."
  if [ "$ASSUME_YES" -ne 1 ]; then
    printf 'Continue with LAN test mode? [y/N] '
    IFS= read -r answer
    case "$answer" in y|Y|yes|YES) ;; *) log "Cancelled."; exit 0 ;; esac
  fi
fi

OLD_FW_PROVIDER="$(meta_value YWD_DMR_FIREWALL_PROVIDER "$FIREWALL_META_FILE" 2>/dev/null || true)"
OLD_FW_MANAGED="$(meta_value YWD_DMR_FIREWALL_MANAGED "$FIREWALL_META_FILE" 2>/dev/null || true)"
OLD_FW_SOURCE="$(meta_value YWD_DMR_FIREWALL_SOURCE "$FIREWALL_META_FILE" 2>/dev/null || true)"
OLD_FW_PORT="$(meta_value YWD_DMR_FIREWALL_PORT "$FIREWALL_META_FILE" 2>/dev/null || true)"
OLD_FW_COMMENT="$(meta_value YWD_DMR_FIREWALL_COMMENT "$FIREWALL_META_FILE" 2>/dev/null || true)"

FIREWALL_PLAN=none
FIREWALL_SOURCE=""
FIREWALL_SUMMARY="No firewall change"

if [ "$BIND_ADDR" = 0.0.0.0 ]; then
  if [ "$MANAGE_FIREWALL" -ne 1 ]; then
    FIREWALL_SUMMARY="Firewall changes disabled by --no-firewall"
  elif ufw_active; then
    FIREWALL_SOURCE="$UFW_SOURCE_OVERRIDE"
    [ -n "$FIREWALL_SOURCE" ] || FIREWALL_SOURCE="$(detect_lan_cidr || true)"

    if [ -z "$FIREWALL_SOURCE" ]; then
      FIREWALL_SUMMARY="UFW active; LAN subnet could not be detected, so no rule will be changed"
      warn "UFW is active, but YWD-DMR could not safely determine the LAN subnet."
      warn "No broad firewall rule will be created automatically. Use --ufw-source CIDR if needed."
    elif [ "$OLD_FW_PROVIDER" = ufw ] && [ "$OLD_FW_MANAGED" = 1 ] \
         && [ "$OLD_FW_SOURCE" = "$FIREWALL_SOURCE" ] && [ "$OLD_FW_PORT" = "$PORT" ] \
         && ufw_managed_rule_present "$FIREWALL_SOURCE" "$PORT" "${OLD_FW_COMMENT:-$FIREWALL_COMMENT}"; then
      FIREWALL_PLAN=keep-managed
      FIREWALL_SUMMARY="Keep YWD-DMR-managed UFW rule $FIREWALL_SOURCE -> $PORT/tcp"
    elif ufw_rule_allows "$FIREWALL_SOURCE" "$PORT"; then
      FIREWALL_PLAN=existing
      FIREWALL_SUMMARY="Use existing UFW rule $FIREWALL_SOURCE -> $PORT/tcp (not owned by YWD-DMR)"
    else
      if [ "$ASSUME_YES" -eq 1 ]; then
        FIREWALL_PLAN=create-managed
      else
        printf 'UFW is active. Allow YWD-DMR TCP port %s from LAN %s? [Y/n] ' "$PORT" "$FIREWALL_SOURCE"
        IFS= read -r answer
        case "$answer" in n|N|no|NO) FIREWALL_PLAN=declined ;; *) FIREWALL_PLAN=create-managed ;; esac
      fi
      if [ "$FIREWALL_PLAN" = create-managed ]; then
        FIREWALL_SUMMARY="Add YWD-DMR-managed UFW rule $FIREWALL_SOURCE -> $PORT/tcp"
      else
        FIREWALL_SUMMARY="UFW active; user declined firewall change"
      fi
    fi
  elif command -v ufw >/dev/null 2>&1; then
    FIREWALL_SUMMARY="UFW installed but inactive; no rule needed"
  elif command -v firewall-cmd >/dev/null 2>&1 && systemctl is-active --quiet firewalld 2>/dev/null; then
    FIREWALL_SUMMARY="firewalld detected; automatic firewall changes are not supported yet"
    warn "firewalld is active. YWD-DMR will not modify it automatically in this development build."
  else
    FIREWALL_SUMMARY="No active supported firewall detected"
  fi
fi

log ""
log "YWD-DMR installer preflight"
log "==========================="
log "Source:     $ROOT"
log "Listener:   $BIND_ADDR:$PORT"
log "Firewall:   $FIREWALL_SUMMARY"
log "App path:   $APP_DIR"
log "Config:     $CONFIG_DIR"
log "Data:       $DATA_DIR"
log ""

log "Building and testing YWD-DMR before changing the installed service..."
cd "$ROOT"
./scripts/build.sh

COMMIT="$(git rev-parse --short HEAD 2>/dev/null || printf unknown)"
RELEASE_ID="dev-${COMMIT}-$(date +%Y%m%d%H%M%S)"
RELEASE_DIR="$APP_DIR/releases/$RELEASE_ID"
OLD_CURRENT=""
[ -L "$APP_DIR/current" ] && OLD_CURRENT="$(readlink -f "$APP_DIR/current" || true)"
OLD_ENV_TMP=""
OLD_RUNTIME_TMP=""
if [ -f "$ENV_FILE" ]; then
  OLD_ENV_TMP="$(mktemp)"
  cp -a "$ENV_FILE" "$OLD_ENV_TMP"
fi
if [ -f "$PUBLIC_RUNTIME_FILE" ]; then
  OLD_RUNTIME_TMP="$(mktemp)"
  cp -a "$PUBLIC_RUNTIME_FILE" "$OLD_RUNTIME_TMP"
fi

# Never hijack an unrelated pre-existing account named ywd-dmr.
if getent passwd "$SERVICE_USER" >/dev/null 2>&1; then
  if [ ! -f "$OWNED_USER_MARKER" ] || ! grep -qx 'created-by-ywd-dmr-installer-v1' "$OWNED_USER_MARKER" 2>/dev/null; then
    die "A system account named '$SERVICE_USER' already exists but is not marked as YWD-DMR-owned. Refusing to modify it."
  fi
else
  log "Creating dedicated service account: $SERVICE_USER"
  useradd --system --user-group --home-dir "$DATA_DIR" --shell /usr/sbin/nologin "$SERVICE_USER"
fi

install -d -m 0755 "$APP_DIR/releases" "$CONFIG_DIR" "$LOG_DIR" "$BACKUP_DIR"
install -d -o "$SERVICE_USER" -g "$SERVICE_USER" -m 0750 "$DATA_DIR" "$DATA_DIR/plugins"
printf '%s\n' 'created-by-ywd-dmr-installer-v1' > "$OWNED_USER_MARKER"
chmod 0644 "$OWNED_USER_MARKER"

log "Installing release payload: $RELEASE_ID"
install -d -m 0755 "$RELEASE_DIR/bin" "$RELEASE_DIR/web" "$RELEASE_DIR/docs"
install -m 0755 "$ROOT/dist/ywd-dmrd" "$RELEASE_DIR/bin/ywd-dmrd"
cp -a "$ROOT/web/." "$RELEASE_DIR/web/"
cp -a "$ROOT/docs/." "$RELEASE_DIR/docs/"
chown -R root:root "$RELEASE_DIR"

if [ -n "$OLD_CURRENT" ] && [ -d "$OLD_CURRENT" ]; then
  ln -sfn "$OLD_CURRENT" "$APP_DIR/previous"
fi
ln -sfn "$RELEASE_DIR" "$APP_DIR/current.new"
mv -Tf "$APP_DIR/current.new" "$APP_DIR/current"

# Protected daemon configuration may contain secrets later and is intentionally
# not readable by normal local users.
cat > "$ENV_FILE" <<EOF
# Managed by the YWD-DMR installer. Protected daemon configuration.
YWD_DMR_LISTEN=$BIND_ADDR:$PORT
YWD_DMR_WEB_ROOT=/opt/ywd-dmr/current/web
YWD_DMR_DOCS_ROOT=/opt/ywd-dmr/current/docs
EOF
chown root:"$SERVICE_USER" "$ENV_FILE"
chmod 0640 "$ENV_FILE"

# Public runtime metadata contains no passwords/tokens. The maintenance CLI can
# safely read this without sudo to show the dashboard URL and run diagnostics.
cat > "$PUBLIC_RUNTIME_FILE" <<EOF
# Managed by the YWD-DMR installer. Non-sensitive runtime metadata only.
YWD_DMR_LISTEN=$BIND_ADDR:$PORT
EOF
chown root:root "$PUBLIC_RUNTIME_FILE"
chmod 0644 "$PUBLIC_RUNTIME_FILE"

install -m 0644 "$ROOT/packaging/systemd/ywd-dmrd.service" "$UNIT_FILE"
install -m 0755 "$ROOT/scripts/ywd-dmr" "$CLI_FILE"
install -m 0755 "$ROOT/scripts/uninstall.sh" "$UNINSTALL_FILE"

systemctl daemon-reload

# Stop our old instance only after the new build and files are ready, then check
# the selected port one last time for a race/conflict.
systemctl stop ywd-dmrd.service >/dev/null 2>&1 || true
sleep 1
if port_busy "$PORT"; then
  warn "Port $PORT became occupied before YWD-DMR could start."
  port_lines "$PORT" || true
  INSTALL_OK=0
else
  systemctl enable --now ywd-dmrd.service
  INSTALL_OK=1
  for _ in $(seq 1 15); do
    if curl -fsS --max-time 2 "http://127.0.0.1:$PORT/api/v1/health" >/dev/null 2>&1; then
      INSTALL_OK=1
      break
    fi
    INSTALL_OK=0
    sleep 1
  done
fi

if [ "$INSTALL_OK" -ne 1 ]; then
  warn "The new YWD-DMR installation failed its health check."
  systemctl disable --now ywd-dmrd.service >/dev/null 2>&1 || true
  if [ -n "$OLD_CURRENT" ] && [ -d "$OLD_CURRENT" ]; then
    warn "Restoring the previous installed release."
    ln -sfn "$OLD_CURRENT" "$APP_DIR/current.new"
    mv -Tf "$APP_DIR/current.new" "$APP_DIR/current"
    if [ -n "$OLD_ENV_TMP" ] && [ -f "$OLD_ENV_TMP" ]; then cp -a "$OLD_ENV_TMP" "$ENV_FILE"; fi
    if [ -n "$OLD_RUNTIME_TMP" ] && [ -f "$OLD_RUNTIME_TMP" ]; then cp -a "$OLD_RUNTIME_TMP" "$PUBLIC_RUNTIME_FILE"; fi
    systemctl enable --now ywd-dmrd.service >/dev/null 2>&1 || true
  else
    rm -f -- "$PUBLIC_RUNTIME_FILE"
  fi
  log ""
  log "Recent service log:"
  journalctl -u ywd-dmrd.service -n 40 --no-pager 2>/dev/null || true
  rm -f -- "$OLD_ENV_TMP" "$OLD_RUNTIME_TMP" 2>/dev/null || true
  exit 1
fi
rm -f -- "$OLD_ENV_TMP" "$OLD_RUNTIME_TMP" 2>/dev/null || true

# Firewall integration happens only after the daemon passes its health check.
# Existing equivalent rules are never claimed as YWD-DMR-owned.
FIREWALL_RESULT="$FIREWALL_SUMMARY"
if [ "$FIREWALL_PLAN" = create-managed ]; then
  if ufw --force allow proto tcp from "$FIREWALL_SOURCE" to any port "$PORT" comment "$FIREWALL_COMMENT" >/dev/null; then
    log "Added UFW rule for YWD-DMR LAN access: $FIREWALL_SOURCE -> $PORT/tcp"
    if [ "$OLD_FW_PROVIDER" = ufw ] && [ "$OLD_FW_MANAGED" = 1 ] \
       && { [ "$OLD_FW_SOURCE" != "$FIREWALL_SOURCE" ] || [ "$OLD_FW_PORT" != "$PORT" ]; }; then
      delete_managed_ufw_rule "$OLD_FW_SOURCE" "$OLD_FW_PORT" "${OLD_FW_COMMENT:-$FIREWALL_COMMENT}" || true
    fi
    write_firewall_meta 1 "$FIREWALL_SOURCE" "$PORT"
    FIREWALL_RESULT="UFW managed: $FIREWALL_SOURCE -> $PORT/tcp"
  else
    write_firewall_meta 0 "$FIREWALL_SOURCE" "$PORT" failed
    warn "YWD-DMR started successfully, but the UFW rule could not be added."
    warn "The service may not be reachable from other LAN devices until the firewall is configured."
    FIREWALL_RESULT="UFW rule creation FAILED; service itself is healthy"
  fi
elif [ "$FIREWALL_PLAN" = keep-managed ]; then
  write_firewall_meta 1 "$FIREWALL_SOURCE" "$PORT"
  FIREWALL_RESULT="UFW managed: $FIREWALL_SOURCE -> $PORT/tcp"
elif [ "$FIREWALL_PLAN" = existing ]; then
  if [ "$OLD_FW_PROVIDER" = ufw ] && [ "$OLD_FW_MANAGED" = 1 ]; then
    delete_managed_ufw_rule "$OLD_FW_SOURCE" "$OLD_FW_PORT" "${OLD_FW_COMMENT:-$FIREWALL_COMMENT}" || true
  fi
  write_firewall_meta 0 "$FIREWALL_SOURCE" "$PORT"
  FIREWALL_RESULT="UFW existing/user-owned: $FIREWALL_SOURCE -> $PORT/tcp"
else
  if [ "$OLD_FW_PROVIDER" = ufw ] && [ "$OLD_FW_MANAGED" = 1 ]; then
    if delete_managed_ufw_rule "$OLD_FW_SOURCE" "$OLD_FW_PORT" "${OLD_FW_COMMENT:-$FIREWALL_COMMENT}"; then
      rm -f -- "$FIREWALL_META_FILE"
    fi
  elif [ -f "$FIREWALL_META_FILE" ]; then
    rm -f -- "$FIREWALL_META_FILE"
  fi
fi

HOST_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
log ""
log "YWD-DMR installation complete"
log "============================="
log "Release:  $RELEASE_ID"
log "Service:  active"
log "Port:     $PORT"
log "Firewall: $FIREWALL_RESULT"
if [ "$BIND_ADDR" = 0.0.0.0 ] && [ -n "$HOST_IP" ]; then
  log "Open:     http://$HOST_IP:$PORT/"
  log ""
  warn "This is LAN TEST MODE with no dashboard authentication yet."
else
  log "Open:     http://127.0.0.1:$PORT/"
fi
log ""
log "Useful commands:"
log "  ywd-dmr status"
log "  ywd-dmr diagnose"
log "  ywd-dmr logs"
log "  sudo ywd-dmr uninstall"