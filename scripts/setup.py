#!/usr/bin/env python3
"""Create a private local configuration without overwriting existing values."""
import os
from pathlib import Path
import re
import secrets
import tempfile

root = Path(__file__).resolve().parent.parent
env_path = root / ".env"
if env_path.is_symlink():
    raise SystemExit("Refusing to replace a symlinked .env")
source = env_path if env_path.exists() else root / ".env.example"
contents = source.read_text(encoding="utf-8")
new_values = {
    "APP_KEY": lambda: secrets.token_hex(16),  # 32 ASCII characters
    "JWT_SECRET": lambda: secrets.token_hex(32),
    "DB_PASSWORD": lambda: secrets.token_hex(24),
    "MINIO_ROOT_USER": lambda: "edu" + secrets.token_hex(6),
    "MINIO_ROOT_PASSWORD": lambda: secrets.token_hex(24),
}
for name, generate in new_values.items():
    line = re.compile(rf"^({name}[ \t]*=[ \t]*)([^\n]*)$", re.MULTILINE)
    matches = list(line.finditer(contents))
    if len(matches) > 1:
        raise SystemExit(f"Please resolve duplicate entries for {name} in {source.name}.")
    if not matches:
        contents = contents.rstrip("\n") + f"\n{name}={generate()}\n"
    elif matches[0][2].strip() in ("", '""', "''"):
        contents = line.sub(lambda m: m[1] + generate(), contents)
# Goravel exits with status 0 during init for an invalid key. Catch this before
# invoking migrations so a bad override cannot look like a successful setup.
app_key = os.environ.get("APP_KEY")
if app_key is None:
    app_key = re.search(r"^APP_KEY[ \t]*=[ \t]*(.*)$", contents, re.MULTILINE)[1].strip()
    if app_key.startswith(("'", '"')):
        quote = app_key[0]
        app_key = app_key[1:].split(quote, 1)[0]
    else:
        app_key = re.split(r"[ \t]+#", app_key, maxsplit=1)[0].strip()
if len(app_key.encode("utf-8")) != 32:
    raise SystemExit("APP_KEY must contain exactly 32 bytes. Correct its existing value before starting.")
# Windows requires the temporary handle to close before the atomic rename.
temp = None
try:
    with tempfile.NamedTemporaryFile("w", dir=root, prefix=".env.setup-", delete=False) as out:
        temp = Path(out.name)
        out.write(contents)
        out.flush()
        os.fsync(out.fileno())
    os.replace(temp, env_path)
finally:
    if temp is not None:
        temp.unlink(missing_ok=True)
print("Local .env is ready. Existing nonempty values were preserved.")
