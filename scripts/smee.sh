#!/bin/sh
# Forwards webhook deliveries from a smee.io channel to the local server.
# Usage: scripts/smee.sh <smee_url> [local_port]

if [ -z "$1" ]; then
  echo "usage: $0 <smee_url> [local_port]" >&2
  exit 1
fi

smee --url "$1" --path "${FREEGO_WEBHOOK_ROUTE:-/webhook}" --port "${2:-${FREEGO_WEBHOOK_PORT:-8080}}" &
