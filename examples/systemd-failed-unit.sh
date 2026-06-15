#!/usr/bin/env bash
set -euo pipefail

export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"

unit="${1:?usage: systemd-failed-unit.sh UNIT_NAME}"

if ! systemctl is-active --quiet "$unit"; then
  printf 'systemd unit failed\nunit: %s\nhost: %s\n' "$unit" "$(hostname)" \
    | nerve send --severity critical --title "systemd failed"
  exit 1
fi
