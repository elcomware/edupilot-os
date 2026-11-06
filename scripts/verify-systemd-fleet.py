#!/usr/bin/env python3
"""Sample every inventoried Linux writer host over authenticated SSH."""
import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

PROBE = Path(__file__).with_name("verify-systemd-writer-fence.py")


def inventory(entries):
    if not isinstance(entries, list) or not entries:
        raise ValueError()
    hosts, machines = set(), set()
    for entry in entries:
        if not isinstance(entry, dict) or set(entry) != {"host", "machine", "units"}:
            raise ValueError()
        host, machine, units = entry["host"], entry["machine"], entry["units"]
        if (not re.fullmatch(r"[a-zA-Z0-9][a-zA-Z0-9.-]{0,252}", host)
                or not re.fullmatch(r"[0-9a-f]{32}", machine) or int(machine, 16) == 0
                or host.lower() in hosts or machine in machines
                or not isinstance(units, list) or not units):
            raise ValueError()
        if (any(not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.@:-]{0,239}\.service", u)
                or u.endswith("@.service") for u in units) or len(set(units)) != len(units)):
            raise ValueError()
        hosts.add(host.lower())
        machines.add(machine)
    return entries


def verify(entries, login, known_hosts, port):
    entries = inventory(entries)
    if not re.fullmatch(r"[a-zA-Z_][a-zA-Z_0-9-]*", login) or not 1 <= port <= 65535:
        raise ValueError()
    # SSH expands tokens and whitespace in config values. Require a simple private path.
    path = str(Path(known_hosts).resolve(strict=True))
    if not re.fullmatch(r"/[a-zA-Z0-9/_.-]+", path):
        raise ValueError()
    probe = PROBE.read_bytes()
    total = 0
    for entry in entries:
        units = entry["units"]
        command = "cat /etc/machine-id && python3 - " + " ".join(units)
        result = subprocess.run(
            ["ssh", "-T", "-F", "/dev/null", "-oBatchMode=yes", "-oConnectTimeout=10",
             "-oStrictHostKeyChecking=yes", "-oUpdateHostKeys=no", "-oGlobalKnownHostsFile=/dev/null",
             "-oUserKnownHostsFile=" + path, "-oClearAllForwardings=yes", "-oForwardAgent=no",
             "-p", str(port), "-l", login, "--", entry["host"], command],
            input=probe, capture_output=True, timeout=20 + 10 * len(units), check=True)
        machine, evidence = result.stdout.decode().splitlines()
        if machine != entry["machine"] or json.loads(evidence) != {
                "classification": "systemd-writer-check", "verifiedUnits": len(units)}:
            raise ValueError()
        total += len(units)
    return {"classification": "systemd-fleet-sample", "verifiedHosts": len(entries),
            "verifiedUnits": total}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--inventory", required=True, type=Path)
    parser.add_argument("--login", required=True)
    parser.add_argument("--known-hosts", required=True)
    parser.add_argument("--port", type=int, default=22)
    args = parser.parse_args()
    try:
        print(json.dumps(verify(json.loads(args.inventory.read_bytes()), args.login,
                                args.known_hosts, args.port)))
    except (ValueError, OSError, TypeError, subprocess.SubprocessError):
        print("Declared writer fleet could not be verified.", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
