#!/usr/bin/env bash
# Creates temporary schools, unique to this run, in a disposable CI stack.
set -euo pipefail
case "${EDUPILOT_E2E_MODE:-native}" in
  native) seed_args=(--native); default_url=http://127.0.0.1:5173 ;;
  docker) seed_args=(--docker "${DB:-sqlite}"); default_url=http://127.0.0.1:8080 ;;
  *) echo "EDUPILOT_E2E_MODE must be native or docker" >&2; exit 2 ;;
esac
export E2E_BASE_URL="${E2E_BASE_URL:-$default_url}"
for attempt in {1..60}; do
  if curl --max-time 3 --fail --silent "$E2E_BASE_URL/api/v1/ready" >/dev/null; then break; fi
  if [ "$attempt" = 60 ]; then echo "Web proxy did not become ready" >&2; exit 1; fi
  sleep 2
done

# Keep browser journeys independent of the real per-account sign-in limit.
for slot in AUTH A B ROSTER PREFS PREFS_OTHER NAV NAV_OTHER; do
  school="e2e-$(openssl rand -hex 8)"
  email="admin@${school}.example"
  password=$(openssl rand -hex 24)
  if [[ -n "${GITHUB_ACTIONS:-}" ]]; then echo "::add-mask::$password"; fi
  export "E2E_SCHOOL_${slot}=$school" "E2E_EMAIL_${slot}=$email" "E2E_PASSWORD_${slot}=$password"
  EDUPILOT_SEED_SCHOOL="$school" EDUPILOT_SEED_EMAIL="$email" EDUPILOT_SEED_PASSWORD="$password" \
    bash scripts/seed-admin.sh "${seed_args[@]}"
done
(cd tests/e2e && npm run test --workspaces=false -- "$@")
