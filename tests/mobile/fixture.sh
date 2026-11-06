#!/usr/bin/env bash
# Disposable hosted-runner fixture. Publish only the three explicitly named PNGs.
set -euo pipefail
platform=$1
case "$platform" in android|ios) ;; *) exit 2 ;; esac
test "${RUNNER_ENVIRONMENT:-}" = github-hosted
test ! -e .env && test ! -e database/edupilot.sqlite || exit 1
export APP_HOST=127.0.0.1 APP_PORT=3000 DB_CONNECTION=sqlite DB_DATABASE=database/edupilot.sqlite
export EDUPILOT_DEDICATED_DATABASES='[]' EDUPILOT_DOCUMENTS_DRIVER=local EDUPILOT_ENABLED_MODULES='tenancy,identity,students,guardians,enrollment,attendance,documents'
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
# Check the same API catalog independently, then revoke this preflight session.
python3 - <<'PREFLIGHT'
import json, os
from urllib.request import Request, urlopen
base = "http://127.0.0.1:3000/api/v1"
credentials = {key: os.environ["EDUPILOT_SEED_" + key.upper()] for key in ("school", "email", "password")}
with urlopen(Request(base + "/session", data=json.dumps(credentials).encode(), headers={"Content-Type": "application/json"}), timeout=10) as r:
    auth = {"Authorization": "Bearer " + json.load(r)["token"]}
try:
    with urlopen(Request(base + "/modules", headers=auth), timeout=10) as r:
        modules = [m["key"] for m in json.load(r)["items"]]
    if set(modules) != set(os.environ["EDUPILOT_ENABLED_MODULES"].split(",")):
        raise SystemExit("Fixture module catalog mismatch: " + ", ".join(modules))
finally:
    with urlopen(Request(base + "/session", headers=auth, method="DELETE"), timeout=10):
        pass
PREFLIGHT
if [[ "$platform" = android ]]; then
  adb shell input keyevent KEYCODE_WAKEUP
  adb shell wm dismiss-keyguard
  adb reverse tcp:4443 tcp:4443
  adb install clients/mobile/android/app/build/outputs/apk/release/app-release.apk
fi
output="$RUNNER_TEMP/mobile-screens"
mkdir -p "$output"
root=$PWD
driver="$RUNNER_TEMP/mobile-maestro"
mkdir "$driver"
result=0
if [[ "$platform" = ios ]]; then
  (cd "$driver" && maestro test -e "SCHOOL=$EDUPILOT_SEED_SCHOOL" -e "PASSWORD=$EDUPILOT_SEED_PASSWORD" \
    --test-output-dir "$driver" "$root/tests/mobile/iphone-login.yml")
  cp -R tests/mobile/iphone "$driver/project"
  (cd "$driver/project" && xcodegen generate &&
    xcodebuild test -project MobileProof.xcodeproj -scheme MobileProof \
      -parallel-testing-enabled NO -destination "id=$IPHONE_UDID" \
      -resultBundlePath "$RUNNER_TEMP/proof.xcresult" -derivedDataPath "$RUNNER_TEMP/mobile-uitests" > "$RUNNER_TEMP/mobile-xctest.log" 2>&1) || result=$?
  # XCTest logs can contain text entered into fields. Only surface assertion/build failure lines.
  if [[ "$result" != 0 ]]; then
    grep -E 'error:|Waited for|Timed out waiting|Missing field|TEST FAILED|failure:' "$RUNNER_TEMP/mobile-xctest.log" | tail -30 || true
    xcrun xcresulttool get test-results summary --path "$RUNNER_TEMP/proof.xcresult" | python3 -c 'import json,sys; print([f.get("failureText","")[:800] for f in json.load(sys.stdin).get("testFailures",[])])' || true
  fi
  appdata=$(xcrun simctl get_app_container "$IPHONE_UDID" org.edupilot.proof.tests.xctrunner data)
  for state in directory students; do
    if [[ -f "$appdata/Documents/edupilot-$state.png" ]]; then
      cp "$appdata/Documents/edupilot-$state.png" "$driver/"
    fi
  done
else
  (cd "$driver" && maestro test -e "SCHOOL=$EDUPILOT_SEED_SCHOOL" -e "PASSWORD=$EDUPILOT_SEED_PASSWORD" \
    --test-output-dir "$driver" "$root/tests/mobile/flow.yml") || result=$?
fi
python3 - "$platform" "$driver" "$output" "$result" <<'PYTHON'
import shutil, sqlite3, sys
from pathlib import Path

platform, driver, output, result = sys.argv[1:]
# Keep test logs and debug screenshots private. Only the three explicitly captured
# states can enter a published artifact.
roots = [Path(driver)]
for state in ("signin", "directory", "students"):
    matches = {p.resolve() for root in roots for p in root.rglob(f"edupilot-{state}.png")}
    if len(matches) == 1:
        source = matches.pop()
        if source.stat().st_size < 1024 or source.read_bytes()[:8] != b"\x89PNG\r\n\x1a\n":
            raise SystemExit(f"Invalid native screenshot: {state}")
        shutil.copyfile(source, Path(output) / f"{platform}-{state}.png")
    elif result == "0" or matches:
        raise SystemExit(f"Expected one named native screenshot for {state}, found {len(matches)}")
if result != "0":
    raise SystemExit(int(result))
with sqlite3.connect("file:database/edupilot.sqlite?mode=ro", uri=True) as db:
    rows = db.execute("SELECT school_id, given_name, family_name, status FROM students_records").fetchall()
    sessions = db.execute("SELECT count(*) FROM identity_sessions").fetchone()[0]
if rows != [("demo-" + platform, "Jamie", "Demo", "active")] or sessions:
    raise SystemExit("Unexpected student or remaining session in the fixture database")
print(f"{platform}: native sign-in and registration verified, one active school record, zero remaining sessions.")
PYTHON
