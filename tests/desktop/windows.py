"""Drive the shipped Windows executable via UI Automation and capture its native window."""
import os
from pathlib import Path
import subprocess
import time

from PIL import ImageGrab, ImageStat
from pywinauto import Desktop
from pywinauto.keyboard import send_keys


def main():
    if os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted":
        raise SystemExit("Requires a disposable Windows desktop.")
    app = subprocess.Popen([str(Path("clients/desktop/bin/edupilot.exe").resolve())])
    try:
        # WebView2 creates child UIA providers in separate processes. Bind the top
        # level HWND to our exe, then search its descendants across that boundary.
        window = Desktop(backend="uia").window(process=app.pid, title="EduPilot OS")
        window.wait("visible", timeout=45)
        window.set_focus()
        def control(label, kind):
            item = window.child_window(title=label, control_type=kind)
            item.wait("exists enabled", timeout=20)
            return item.wrapper_object()

        def type_into(label, value):
            control(label, "Edit").set_focus()
            send_keys(value, pause=0.025, with_spaces=True)

        def capture(state):
            time.sleep(0.4)
            frame = ImageGrab.grab(window=window.handle)
            width, height = frame.size
            content = frame.crop((width // 8, height // 6, width * 7 // 8, height * 5 // 6))
            if min(width, height) < 450 or max(ImageStat.Stat(content).stddev) < 10:
                raise RuntimeError("Windows returned an empty or undersized window capture.")
            frame.save(Path(os.environ["DESKTOP_SCREEN_DIR"]) / ("windows-" + state + ".png"))

        control("School ID", "Edit")
        capture("signin")
        type_into("School ID", os.environ["EDUPILOT_SEED_SCHOOL"])
        type_into("Email", os.environ["EDUPILOT_SEED_EMAIL"])
        type_into("Password", os.environ["EDUPILOT_SEED_PASSWORD"])
        control("Sign in", "Button").invoke()
        students = control("Students", "Hyperlink")
        capture("directory")
        students.invoke()
        control("Add student", "Button").invoke()
        type_into("Given name", "Jamie")
        type_into("Family name", "Demo")
        control("Save student", "Button").invoke()
        control("Edit Jamie Demo", "Button")
        capture("students")
        control("Sign out", "Button").invoke()
        control("School ID", "Edit")
    finally:
        app.terminate()
        app.wait(timeout=10)


if __name__ == "__main__":
    main()
