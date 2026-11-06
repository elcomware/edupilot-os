#!/usr/bin/env bash
# Verify the installed Linux GUI opens its branded window on a disposable display.
set -euo pipefail
if [[ $# != 1 || ! -x "$1" ]]; then
  echo "Usage: check-desktop-window.sh PATH_TO_EXECUTABLE" >&2
  exit 2
fi
export EDUPILOT_API_ORIGIN=http://127.0.0.1:3000
export GDK_BACKEND=x11
xvfb-run -a dbus-run-session -- bash -eu -o pipefail -s -- "$1" <<'WINDOW'
"$1" &
app_pid=$!
cleanup() {
  kill -TERM "$app_pid" 2>/dev/null || true
  wait "$app_pid" || true
}
trap cleanup EXIT
for attempt in {1..60}; do
  if ! kill -0 "$app_pid" 2>/dev/null; then
    echo "Desktop process exited before its window appeared." >&2
    exit 1
  fi
  if xdotool search --all --onlyvisible --pid "$app_pid" --name 'EduPilot OS$' >/dev/null 2>&1; then
    echo "Opened the EduPilot OS desktop window."
    exit 0
  fi
  sleep 1
done
echo "Timed out waiting for the EduPilot OS desktop window." >&2
exit 1
WINDOW
