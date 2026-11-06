"""Drive the shipped Linux executable via WebKitGTK accessibility on an isolated X display."""
from collections import deque
import os
from pathlib import Path
import subprocess
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
            subprocess.run(["xdotool", "type", "--clearmodifiers", "--file", "-"],
                           input=value.encode(), check=True)

        def activate(label, roles):
            action = element(label, roles).queryAction()
            if not action.doAction(0):
                raise RuntimeError("Could not activate " + label)

        def capture(state):
            time.sleep(0.35)
            path = Path(os.environ["DESKTOP_SCREEN_DIR"]) / ("linux-" + state + ".png")
            subprocess.run(["import", "-window", window, str(path)], check=True, timeout=12)

        buttons = {"push button", "button"}
        element("Email", {"entry", "text"})
        capture("signin")
        type_into("School ID", os.environ["EDUPILOT_SEED_SCHOOL"])
        type_into("Email", os.environ["EDUPILOT_SEED_EMAIL"])
        type_into("Password", os.environ["EDUPILOT_SEED_PASSWORD"])
        activate("Sign in", buttons)
        element("Students", {"link"})
        capture("directory")
        activate("Students", {"link"})
        activate("Add student", buttons)
        type_into("Given name", "Jamie")
        type_into("Family name", "Demo")
        activate("Save student", buttons)
        element("Edit Jamie Demo", buttons)
        capture("students")
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
