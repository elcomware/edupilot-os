"""Drive the actual WebKitGTK window using visible labels on a private X display."""
import csv
import io
import os
from pathlib import Path
import re
import subprocess
import tempfile
import time


def xdo(*args, **kwargs):
    return subprocess.run(["xdotool", *args], check=True, **kwargs)


def type_text(value):
    # Never put a credential on the command line.
    xdo("type", "--clearmodifiers", "--file", "-", input=value.encode())


def main():
    if os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted" or not os.environ.get("DISPLAY"):
        raise SystemExit("Requires a private display on a disposable hosted runner.")
    output = Path(os.environ["DESKTOP_SCREEN_DIR"])
    app = subprocess.Popen(["/opt/edupilot-ci-window/edupilot"], env=dict(os.environ, GDK_BACKEND="x11"))
    try:
        window = subprocess.check_output(
            ["xdotool", "search", "--sync", "--onlyvisible", "--all", "--pid",
             str(app.pid), "--name", "EduPilot OS$"], timeout=45, text=True).splitlines()[0]
        xdo("windowfocus", "--sync", window)

        def capture(path):
            subprocess.run(["import", "-window", window, str(path)], check=True, timeout=12)

        # Locate the text users actually see, without instrumenting the production webview.
        def locate(words, last=False):
            wanted = words.lower().split()
            deadline = time.monotonic() + 18
            while time.monotonic() < deadline:
                capture(frame)
                result = subprocess.check_output(["tesseract", str(frame), "stdout", "--psm", "11", "tsv"],
                                                 stderr=subprocess.DEVNULL, text=True, timeout=10)
                records = [r for r in csv.DictReader(io.StringIO(result), delimiter="\t", quoting=csv.QUOTE_NONE)
                           if r.get("text") and r["level"] == "5"]
                labels = [re.sub(r"^\W+|\W+$", "", r["text"].lower()) for r in records]
                matches = [i for i in range(len(labels)) if labels[i:i+len(wanted)] == wanted]
                if matches:
                    r = records[matches[-1] if last else matches[0]]
                    return str(int(r["left"]) + int(r["width"]) // 2), str(int(r["top"]) + int(r["height"]) // 2)
                time.sleep(0.2)
            raise RuntimeError("Native window did not display expected label: " + words)

        def click(words, last=False):
            x, y = locate(words, last)
            xdo("mousemove", "--window", window, x, y, "click", "1")

        with tempfile.TemporaryDirectory() as scratch:
            frame = Path(scratch) / "private-frame.png"
            locate("Email")
            capture(output / "linux-signin.png")
            click("Email")
            xdo("key", "shift+Tab")
            type_text(os.environ["EDUPILOT_SEED_SCHOOL"])
            xdo("key", "Tab")
            type_text(os.environ["EDUPILOT_SEED_EMAIL"])
            xdo("key", "Tab")
            type_text(os.environ["EDUPILOT_SEED_PASSWORD"])
            xdo("key", "Return")
            locate("Identity")
            capture(output / "linux-directory.png")
            click("Find a module", last=True)
            type_text("Students")
            click("Students", last=True)
            click("Add student")
            click("Given name")
            type_text("Jamie")
            xdo("key", "Tab")
            type_text("Demo")
            click("Save student")
            locate("Jamie Demo")
            capture(output / "linux-students.png")
            click("Sign out")
            locate("Open your workspace")
    finally:
        app.terminate()
        try:
            app.wait(timeout=6)
        except subprocess.TimeoutExpired:
            app.kill()
            app.wait()


if __name__ == "__main__":
    main()
