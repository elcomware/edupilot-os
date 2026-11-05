#!/usr/bin/env bash
# Reads credentials from environment (CI) or prompts with hidden password input.
set -euo pipefail
: "${EDUPILOT_SEED_SCHOOL:=}" "${EDUPILOT_SEED_EMAIL:=}" "${EDUPILOT_SEED_PASSWORD:=}"
[[ -n "$EDUPILOT_SEED_SCHOOL" ]] || read -rp "School ID: " EDUPILOT_SEED_SCHOOL
[[ -n "$EDUPILOT_SEED_EMAIL" ]] || read -rp "Administrator email: " EDUPILOT_SEED_EMAIL
if [[ -z "$EDUPILOT_SEED_PASSWORD" ]]; then
  read -rsp "Administrator password (12–72 bytes): " EDUPILOT_SEED_PASSWORD
  printf '\n'
fi
export EDUPILOT_SEED_SCHOOL EDUPILOT_SEED_EMAIL EDUPILOT_SEED_PASSWORD
if [[ "${1:-}" == "--native" ]]; then
  exec go run ./cmd/seed-admin
fi
db="${1:-sqlite}"
compose=(-f docker-compose.yml)
case "$db" in
  sqlite) ;;
  postgres) compose+=(-f docker/compose.postgres.yml) ;;
  *) echo "Expected sqlite or postgres" >&2; exit 2 ;;
esac
exec docker compose "${compose[@]}" run --rm \
  -e EDUPILOT_SEED_SCHOOL -e EDUPILOT_SEED_EMAIL -e EDUPILOT_SEED_PASSWORD seed-admin
