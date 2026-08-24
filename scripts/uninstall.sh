#!/usr/bin/env bash
# YWD-DMR safe uninstaller
#
# Removes only YWD-DMR-owned paths/rules. Shared OS packages, unrelated firewall
# rules, and unrelated services are never removed or reconfigured.

set -u

PURGE_DATA=0
NO_BACKUP=0
ASSUME_YES=0
DRY_RUN=0

APP_DIR="/opt/ywd-dmr"
CONFIG_DIR="/etc/ywd-dmr"
DATA_DIR="/var/lib/ywd-dmr"
LOG_DIR="/var/log/ywd-dmr"
BACKUP_DIR="/var/backups/ywd-dmr"
UNIT_FILE="/etc/systemd/system/ywd-dmrd.service"
UNIT_DROPIN_DIR="/etc/systemd/system/ywd-dmrd.service.d"
CLI_FILE="/usr/local/bin/ywd-dmr"
UNINSTALL_FILE="/usr/local/sbin/ywd-dmr-uninstall"
FIREWALL_META_FILE="${CONFIG_DIR}/firewall.conf"
OWNED_USER_MARKER="${CONFIG_DIR}/install-owned-user"
OWNED_USER="ywd-dmr"

usage() {
  cat <<'EOF'
YWD-DMR safe uninstaller

Usage:
  sudo ywd-dmr uninstall [options]
  sudo /usr/local/sbin/ywd-dmr-uninstall [options]
  sudo bash scripts/uninstall.sh [options]

Default behavior removes the YWD-DMR application/service but preserves user
configuration, local vocoder plugins, history, and backups for reinstall.
Installer-owned firewall rules are removed in both normal and full removal.

Options:
  --purge-data   Also remove YWD-DMR configuration, data, plugins, logs, and
                 YWD-DMR-managed backups. A safety backup is created first.
  --no-backup    With --purge-data, do not create the safety backup.
  --yes          Skip the interactive confirmation.
  --dry-run      Show what would happen without deleting anything.
  -h, --help     Show this help.

For a completely clean removal with no retained backup:
  sudo ywd-dmr uninstall --purge-data --no-backup
EOF
}

log() { printf '%s\n' "$*"; }
warn() { printf 'WARNING: %s\n' "$*" >&2; }

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
    return 0
  fi
  "$@"
}

meta_value() {
  local key="$1" file="$2" value
  [ -f "$file" ] || return 1
  value="$(sed -n "s/^${key}=//p" "$file" 2>/dev/null | tail -1)"
  [ -n "$value" ] || return 1
  printf '%s\n' "$value"
}

remove_tree() {
  case "$1" in
    "$APP_DIR"|"$CONFIG_DIR"|"$DATA_DIR"|"$LOG_DIR"|"$BACKUP_DIR"|"$UNIT_DROPIN_DIR") ;;
    *) warn "Refusing to remove unexpected directory: $1"; return 1 ;;
  esac
  [ -e "$1" ] || return 0
  log "Removing $1"
  run rm -rf --one-file-system -- "$1"
}

remove_file() {
  case "$1" in
    "$UNIT_FILE"|"$CLI_FILE"|"$UNINSTALL_FILE"|"$FIREWALL_META_FILE") ;;
    *) warn "Refusing to remove unexpected file: $1"; return 1 ;;
  esac
  [ -e "$1" ] || [ -L "$1" ] || return 0
  log "Removing $1"
  run rm -f -- "$1"
}

ufw_managed_rule_present() {
  local source="$1" port="$2" comment="$3"
  command -v ufw >/dev/null 2>&1 || return 1
  LC_ALL=C ufw show added 2>/dev/null \
    | grep -F "$source" \
    | grep -F "port $port" \
    | grep -F "$comment" >/dev/null 2>&1
}

remove_firewall_integration() {
  [ -f "$FIREWALL_META_FILE" ] || return 0

  local provider managed source port comment
  provider="$(meta_value YWD_DMR_FIREWALL_PROVIDER "$FIREWALL_META_FILE" 2>/dev/null || true)"
  managed="$(meta_value YWD_DMR_FIREWALL_MANAGED "$FIREWALL_META_FILE" 2>/dev/null || true)"
  source="$(meta_value YWD_DMR_FIREWALL_SOURCE "$FIREWALL_META_FILE" 2>/dev/null || true)"
  port="$(meta_value YWD_DMR_FIREWALL_PORT "$FIREWALL_META_FILE" 2>/dev/null || true)"
  comment="$(meta_value YWD_DMR_FIREWALL_COMMENT "$FIREWALL_META_FILE" 2>/dev/null || true)"

  if [ "$managed" = 1 ]; then
    if [ "$provider" != ufw ]; then
      warn "Firewall metadata says YWD-DMR owns a '$provider' rule, but this uninstaller only knows how to safely remove UFW rules."
      return 1
    fi
    if ! command -v ufw >/dev/null 2>&1; then
      warn "UFW is unavailable; the installer-owned firewall rule could not be removed."
      return 1
    fi
    if [ -z "$source" ] || [ -z "$port" ] || [ -z "$comment" ]; then
      warn "Firewall ownership metadata is incomplete. Refusing to guess which rule to remove."
      return 1
    fi

    if ufw_managed_rule_present "$source" "$port" "$comment"; then
      log "Removing YWD-DMR-managed UFW rule: $source -> $port/tcp"
      if ! run ufw --force delete allow proto tcp from "$source" to any port "$port" comment "$comment" >/dev/null; then
        warn "Could not remove the YWD-DMR-managed UFW rule. It was left in place."
        return 1
      fi
    else
      log "YWD-DMR-managed UFW rule is no longer present; no firewall deletion is needed."
    fi
  else
    log "Existing firewall rule was not created by YWD-DMR; leaving it untouched."
  fi

  remove_file "$FIREWALL_META_FILE"
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --purge-data) PURGE_DATA=1 ;;
    --no-backup) NO_BACKUP=1 ;;
    --yes) ASSUME_YES=1 ;;
    --dry-run) DRY_RUN=1 ;;
    -h|--help) usage; exit 0 ;;
    *) warn "Unknown option: $1"; usage; exit 2 ;;
  esac
  shift
done

if [ "$NO_BACKUP" -eq 1 ] && [ "$PURGE_DATA" -ne 1 ]; then
  warn "--no-backup only makes sense with --purge-data."
  exit 2
fi
if [ "$(id -u)" -ne 0 ]; then
  warn "Please run this uninstaller with sudo/root privileges."
  exit 1
fi

OWNED_USER_WAS_CREATED=0
if [ -f "$OWNED_USER_MARKER" ] && grep -qx 'created-by-ywd-dmr-installer-v1' "$OWNED_USER_MARKER" 2>/dev/null; then
  OWNED_USER_WAS_CREATED=1
fi

FW_PROVIDER="$(meta_value YWD_DMR_FIREWALL_PROVIDER "$FIREWALL_META_FILE" 2>/dev/null || true)"
FW_MANAGED="$(meta_value YWD_DMR_FIREWALL_MANAGED "$FIREWALL_META_FILE" 2>/dev/null || true)"
FW_SOURCE="$(meta_value YWD_DMR_FIREWALL_SOURCE "$FIREWALL_META_FILE" 2>/dev/null || true)"
FW_PORT="$(meta_value YWD_DMR_FIREWALL_PORT "$FIREWALL_META_FILE" 2>/dev/null || true)"

cat <<EOF

YWD-DMR Safe Uninstaller
========================
Application:   $APP_DIR
Configuration: $CONFIG_DIR
Data/plugins:  $DATA_DIR
Logs:          $LOG_DIR
Backups:       $BACKUP_DIR

EOF

if [ "$FW_MANAGED" = 1 ]; then
  log "Firewall:      YWD-DMR-managed ${FW_PROVIDER:-firewall} rule ${FW_SOURCE:-unknown} -> ${FW_PORT:-unknown}/tcp will be removed."
elif [ -f "$FIREWALL_META_FILE" ]; then
  log "Firewall:      existing/user-owned rule will be left untouched."
else
  log "Firewall:      no installer-owned firewall rule recorded."
fi

if [ "$PURGE_DATA" -eq 1 ]; then
  warn "FULL REMOVAL selected. Configuration, local plugins, history, logs, and YWD-DMR-managed backups will be removed."
else
  log "Software-only removal selected. Configuration, local plugins, history, and backups will be preserved."
fi
[ "$DRY_RUN" -eq 1 ] && log "Dry-run mode: nothing will be deleted."

if [ "$ASSUME_YES" -ne 1 ]; then
  if [ "$PURGE_DATA" -eq 1 ]; then
    printf 'Type REMOVE YWD-DMR to continue: '
    IFS= read -r answer
    [ "$answer" = "REMOVE YWD-DMR" ] || { log "Cancelled."; exit 0; }
  else
    printf 'Continue with software-only removal? [y/N] '
    IFS= read -r answer
    case "$answer" in y|Y|yes|YES) ;; *) log "Cancelled."; exit 0 ;; esac
  fi
fi

log "Stopping YWD-DMR service if present..."
if command -v systemctl >/dev/null 2>&1; then
  run systemctl disable --now ywd-dmrd.service >/dev/null 2>&1 || true
fi

# Firewall rules are operational integration, not user station data. Remove an
# installer-owned rule even during software-only removal. Never remove an
# equivalent rule that existed before YWD-DMR or was created by the user.
remove_firewall_integration || warn "Firewall cleanup was incomplete; review the warning above."

SAFETY_BACKUP=""
if [ "$PURGE_DATA" -eq 1 ] && [ "$NO_BACKUP" -ne 1 ]; then
  stamp="$(date +%Y%m%d-%H%M%S)"
  SAFETY_BACKUP="/var/backups/ywd-dmr-uninstall-${stamp}.tar.gz"
  backup_items=""
  [ -d "$CONFIG_DIR" ] && backup_items="$backup_items etc/ywd-dmr"
  [ -d "$DATA_DIR" ] && backup_items="$backup_items var/lib/ywd-dmr"
  [ -d "$LOG_DIR" ] && backup_items="$backup_items var/log/ywd-dmr"

  if [ -n "$backup_items" ]; then
    log "Creating safety backup: $SAFETY_BACKUP"
    if [ "$DRY_RUN" -eq 1 ]; then
      log "+ tar -C / -czf $SAFETY_BACKUP$backup_items"
    else
      # shellcheck disable=SC2086
      tar -C / -czf "$SAFETY_BACKUP" $backup_items || {
        warn "Safety backup failed. Nothing has been purged."
        exit 1
      }
      chmod 600 "$SAFETY_BACKUP" 2>/dev/null || true
    fi
  else
    log "No configuration/data/log directories were found to back up."
  fi
fi

remove_file "$UNIT_FILE"
remove_tree "$UNIT_DROPIN_DIR"
remove_file "$CLI_FILE"
remove_file "$UNINSTALL_FILE"
remove_tree "$APP_DIR"

if [ "$PURGE_DATA" -eq 1 ]; then
  remove_tree "$CONFIG_DIR"
  remove_tree "$DATA_DIR"
  remove_tree "$LOG_DIR"
  remove_tree "$BACKUP_DIR"
fi

if command -v systemctl >/dev/null 2>&1; then
  run systemctl daemon-reload >/dev/null 2>&1 || true
  run systemctl reset-failed ywd-dmrd.service >/dev/null 2>&1 || true
fi

# The service account is retained during software-only removal because preserved
# data still belongs to it. On full purge, remove it only when the root-owned
# installer marker proves YWD-DMR created it and its profile still looks safe.
if [ "$PURGE_DATA" -eq 1 ] && [ "$OWNED_USER_WAS_CREATED" -eq 1 ] && command -v getent >/dev/null 2>&1; then
  if getent passwd "$OWNED_USER" >/dev/null 2>&1; then
    passwd_line="$(getent passwd "$OWNED_USER")"
    user_home="$(printf '%s' "$passwd_line" | cut -d: -f6)"
    user_shell="$(printf '%s' "$passwd_line" | cut -d: -f7)"
    case "$user_home:$user_shell" in
      "/var/lib/ywd-dmr:/usr/sbin/nologin"|"/nonexistent:/usr/sbin/nologin"|"/var/lib/ywd-dmr:/bin/false"|"/nonexistent:/bin/false")
        log "Removing installer-created service account: $OWNED_USER"
        run userdel "$OWNED_USER" >/dev/null 2>&1 || warn "Could not remove service account $OWNED_USER; it may still be in use."
        ;;
      *) warn "Service account $OWNED_USER does not match the installer-owned profile; leaving it untouched." ;;
    esac
  fi
fi

log ""
log "YWD-DMR uninstall finished."
if [ "$PURGE_DATA" -eq 1 ]; then
  if [ -n "$SAFETY_BACKUP" ]; then
    log "Safety backup retained at: $SAFETY_BACKUP"
    log "Delete that file manually only when you are sure you no longer need it."
  elif [ "$NO_BACKUP" -eq 1 ]; then
    log "Full removal was performed without creating a safety backup."
  fi
else
  log "Your YWD-DMR configuration/data and service account were preserved for a future reinstall."
fi
log "Shared Linux packages, unrelated firewall rules, and unrelated services were intentionally left untouched."