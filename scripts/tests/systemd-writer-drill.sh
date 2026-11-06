#!/usr/bin/env bash
# Dedicated disposable systemd runner only. Create two unique, harmless writers.
set -euo pipefail
prefix="edupilot-drill-$(cat /proc/sys/kernel/random/uuid)"
units=("$prefix-api.service" "$prefix-worker.service")
dir=/run/systemd/system
check() { python3 scripts/verify-systemd-writer-fence.py "${units[@]}"; }
cleanup() {
  for unit in "${units[@]}"; do
    sudo systemctl stop "$unit" 2>/dev/null || true
    sudo rm -f "/etc/systemd/system/$unit" "$dir/$unit"
  done
  sudo systemctl daemon-reload
}
trap cleanup EXIT
for unit in "${units[@]}"; do
  printf '[Service]\nExecStart=/usr/bin/sleep infinity\n' | sudo tee "$dir/$unit" >/dev/null
done
sudo systemctl daemon-reload
sudo systemctl start "${units[@]}"
if check; then exit 1; fi
sudo systemctl stop "${units[@]}"
if check; then exit 1; fi
sudo systemctl mask "${units[0]}"
if check; then exit 1; fi
sudo systemctl mask "${units[1]}"
check
for unit in "${units[@]}"; do
  if sudo systemctl start "$unit"; then exit 1; fi
done
check
