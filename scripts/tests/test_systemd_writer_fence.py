import importlib.util
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

spec = importlib.util.spec_from_file_location(
    "writer_fence", Path(__file__).resolve().parents[1] / "verify-systemd-writer-fence.py")
fence = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fence)

PROPERTIES = """Id=synthetic.service
LoadState=masked
UnitFileState=masked
ActiveState=inactive
SubState=dead
MainPID=0
ControlPID=0
ControlGroup=
"""


class FenceTests(unittest.TestCase):
    def verify(self, properties=PROPERTIES, target="/dev/null"):
        with patch.object(subprocess, "run", return_value=subprocess.CompletedProcess(
                [], 0, properties)), patch.object(fence.os, "readlink", return_value=target):
            return fence.verify("synthetic.service")

    def test_stopped_persistently_masked(self):
        self.assertTrue(self.verify())

    def test_partial_or_inconsistent_state(self):
        for old, new in [
            ("LoadState=masked", "LoadState=loaded"), ("MainPID=0", "MainPID=22"),
            ("UnitFileState=masked", "UnitFileState=masked-runtime"),
            ("ControlGroup=", "ControlGroup=/system.slice/test.service"),
            ("Id=synthetic", "Id=other"), ("ControlPID=0\n", ""),
        ]:
            with self.subTest(old=old):
                self.assertFalse(self.verify(PROPERTIES.replace(old, new)))
        self.assertFalse(self.verify(target="/tmp/elsewhere"))

    def test_second_unfenced_writer_withholds_success_report(self):
        from io import StringIO
        with patch("sys.argv", ["check", "api.service", "worker.service"]), \
                patch.object(fence, "verify", side_effect=[True, False]), \
                patch("sys.stdout", new_callable=StringIO) as out, \
                patch("sys.stderr", new_callable=StringIO):
            self.assertEqual(fence.main(), 1)
            self.assertEqual(out.getvalue(), "")
