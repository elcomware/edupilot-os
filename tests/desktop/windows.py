"""Drive the shipped Windows executable via UI Automation and capture its native window."""
import os
import re
from pathlib import Path
import subprocess
import tempfile
import time

from PIL import ImageGrab
import win32con
import win32gui
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
        win32gui.ShowWindow(window.handle, win32con.SW_MAXIMIZE)
        window.set_focus()
        failed_paints = []

        def control(label, kind):
            item = window.child_window(title=label, control_type=kind)
            item.wait("exists enabled", timeout=20)
            return item.wrapper_object()

        def type_into(label, value):
            control(label, "Edit").set_focus()
            send_keys(value, pause=0.025, with_spaces=True)

        def capture(state, marker):
            # Some WebView2 surfaces paint via the compositor rather than the
            # HWND device context. Read the foreground app rectangle in that case.
            deadline = time.monotonic() + 18
            with tempfile.TemporaryDirectory() as scratch:
                sample = Path(scratch) / "frame.png"
                while time.monotonic() < deadline:
                    rect = win32gui.GetWindowRect(window.handle)
                    client = win32gui.GetClientRect(window.handle)
                    left, _ = win32gui.ClientToScreen(window.handle, (0, 0))
                    right, bottom = win32gui.ClientToScreen(window.handle, client[2:])
                    visible = (left, max(0, rect[1]), right, bottom)
                    readers = [lambda: ImageGrab.grab(window=window.handle)]
                    if win32gui.GetForegroundWindow() == window.handle:
                        readers.append(lambda: ImageGrab.grab(bbox=visible, all_screens=True))
                    for read in readers:
                        frame = read()
                        if min(frame.size) < 450 or len(set(frame.resize((64, 48)).getdata())) < 16:
                            continue
                        frame.save(sample)
                        words = subprocess.check_output([os.environ["DESKTOP_TESSERACT"], str(sample), "stdout",
                                                         "--psm", "11"], text=True, errors="replace",
                                                        stderr=subprocess.DEVNULL, timeout=10)
                        if marker in re.findall(r"\w+", words.casefold()):
                            frame.save(Path(os.environ["DESKTOP_SCREEN_DIR"]) / ("windows-" + state + ".png"))
                            return
                    time.sleep(0.25)
            failed_paints.append(state)
            print(f"No validated {state} frame within window bounds {rect}", flush=True)

        control("School ID", "Edit")
        capture("signin", "email")
        type_into("School ID", os.environ["EDUPILOT_SEED_SCHOOL"])
        type_into("Email", os.environ["EDUPILOT_SEED_EMAIL"])
        type_into("Password", os.environ["EDUPILOT_SEED_PASSWORD"])
        control("Sign in", "Button").invoke()
        students = control("Students", "Hyperlink")
        capture("directory", "identity")
        students.invoke()
        control("Add student", "Button").invoke()
        type_into("Given name", "Jamie")
        type_into("Family name", "Demo")
        control("Save student", "Button").invoke()
        control("Edit Jamie Demo", "Button")
        capture("students", "jamie")
        navigation = window.child_window(title="Navigation", control_type="Button")
        if navigation.exists(timeout=0.5):
            navigation.wrapper_object().invoke()
        control("Sign out", "Button").invoke()
        control("School ID", "Edit")
        if failed_paints:
            raise RuntimeError("Native journey completed; missing validated frames: " + ", ".join(failed_paints))
    finally:
        app.terminate()
        app.wait(timeout=10)


if __name__ == "__main__":
    main()
