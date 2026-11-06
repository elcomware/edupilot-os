"""Real-process regression tests for the native development launcher."""
import importlib.util
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time
import unittest


DEV = Path(__file__).resolve().parents[1] / "dev.py"


class NativeLauncherTests(unittest.TestCase):
    def setUp(self):
        self.assertTrue(DEV.is_file(), "native launcher scripts/dev.py is missing")
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.running = []
        self.addCleanup(self.cleanup)

    def cleanup(self):
        for process in self.running:
            if process.poll() is None:
                process.kill()
            process.wait(timeout=5)

    def launch(self, preparation, services, observe_cleanup=False):
        audit = ""
        if observe_cleanup:
            audit = (
                "\ndef observe(event, args):\n"
                f"    if event == 'os.killpg' and Path({str(self.root / 'api')!r}).exists():\n"
                f"        with open({str(self.root / 'shutdown-targets')!r}, 'a') as log:\n"
                "            log.write(str(args[0]) + '\\n')\n"
                "sys.addaudithook(observe)\n"
            )
        code = (
            "import importlib.util,sys; from pathlib import Path; "
            f"s=importlib.util.spec_from_file_location('dev',{str(DEV)!r}); "
            "m=importlib.util.module_from_spec(s); s.loader.exec_module(m); "
            + audit + f"\nsys.exit(m.supervise({preparation!r},{services!r},Path({str(self.root)!r}),stop_timeout=0.3))"
        )
        process = subprocess.Popen([sys.executable, "-c", code], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True)
        self.running.append(process)
        return process

    def command(self, code):
        return [sys.executable, "-c", code]

    def sleeper(self, name, ignore_term=False):
        marker = str(self.root / name)
        setup = "signal.signal(signal.SIGTERM, signal.SIG_IGN); " if ignore_term else ""
        return self.command(f"import os,time,signal; from pathlib import Path; {setup}Path({marker!r}).write_text(str(os.getpid())); time.sleep(60)")

    def wait_marker(self, name):
        path = self.root / name
        deadline = time.monotonic() + 5
        while not path.exists() and time.monotonic() < deadline:
            time.sleep(0.02)
        self.assertTrue(path.exists(), f"child {name} never started")
        return int(path.read_text())

    def assert_stopped(self, pid):
        deadline = time.monotonic() + 3
        while time.monotonic() < deadline:
            try:
                os.kill(pid, 0)
            except ProcessLookupError:
                return
            # A killed grandchild may briefly wait for the host's init to reap it.
            stat = Path(f"/proc/{pid}/stat")
            if stat.exists() and stat.read_text().split()[2] == "Z":
                return
            time.sleep(0.02)
        self.fail(f"child {pid} survived launcher shutdown")

    def check_fixture(self, node_version="v24.3.0", go_version="go version go1.26.10 linux/amd64"):
        tools = self.root / "tools"
        tools.mkdir()
        for name, output in (("go", go_version), ("node", node_version), ("npm", "11.0.0"), ("make", "GNU Make 4.3")):
            executable = tools / name
            executable.write_text(f"#!{sys.executable}\nprint({output!r})\n")
            executable.chmod(0o700)
        (self.root / "package.json").write_text('{"engines":{"node":"^22.13.0 || ^24.3.0 || >=26"}}')
        (self.root / "node_modules/vite/bin").mkdir(parents=True)
        (self.root / "node_modules/vite/bin/vite.js").write_text("throw Error('must not start');")
        (self.root / ".env").write_text("APP_KEY=secret-check-sentinel\n")
        return {**os.environ, "PATH": str(tools)}

    def run_check(self, env, without_waitid=False, argv=("--check",)):
        code = (
            "import importlib.util,sys; from pathlib import Path; "
            f"s=importlib.util.spec_from_file_location('dev',{str(DEV)!r}); "
            "m=importlib.util.module_from_spec(s); s.loader.exec_module(m); "
            + ("del m.os.WNOWAIT; " if without_waitid else "")
            + f"sys.exit(m.main({list(argv)!r},root=Path({str(self.root)!r})))"
        )
        return subprocess.run([sys.executable, "-c", code], env=env, capture_output=True, text=True, timeout=5)

    def test_check_is_read_only_and_starts_no_services(self):
        env = self.check_fixture()
        before = {str(p.relative_to(self.root)): (p.stat().st_mtime_ns, p.read_bytes()) for p in self.root.rglob("*") if p.is_file()}
        result = self.run_check(env)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        after = {str(p.relative_to(self.root)): (p.stat().st_mtime_ns, p.read_bytes()) for p in self.root.rglob("*") if p.is_file()}
        self.assertEqual(before, after)
        self.assertNotIn("secret-check-sentinel", result.stdout + result.stderr)
        self.assertFalse((self.root / "tmp").exists())

    def test_check_rejects_hosts_without_safe_child_ownership_support(self):
        result = self.run_check(self.check_fixture(), without_waitid=True, argv=[])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("waitid", result.stderr)
        self.assertIn("docker-up", result.stderr)
        self.assertFalse((self.root / "tmp").exists())

    def test_check_rejects_unpatched_go_toolchain(self):
        result = self.run_check(self.check_fixture(go_version="go version go1.26.3 linux/amd64"))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("1.26.10", result.stderr)

    def test_check_rejects_node_outside_package_engines(self):
        result = self.run_check(self.check_fixture(node_version="v20.19.0"))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Node", result.stderr)

    def test_check_rejects_non_go_program_without_echoing_its_output(self):
        result = self.run_check(self.check_fixture(go_version="secret-tool-sentinel"))
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("Go", result.stderr)
        self.assertNotIn("secret-tool-sentinel", result.stdout + result.stderr)

    def test_shutdown_does_not_stop_an_unrelated_process(self):
        unrelated = subprocess.Popen(self.sleeper("unrelated"), start_new_session=True)
        self.running.append(unrelated)
        self.wait_marker("unrelated")
        process = self.launch([], [("api", self.sleeper("api"))])
        child = self.wait_marker("api")
        process.send_signal(signal.SIGTERM)
        output, _ = process.communicate(timeout=5)
        self.assertEqual(process.returncode, 143, output)
        self.assert_stopped(child)
        self.assertIsNone(unrelated.poll())

    def test_missing_prerequisite_fails_before_any_command(self):
        spec = importlib.util.spec_from_file_location("dev", DEV)
        dev = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(dev)
        with self.assertRaisesRegex(dev.RequirementError, "missing.*edupilot-not-installed"):
            dev.require_tools(["edupilot-not-installed"])
        self.assertEqual(list(self.root.iterdir()), [])

    def test_completed_preparation_is_never_targeted_at_later_service_shutdown(self):
        preparation = self.command(f"import os; from pathlib import Path; Path({str(self.root / 'prepared')!r}).write_text(str(os.getpid()))")
        process = self.launch([("migration", preparation)], [("api", self.sleeper("api"))], observe_cleanup=True)
        prepared = self.wait_marker("prepared")
        api = self.wait_marker("api")
        process.send_signal(signal.SIGINT)
        output, _ = process.communicate(timeout=5)
        self.assertEqual(process.returncode, 130, output)
        self.assert_stopped(api)
        targets = (self.root / "shutdown-targets").read_text().splitlines()
        self.assertNotIn(str(prepared), targets, "completed preparation PGID was retained for later shutdown")

    def test_completed_preparation_descendants_stop_before_services_start(self):
        descendant = self.sleeper("grandchild", ignore_term=True)
        preparation = self.command(
            f"import subprocess,time; from pathlib import Path; subprocess.Popen({descendant!r}); "
            f"p=Path({str(self.root / 'grandchild')!r});\nwhile not p.exists(): time.sleep(0.02)"
        )
        process = self.launch([("migration", preparation)], [("api", self.sleeper("api"))])
        child = self.wait_marker("grandchild")
        api = self.wait_marker("api")
        try:
            self.assert_stopped(child)
        finally:
            process.send_signal(signal.SIGINT)
            output, _ = process.communicate(timeout=5)
        self.assertEqual(process.returncode, 130, output)
        self.assert_stopped(api)

    def test_migration_failure_stops_before_starting_children(self):
        process = self.launch([("migration", self.command("raise SystemExit(7)"))], [("api", self.sleeper("api"))])
        output, _ = process.communicate(timeout=5)
        self.assertNotEqual(process.returncode, 0, output)
        self.assertIn("migration", output)
        self.assertFalse((self.root / "api").exists())

    def test_child_failure_stops_its_siblings_and_descendants(self):
        grandchild = self.sleeper("grandchild")
        spawn = self.command(f"import subprocess,time; subprocess.Popen({grandchild!r}); time.sleep(60)")
        wait_then_fail = self.command(f"import time; from pathlib import Path; p=Path({str(self.root / 'grandchild')!r});\nwhile not p.exists(): time.sleep(0.02)\nraise SystemExit(9)")
        process = self.launch([], [("worker", spawn), ("api", self.sleeper("api")), ("web", wait_then_fail)])
        api = self.wait_marker("api")
        descendant = self.wait_marker("grandchild")
        output, _ = process.communicate(timeout=5)
        self.assertNotEqual(process.returncode, 0, output)
        self.assertIn("web", output)
        self.assert_stopped(api)
        self.assert_stopped(descendant)

    def test_interrupt_stops_children_even_when_they_ignore_termination(self):
        process = self.launch([], [("api", self.sleeper("api", ignore_term=True)), ("worker", self.sleeper("worker"))])
        children = [self.wait_marker(name) for name in ("api", "worker")]
        process.send_signal(signal.SIGINT)
        output, _ = process.communicate(timeout=5)
        self.assertEqual(process.returncode, 130, output)
        for pid in children:
            self.assert_stopped(pid)

    def test_interrupt_during_migration_stops_preparation(self):
        process = self.launch([("migration", self.sleeper("migration"))], [("api", self.sleeper("api"))])
        child = self.wait_marker("migration")
        process.send_signal(signal.SIGINT)
        output, _ = process.communicate(timeout=5)
        self.assertEqual(process.returncode, 130, output)
        self.assert_stopped(child)
        self.assertFalse((self.root / "api").exists())

    def test_successful_child_exit_is_an_unexpected_service_failure(self):
        process = self.launch([], [("api", self.command("pass")), ("worker", self.sleeper("worker"))])
        output, _ = process.communicate(timeout=5)
        self.assertNotEqual(process.returncode, 0, output)
        self.assertIn("api", output)


if __name__ == "__main__":
    unittest.main()
