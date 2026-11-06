#!/usr/bin/env python3
"""Check that a local private document root resides on the expected read-only ext4 volume."""
import argparse
import os
from pathlib import Path
import re
import stat
import sys


def mount_path(value):
    # mountinfo escapes whitespace and literal backslashes with octal sequences.
    return re.sub(rb"\\([0-7]{3})", lambda m: bytes([int(m[1], 8)]), value)


def check_mounts(table, mount_id, device, root):
    found = False
    for line in table.splitlines():
        before, after = line.split(b" - ", 1)
        fields = before.split()
        kind, _, super_options = after.split()
        point = mount_path(fields[4])
        if fields[0] == mount_id:
            found = (fields[2] == device and fields[3] == b"/" and kind == b"ext4"
                     and b"ro" in fields[5].split(b",") and b"ro" in super_options.split(b",")
                     and point != b"/" and (root == point or root.startswith(point + b"/")))
        elif point == root or point.startswith(root + b"/"):
            return False
    return found


def verify(root, device):
    path = Path(root)
    if not path.is_absolute() or path.resolve(strict=True) != path or ".." in path.parts:
        return False
    block = os.stat(device)
    if not stat.S_ISBLK(block.st_mode):
        return False
    fd = os.open(path, os.O_RDONLY | os.O_DIRECTORY | os.O_NOFOLLOW)
    try:
        entry = os.fstat(fd)
        if entry.st_dev != block.st_rdev or not os.fstatvfs(fd).f_flag & os.ST_RDONLY:
            return False
        mount_id = re.search(rb"^mnt_id:\s*(\d+)$", Path(f"/proc/self/fdinfo/{fd}").read_bytes(), re.M)
        number = f"{os.major(entry.st_dev)}:{os.minor(entry.st_dev)}".encode()
        return bool(mount_id and check_mounts(Path("/proc/self/mountinfo").read_bytes(),
                                              mount_id[1], number, os.fsencode(path)))
    finally:
        os.close(fd)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", required=True)
    parser.add_argument("--device", required=True)
    args = parser.parse_args()
    try:
        if verify(args.root, args.device):
            print('{"classification":"local-ext4-document-fence","verifiedRoots":1}')
            return 0
    except (OSError, RuntimeError, ValueError, IndexError):
        pass
    print("Local document volume fence could not be verified.", file=sys.stderr)
    return 1


if __name__ == "__main__":
    sys.exit(main())
