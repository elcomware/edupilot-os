"""Run the packaged client against a fresh native API on a disposable hosted runner."""
import os
from pathlib import Path
import secrets
import sqlite3
import subprocess
import sys
import time
from urllib.error import URLError
from urllib.request import urlopen

ROOT = Path(__file__).resolve().parents[2]
WINDOWS = sys.platform == "win32"
SUFFIX = ".exe" if WINDOWS else ""
PLATFORM = "windows" if WINDOWS else "linux"
SCHOOL = "demo-" + PLATFORM
EMAIL = "registrar@example.test"
OUTPUT = Path(os.environ["RUNNER_TEMP"]) / "desktop-screens"
DATABASE = ROOT / "database" / "edupilot.sqlite"


def run():
    if (os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted"
            or DATABASE.exists() or (ROOT / ".env").exists()):
        raise SystemExit("The desktop fixture requires a clean GitHub-hosted runner.")
    os.chdir(ROOT)
    OUTPUT.mkdir()
    # Override placement even if the caller has configured another database.
    base = dict(os.environ, DB_CONNECTION="sqlite", DB_DATABASE="database/edupilot.sqlite",
                EDUPILOT_DEDICATED_DATABASES="[]", EDUPILOT_ENABLED_MODULES="",
                EDUPILOT_DOCUMENTS_DRIVER="local", APP_HOST="127.0.0.1", APP_PORT="3000",
                EDUPILOT_API_ORIGIN="http://127.0.0.1:3000")
    subprocess.run([sys.executable, "scripts/setup.py"], env=base, check=True)
    subprocess.run(["go", "run", "./cmd/migrate"], env=base, check=True)
    password = secrets.token_hex(24)
    print("::add-mask::" + password, flush=True)
    fixture = dict(EDUPILOT_SEED_SCHOOL=SCHOOL, EDUPILOT_SEED_EMAIL=EMAIL,
                   EDUPILOT_SEED_PASSWORD=password)
    subprocess.run(["go", "run", "./cmd/seed-admin"], env=base | fixture, check=True)
    binary = Path(os.environ["RUNNER_TEMP"]) / ("desktop-fixture-api" + SUFFIX)
    subprocess.run(["go", "build", "-o", str(binary), "."], env=base, check=True)
    api = subprocess.Popen([str(binary)], env=base)
    try:
        for _ in range(120):
            if api.poll() is not None:
                raise RuntimeError("The fixture API exited during startup.")
            try:
                with urlopen(base["EDUPILOT_API_ORIGIN"] + "/api/v1/ready", timeout=2) as r:
                    if r.status == 200:
                        break
            except URLError:
                pass
            time.sleep(0.5)
        else:
            raise RuntimeError("Fixture API did not become ready.")
        proof = base | fixture | {"DESKTOP_SCREEN_DIR": str(OUTPUT)}
        if WINDOWS:
            command = [sys.executable, "tests/desktop/windows.py"]
        else:
            command = ["xvfb-run", "-a", "-s", "-screen 0 1440x960x24",
                       "dbus-run-session", "--", "/usr/bin/python3", "tests/desktop/linux.py"]
        subprocess.run(command, env=proof, check=True, timeout=180)
        # An independent read proves the native form really wrote a school record.
        with sqlite3.connect(DATABASE.as_uri() + "?mode=ro", uri=True) as db:
            rows = db.execute("SELECT school_id,given_name,family_name,status FROM students_records").fetchall()
        if rows != [(SCHOOL, "Jamie", "Demo", "active")]:
            raise RuntimeError("The student saved in the window did not match the fixture database.")
        for state in ("signin", "directory", "students"):
            image = OUTPUT / f"{PLATFORM}-{state}.png"
            if not image.is_file() or image.stat().st_size < 1024:
                raise RuntimeError("Missing native capture for " + state)
        print(f"{PLATFORM}: signed in, opened the directory, saved a student, verified the database and signed out.")
    finally:
        if api.poll() is None:
            api.terminate()
            try:
                api.wait(timeout=8)
            except subprocess.TimeoutExpired:
                api.kill()
                api.wait()


if __name__ == "__main__":
    run()
