#!/usr/bin/env python3
"""Check operator-declared system services during an offline upgrade."""
import argparse
import json
import os
from pathlib import Path
import re
import subprocess
import sys


def verify(unit):
    # Query only non-sensitive properties. Never log child output.
    expected = {
        "Id": unit, "LoadState": "masked", "UnitFileState": "masked",
        "ActiveState": "inactive", "SubState": "dead", "MainPID": "0",
        "ControlPID": "0", "ControlGroup": "",
    }
    result = subprocess.run(
        ["systemctl", "--system", "show", "--all",
         "--property=" + ",".join(expected), "--", unit],
        capture_output=True, text=True, timeout=10, check=True,
    )
    values = dict(line.split("=", 1) for line in result.stdout.splitlines())
    return (values == expected and
            os.readlink(Path("/etc/systemd/system") / unit) == "/dev/null")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("units", nargs="+", help="all canonical writer service names on this host")
    units = parser.parse_args().units
    if (len(set(units)) != len(units) or
            any(not re.fullmatch(r"[A-Za-z0-9][A-Za-z0-9_.@:-]{0,239}\.service", u)
                or u.endswith("@.service") for u in units)):
        parser.error("use unique concrete service names")
    try:
        if not all(verify(unit) for unit in units):
            raise ValueError()
    except (OSError, subprocess.SubprocessError, ValueError):
        print("Declared system service fence could not be verified.", file=sys.stderr)
        return 1
    print(json.dumps({"classification": "systemd-writer-check", "verifiedUnits": len(units)}))
    return 0


if __name__ == "__main__":
    sys.exit(main())
