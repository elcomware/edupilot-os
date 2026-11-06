#!/usr/bin/env bash
# Invoke from the disposable systemd drill with two masked synthetic services.
set -euo pipefail
dir=$(mktemp -d)
cleanup() {
  if [[ -f "$dir/pid" ]]; then sudo kill "$(cat "$dir/pid")" || true; fi
  [[ -z "${SSH_AGENT_PID:-}" ]] || ssh-agent -k >/dev/null
  rm -rf "$dir"
}
unset SSH_AGENT_PID SSH_AUTH_SOCK
trap cleanup EXIT
ssh-keygen -q -t ed25519 -N '' -f "$dir/host"
ssh-keygen -q -t ed25519 -N '' -f "$dir/client"
eval "$(ssh-agent -s)" >/dev/null
ssh-add "$dir/client" 2>/dev/null
port=22094
cat > "$dir/sshd" <<EOF
ListenAddress 127.0.0.1
Port $port
PidFile $dir/pid
HostKey $dir/host
AuthorizedKeysFile $dir/client.pub
StrictModes no
AllowUsers $(id -un)
AuthenticationMethods publickey
PasswordAuthentication no
KbdInteractiveAuthentication no
UsePAM yes
DisableForwarding yes
EOF
sudo mkdir -p /run/sshd
sudo /usr/sbin/sshd -f "$dir/sshd"
printf '[127.0.0.1]:%s %s\n' "$port" "$(cat "$dir/host.pub")" > "$dir/pins"
machine=$(cat /etc/machine-id)
write_inventory() {
  printf '[{"host":"127.0.0.1","machine":"%s","units":["%s","%s"]}]\n' "$1" "$2" "$3" > "$dir/inventory"
}
check() {
  python3 scripts/verify-systemd-fleet.py --inventory "$dir/inventory" --login "$(id -un)" --known-hosts "$dir/pins" --port "$port"
}
write_inventory "$machine" "$1" "$2"
check
write_inventory "$(tr -d - </proc/sys/kernel/random/uuid)" "$1" "$2"
if check; then exit 1; fi
write_inventory "$machine" "$1" "edupilot-absent.service"
if check; then exit 1; fi
write_inventory "$machine" "$1" "$2"
printf '[127.0.0.1]:%s %s\n' "$port" "$(cat "$dir/client.pub")" > "$dir/pins"
if check; then exit 1; fi
