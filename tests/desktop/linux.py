"""Drive the shipped Linux executable via WebKitGTK accessibility on an isolated X display."""
from collections import deque
import os
import re
from pathlib import Path
import subprocess
import tempfile
import time

import pyatspi


def main():
    if os.environ.get("RUNNER_ENVIRONMENT") != "github-hosted" or not os.environ.get("DISPLAY"):
        raise SystemExit("Requires a private display on a disposable hosted runner.")
    subprocess.run(["gsettings", "set", "org.gnome.desktop.interface", "toolkit-accessibility", "true"], check=True)
    app = subprocess.Popen(["/opt/edupilot-ci-window/edupilot"],
                           env=dict(os.environ, GDK_BACKEND="x11", GTK_A11Y="atspi"))
    try:
        window = subprocess.check_output(
            ["xdotool", "search", "--sync", "--onlyvisible", "--all", "--pid",
             str(app.pid), "--name", "EduPilot OS$"], timeout=45, text=True).splitlines()[0]
        subprocess.run(["xdotool", "windowfocus", "--sync", window], check=True)

        def element(label, roles):
            deadline = time.monotonic() + 22
            while time.monotonic() < deadline:
                nodes = deque([pyatspi.Registry.getDesktop(0)])
                observed = set()
                while nodes:
                    node = nodes.popleft()
                    if node is None:
                        continue
                    try:
                        role = node.getRoleName()
                        observed.add(role)
                        if role in roles and node.name == label:
                            return node
                        nodes.extend(node)
                    except (RuntimeError, TypeError):
                        continue  # A busy panel may have just been replaced.
                time.sleep(0.25)
            raise RuntimeError(f"Missing {roles} labelled {label!r}; available roles: {sorted(observed)}")

        def type_into(label, value):
            field = element(label, {"entry", "text", "password text"})
            if not field.queryComponent().grabFocus():
                raise RuntimeError("Could not focus " + label)
            deadline = time.monotonic() + 5
            while not field.getState().contains(pyatspi.STATE_FOCUSED):
                if time.monotonic() >= deadline:
                    raise RuntimeError("Native focus did not reach " + label)
                time.sleep(0.05)
            subprocess.run(["xdotool", "type", "--clearmodifiers", "--delay", "30", "--file", "-"],
                           input=value.encode(), check=True)

        def activate(label, roles):
            action = element(label, roles).queryAction()
            if not action.doAction(0):
                raise RuntimeError("Could not activate " + label)

        def capture(state, marker):
            # Accessibility events can precede WebKit's first visible frame.
            # Publish the exact frame that contains the expected rendered label.
            path = Path(os.environ["DESKTOP_SCREEN_DIR"]) / ("linux-" + state + ".png")
            with tempfile.TemporaryDirectory() as directory:
                frame = Path(directory) / "frame.png"
                deadline = time.monotonic() + 20
                while time.monotonic() < deadline:
                    subprocess.run(["import", "-window", window, str(frame)], check=True, timeout=10)
                    words = subprocess.check_output(["tesseract", str(frame), "stdout", "--psm", "11"],
                                                    text=True, timeout=8, stderr=subprocess.DEVNULL)
                    if marker in re.findall(r"\w+", words.casefold()):
                        path.write_bytes(frame.read_bytes())
                        return
                    time.sleep(0.2)
            raise RuntimeError("Native display did not paint the " + state + " screen")

        buttons = {"push button", "button"}
        element("Email", {"entry", "text"})
        capture("signin", "email")
        type_into("School ID", os.environ["EDUPILOT_SEED_SCHOOL"])
        type_into("Email", os.environ["EDUPILOT_SEED_EMAIL"])
        type_into("Password", os.environ["EDUPILOT_SEED_PASSWORD"])
        activate("Sign in", buttons)
        element("Schools", {"link"})
        capture("directory", "access")
        activate("Schools", {"link"})
        activate("Add entry", buttons)
        type_into("Title", "Campus review")
        # Sending an X key event completes before WebKit necessarily processes
        # it. Observe the synthetic title in the field before submitting it.
        title = element("Title", {"entry", "text"}).queryText()
        deadline = time.monotonic() + 5
        while title.getText(0, -1) != "Campus review":
            if time.monotonic() >= deadline:
                raise RuntimeError("Synthetic title before submit: " + repr(title.getText(0, -1)))
            time.sleep(0.05)
        activate("Save entry", buttons)
        element("Edit Campus review", buttons)
        capture("workspace", "campus")
        activate("Sign out", buttons)
        element("School ID", {"entry", "text"})
    finally:
        app.terminate()
        try:
            app.wait(timeout=6)
        except subprocess.TimeoutExpired:
            app.kill()
            app.wait()


if __name__ == "__main__":
    main()
