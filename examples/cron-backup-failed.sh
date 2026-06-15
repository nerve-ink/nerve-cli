#!/usr/bin/env bash
set -euo pipefail

export NERVE_DSN="nerve://TOKEN:SENDER_KEY@api.nerve.ink"

if ! /opt/jobs/backup.sh; then
  printf 'backup failed\nhost: %s\n' "$(hostname)" \
    | nerve send --severity alert --title "Backup failed"
  exit 1
fi
