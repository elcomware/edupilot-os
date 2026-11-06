import importlib.util
from pathlib import Path
import unittest

spec = importlib.util.spec_from_file_location(
    "document_fence", Path(__file__).resolve().parents[1] / "verify-local-document-fence.py")
fence = importlib.util.module_from_spec(spec)
spec.loader.exec_module(fence)

ROOT = b"/backup volume/documents"
MOUNT = rb"77 2 7:1 / /backup\040volume ro,relatime - ext4 /dev/loop1 ro"


class DocumentFenceTests(unittest.TestCase):
    def test_read_only_superblock(self):
        self.assertTrue(fence.check_mounts(MOUNT, b"77", b"7:1", ROOT))

    def test_unqualified_mounts(self):
        for table in [
            MOUNT.replace(b"ext4", b"nfs4"), MOUNT.replace(b"7:1", b"7:2"),
            MOUNT.replace(b"77 ", b"78 "), MOUNT.replace(b"/ /backup", b"/slice /backup"),
            MOUNT.replace(b"1 ro", b"1 rw"), MOUNT.replace(b" ro,", b" rw,"),
            MOUNT + b"\n" + rb"88 77 0:9 / /backup\040volume/documents/sub ro - tmpfs tmpfs rw",
        ]:
            with self.subTest(table=table):
                self.assertFalse(fence.check_mounts(table, b"77", b"7:1", ROOT))
