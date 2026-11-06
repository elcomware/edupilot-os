import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "fleet", Path(__file__).resolve().parents[1] / "verify-systemd-fleet.py")
fleet = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fleet)


class FleetTests(unittest.TestCase):
    def setUp(self):
        self.entries = [{"host": host, "machine": digit * 32, "units": ["api.service"]}
                        for host, digit in [("first", "1"), ("second", "2")]]

    def test_invalid_inventory_before_any_connection(self):
        for field, value in [("host", "FIRST"), ("machine", "1" * 32), ("units", []),
                             ("units", ["a;id.service"]), ("units", ["a.service"] * 2)]:
            entries = [self.entries[0], {**self.entries[1], field: value}]
            with self.subTest(field=field), patch.object(subprocess, "run") as ssh:
                with self.assertRaises(ValueError):
                    fleet.verify(entries, "operator", __file__, 22)
                ssh.assert_not_called()

    def test_all_hosts_must_reply(self):
        evidence = json.dumps({"classification": "systemd-writer-check", "verifiedUnits": 1})
        replies = [subprocess.CompletedProcess([], 0, (e["machine"] + "\n" + evidence).encode())
                   for e in self.entries]
        with patch.object(subprocess, "run", side_effect=replies):
            result = fleet.verify(self.entries, "operator", __file__, 22)
            self.assertEqual((result["verifiedHosts"], result["verifiedUnits"]), (2, 2))
        for failure in [subprocess.TimeoutExpired("ssh", 10), replies[0]]:
            with patch.object(subprocess, "run", side_effect=[replies[0], failure]):
                with self.assertRaises((ValueError, subprocess.TimeoutExpired)):
                    fleet.verify(self.entries, "operator", __file__, 22)
