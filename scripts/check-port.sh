#!/usr/bin/env bash
set -Eeuo pipefail

PORT="${1:-8989}"

if ! [[ "$PORT" =~ ^[0-9]+$ ]] || (( PORT < 1024 || PORT > 65535 )); then
  echo "Invalid port: $PORT"
  echo "Choose a TCP port from 1024 through 65535."
  exit 1
fi

if ! command -v ss >/dev/null 2>&1; then
  echo "Cannot check port $PORT because the 'ss' command is unavailable."
  echo "Install iproute2 or let the YWD-DMR installer perform the check."
  exit 1
fi

MATCHES="$(ss -H -ltnp 2>/dev/null | awk -v port=":$PORT" '$4 ~ port"$" {print}')"

if [[ -n "$MATCHES" ]]; then
  echo "Port $PORT is already in use."
  echo
  echo "$MATCHES"
  echo
  echo "YWD-DMR will not stop or reconfigure the existing service."
  exit 2
fi

echo "Port $PORT is available for YWD-DMR."
