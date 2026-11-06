#!/usr/bin/env bash
# Disposable hosted-runner fixture. Publish only the three explicitly named PNGs.
set -euo pipefail
platform=$1
case "$platform" in android|ios) ;; *) exit 2 ;; esac
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test ! -e .env && test ! -e database/edupilot.sqlite || exit 1
export APP_HOST=127.0.0.1 APP_PORT=3000 DB_CONNECTION=sqlite DB_DATABASE=database/edupilot.sqlite
export EDUPILOT_DEDICATED_DATABASES='[]' EDUPILOT_DOCUMENTS_DRIVER=local EDUPILOT_ENABLED_MODULES=''
export EDUPILOT_SEED_SCHOOL="demo-$platform" EDUPILOT_SEED_EMAIL=registrar@example.test
EDUPILOT_SEED_PASSWORD=$(openssl rand -hex 12)
export EDUPILOT_SEED_PASSWORD
echo "::add-mask::$EDUPILOT_SEED_PASSWORD"
python3 scripts/setup.py
go run ./cmd/migrate
go run ./cmd/seed-admin
go build -o "$RUNNER_TEMP/mobile-api" .
"$RUNNER_TEMP/mobile-api" >"$RUNNER_TEMP/mobile-api.log" 2>&1 &
api_pid=$!
tls="$RUNNER_TEMP/mobile-tls"
socat "OPENSSL-LISTEN:4443,bind=127.0.0.1,reuseaddr,fork,cert=$tls/server.pem,key=$tls/server.key,verify=0" TCP:127.0.0.1:3000 &
proxy_pid=$!
trap 'kill "$proxy_pid" "$api_pid" 2>/dev/null || true' EXIT
for _ in {1..60}; do
  if curl -fsS --cacert "$tls/ca.pem" https://localhost:4443/api/v1/ready >/dev/null 2>&1; then break; fi
  sleep 1
done
curl -fsS --cacert "$tls/ca.pem" https://localhost:4443/api/v1/ready >/dev/null
if [[ "$platform" = android ]]; then
  adb reverse tcp:4443 tcp:4443
  adb install clients/mobile/android/app/build/outputs/apk/release/app-release.apk
fi
output="$RUNNER_TEMP/mobile-screens"
mkdir -p "$output"
maestro test -e "PLATFORM=$platform" -e "SCHOOL=$EDUPILOT_SEED_SCHOOL" -e "PASSWORD=$EDUPILOT_SEED_PASSWORD" \
  --test-output-dir "$output" tests/mobile/flow.yml
python3 - "$platform" "$output" <<'PY'
import sqlite3, sys
from pathlib import Path
platform, directory = sys.argv[1:]
with sqlite3.connect("file:database/edupilot.sqlite?mode=ro", uri=True) as db:
    rows = db.execute("SELECT school_id, given_name, family_name, status FROM students_records").fetchall()
    sessions = db.execute("SELECT count(*) FROM identity_sessions").fetchone()[0]
if rows != [("demo-" + platform, "Jamie", "Demo", "active")] or sessions:
    raise SystemExit("Unexpected student or remaining session in the fixture database")
for state in ("signin", "directory", "students"):
    path = Path(directory) / "takeScreenshot" / f"{platform}-{state}.png"
    if not path.is_file() or path.stat().st_size < 1024:
        raise SystemExit(f"Missing screenshot: {state}")
print(f"{platform}: native sign-in and registration verified, one active school record, zero remaining sessions.")
PY
