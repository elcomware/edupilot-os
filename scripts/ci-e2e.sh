#!/usr/bin/env bash
# Creates temporary schools, unique to this run, in a disposable CI stack.
set -euo pipefail
export E2E_BASE_URL="${E2E_BASE_URL:-http://127.0.0.1:8080}"
for attempt in {1..60}; do
  if curl --max-time 3 --fail --silent "$E2E_BASE_URL/api/v1/ready" >/dev/null; then break; fi
  if [ "$attempt" = 60 ]; then echo "Web proxy did not become ready" >&2; exit 1; fi
  sleep 2
done

for slot in A B; do
  school="e2e-$(openssl rand -hex 8)"
  email="admin@${school}.example"
  password=$(openssl rand -hex 24)
  if [[ -n "${GITHUB_ACTIONS:-}" ]]; then echo "::add-mask::$password"; fi
  export "E2E_SCHOOL_${slot}=$school" "E2E_EMAIL_${slot}=$email" "E2E_PASSWORD_${slot}=$password"
  EDUPILOT_SEED_SCHOOL="$school" EDUPILOT_SEED_EMAIL="$email" EDUPILOT_SEED_PASSWORD="$password" \
    bash scripts/seed-admin.sh "${DB:-sqlite}"
done
make e2e
