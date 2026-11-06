#!/usr/bin/env bash
# Run only on a disposable Linux test host with loop and mount privileges.
set -euo pipefail
dir=$(mktemp -d)
volume="$dir/volume"
alias_dir="$dir/alias"
loop=''
cleanup() {
  if mountpoint -q "$volume/docs/nested"; then sudo umount "$volume/docs/nested" || return 1; fi
  if mountpoint -q "$alias_dir"; then sudo umount "$alias_dir" || return 1; fi
  if mountpoint -q "$volume"; then sudo umount "$volume" || return 1; fi
  if [[ -n "$loop" ]]; then sudo losetup -d "$loop" || return 1; fi
  rm -rf "$dir"
}
trap cleanup EXIT
mkdir "$volume" "$alias_dir"
truncate -s 32M "$dir/image"
loop=$(sudo losetup --find --show "$dir/image")
sudo mkfs.ext4 -q "$loop"
sudo mount "$loop" "$volume"
sudo mkdir -p "$volume/docs/nested"
sudo sh -c 'printf synthetic > "$1/record"' sh "$volume/docs"
sudo mount --bind "$volume" "$alias_dir"
check() { python3 scripts/verify-local-document-fence.py --root "$1/docs" --device "$loop"; }
reject() { if check "$1"; then echo 'Unexpected acceptance' >&2; exit 1; fi; }
reject "$volume"
sudo mount -o remount,bind,ro "$alias_dir"
reject "$alias_dir"
sudo touch "$volume/docs/via-writable-view"
sudo umount "$alias_dir"
sudo mount -o remount,ro "$volume"
check "$volume"
sudo mount --bind "$volume" "$alias_dir"
for path in "$alias_dir/docs" "$volume/docs"; do
  if sudo touch "$path/new" || sudo truncate -s 0 "$path/record" || sudo rm "$path/record"; then
    echo 'Unexpected successful write' >&2; exit 1
  fi
done
sudo mount -t tmpfs tmpfs "$volume/docs/nested"
reject "$volume"
sudo umount "$volume/docs/nested"
check "$volume"
