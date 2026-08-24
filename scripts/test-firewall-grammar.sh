#!/usr/bin/env bash
set -Eeuo pipefail

grep -Fq 'ufw allow proto tcp from "$FIREWALL_SOURCE" to any port "$PORT" comment "$FIREWALL_COMMENT"' scripts/install.sh
grep -Fq 'ufw delete allow proto tcp from "$source" to any port "$port" comment "$comment"' scripts/install.sh
grep -Fq 'ufw delete allow proto tcp from "$source" to any port "$port" comment "$comment"' scripts/uninstall.sh

echo "OK: managed UFW command grammar"
