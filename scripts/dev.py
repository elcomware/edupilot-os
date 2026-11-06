#!/usr/bin/env python3
"""Run the native EduPilot OS stack in the foreground (POSIX systems)."""
import argparse
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import sys
import tempfile
import time


class RequirementError(Exception):
    """A local prerequisite is missing or incompatible."""


def require_tools(names):
    for name in names:
        if shutil.which(name) is None:
            raise RequirementError(f"missing prerequisite: {name}")


def require_process_ownership():
    capabilities = ("waitid", "WNOWAIT", "WEXITED", "WNOHANG", "P_PID", "CLD_EXITED")
    if os.name != "posix" or any(not hasattr(os, name) for name in capabilities):
        raise RequirementError("native supervision requires POSIX waitid/WNOWAIT support; use a supported Python/OS or make docker-up")
    if signal.getsignal(signal.SIGCHLD) == signal.SIG_IGN:
        raise RequirementError("native supervision requires waitable child processes; restore SIGCHLD handling or use make docker-up")


def supervise(preparation, services, cwd, stop_timeout=5.0):
    """Run preparation, then services; pin owned group leaders until final signals."""
    require_process_ownership()
    processes = []
    previous_handlers = {}
    changing_ownership = False
    shutting_down = False
    pending_signal = None

    def interrupted(signum, _frame):
        nonlocal pending_signal
        if shutting_down:
            return
        if changing_ownership:
            # Finish registering/retiring the child before interruption is handled.
            pending_signal = signum
        else:
            raise KeyboardInterrupt(signum)

    def start(name, argv):
        nonlocal changing_ownership
        print(f"EduPilot OS: starting {name}", flush=True)
        # No shell; subprocesses and their descendants belong to our new session.
        changing_ownership = True
        try:
            process = subprocess.Popen(argv, cwd=cwd, start_new_session=True)
            processes.append(process)
        finally:
            changing_ownership = False
        if pending_signal is not None:
            raise KeyboardInterrupt(pending_signal)
        return process

    def status(process, block=False):
        # Unlike Popen.poll()/wait(), WNOWAIT preserves the child's kernel PID
        # reservation, even after exit. Its process-group ID cannot be reused
        # by an unrelated session while we still intend to signal that group.
        flags = os.WEXITED | os.WNOWAIT | (0 if block else os.WNOHANG)
        event = os.waitid(os.P_PID, process.pid, flags)
        if event is None:
            return None
        return event.si_status if event.si_code == os.CLD_EXITED else -event.si_status

    def send(owned, signum):
        for process in owned:
            try:
                os.killpg(process.pid, signum)
            except ProcessLookupError:
                pass

    def retire(owned):
        nonlocal changing_ownership
        changing_ownership = True
        try:
            # Every leader stays unreaped through the final group signal. Remove
            # ownership before reaping so no later cleanup can target a reused ID.
            send(owned, signal.SIGKILL)
            for process in owned:
                processes.remove(process)
                process.wait()
        finally:
            changing_ownership = False
        if pending_signal is not None and not shutting_down:
            raise KeyboardInterrupt(pending_signal)

    try:
        for signum in (signal.SIGINT, signal.SIGTERM):
            previous_handlers[signum] = signal.signal(signum, interrupted)
        for name, argv in preparation:
            process = start(name, argv)
            result = status(process, block=True)
            # A finished preparation must leave no background work behind. Kill
            # any remaining descendants while the unreaped leader pins ownership.
            retire([process])
            if result != 0:
                print(f"EduPilot OS: {name} failed; startup stopped.", file=sys.stderr)
                return 1
        active = [(name, start(name, argv)) for name, argv in services]
        while active:
            for name, process in active:
                result = status(process)
                if result is not None:
                    print(f"EduPilot OS: {name} exited (status {result}); stopping services.", file=sys.stderr)
                    return 1
            time.sleep(0.05)
        return 0
    except KeyboardInterrupt as exc:
        print("EduPilot OS: stopping native services.", flush=True)
        return 143 if exc.args and exc.args[0] == signal.SIGTERM else 130
    except OSError:
        # Do not interpolate argv, environment values, or raw OS diagnostics.
        print("EduPilot OS: unable to start a required process.", file=sys.stderr)
        return 1
    finally:
        shutting_down = True
        # Further interrupts must not abort cleanup and orphan children.
        for signum in previous_handlers:
            signal.signal(signum, signal.SIG_IGN)
        owned = processes.copy()
        if owned:
            send(owned, signal.SIGTERM)
            # Allow the full grace period for descendants as well as their leader.
            # Reaping a leader early would discard our ownership reservation.
            time.sleep(stop_timeout)
            retire(owned)
        for signum, handler in previous_handlers.items():
            signal.signal(signum, handler)


def version_output(argv, label):
    try:
        result = subprocess.run(argv, capture_output=True, text=True, timeout=10)
        if result.returncode == 0:
            return result.stdout.strip()
    except (OSError, subprocess.TimeoutExpired):
        pass
    raise RequirementError(f"{label} is unavailable; check its installation")


def node_compatible(version, engines):
    """Evaluate the numeric caret/minimum alternatives used by package.json."""
    for alternative in engines.split("||"):
        match = re.fullmatch(r"\s*(\^|>=)(\d+)(?:\.(\d+))?(?:\.(\d+))?\s*", alternative)
        if not match:
            raise RequirementError("unsupported Node engine constraint in package.json")
        minimum = tuple(int(part or 0) for part in match.groups()[1:])
        if version >= minimum and (match[1] == ">=" or version[0] == minimum[0]):
            return True
    return False


def check_environment(root):
    require_process_ownership()
    require_tools(("go", "node", "npm", "make"))
    go = re.search(r"\bgo(\d+)\.(\d+)(?:\.(\d+))?\b", version_output(["go", "version"], "Go"))
    if not go or tuple(int(part or 0) for part in go.groups()) < (1, 26, 10):
        raise RequirementError("Go 1.26.10 or newer is required; ensure the Go compiler is on PATH")
    node = re.fullmatch(r"v(\d+)\.(\d+)\.(\d+)", version_output(["node", "--version"], "Node"))
    try:
        engines = json.loads((root / "package.json").read_text())["engines"]["node"]
    except (OSError, ValueError, KeyError):
        raise RequirementError("cannot read Node engine requirements from package.json") from None
    if not node or not node_compatible(tuple(map(int, node.groups())), engines):
        raise RequirementError(f"Node must satisfy package.json engines: {engines}")
    version_output(["npm", "--version"], "npm")
    if not (root / "node_modules/vite/bin/vite.js").is_file():
        raise RequirementError("local Vite dependency is missing; run make install")
    # Readiness checks deliberately do not create .env, migrate, build, or bind ports.
    print("EduPilot OS: native prerequisites are available. Configuration and database connectivity are checked at startup.")


def main(argv=None, root=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true", help="check local prerequisites without changing data or starting services")
    args = parser.parse_args(argv)
    root = root or Path(__file__).resolve().parent.parent
    try:
        check_environment(root)
        if args.check:
            return 0
        # tmp is git-ignored; mkdtemp makes this invocation's binaries private (0700).
        runtime_parent = root / "tmp"
        runtime_parent.mkdir(mode=0o700, exist_ok=True)
        with tempfile.TemporaryDirectory(prefix="edupilot-native-", dir=runtime_parent) as runtime:
            api = str(Path(runtime) / "api")
            worker = str(Path(runtime) / "worker")
            preparation = [
                ("local configuration", [sys.executable, "scripts/setup.py"]),
                ("migrations", ["go", "run", "./cmd/migrate"]),
                ("API build", ["go", "build", "-o", api, "."]),
                ("worker build", ["go", "build", "-o", worker, "./cmd/worker"]),
            ]
            services = [
                ("API", [api]),
                ("outbox worker", [worker]),
                ("web", ["node", str(root / "node_modules/vite/bin/vite.js"), str(root / "clients/web"),
                         "--config", str(root / "clients/web/vite.config.ts"),
                         "--host", "127.0.0.1", "--port", "5173", "--strictPort"]),
            ]
            print("EduPilot OS: native foreground stack; open http://127.0.0.1:5173 and press Ctrl-C here to stop.", flush=True)
            return supervise(preparation, services, root)
    except RequirementError as exc:
        print(f"EduPilot OS: {exc}", file=sys.stderr)
        return 1
    except KeyboardInterrupt:
        return 130
    except OSError:
        print("EduPilot OS: unable to prepare the private runtime directory.", file=sys.stderr)
        return 1


if __name__ == "__main__":
    sys.exit(main())
