"""Check catalog pinning through the real checkout entry without host SDKs."""

import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile


def main():
    source = Path(__file__).resolve().parent.parent
    with tempfile.TemporaryDirectory(prefix="pwnden-catalog-") as temporary:
        work = Path(temporary)
        root = work / "platform with spaces"
        root.mkdir()
        for name in ("pwnden", "go.mod", "Dockerfile.bootstrap", "Dockerfile.bootstrap.dockerignore"):
            shutil.copy2(source / name, root / name)
        for name in ("internal/checkout", "tools/checkout"):
            shutil.copytree(source / name, root / name)
        lock = root / "catalog.lock"
        lock.write_text("0" * 40 + "\n")
        path = work / "path"
        path.mkdir()
        for name in ("docker", "sh", "dirname", "uname", "mkdir", "mktemp", "rm"):
            executable = shutil.which(name)
            if not executable:
                raise RuntimeError(f"catalog entry requires {name}")
            (path / name).symlink_to(Path(executable).resolve())
        environment = dict(os.environ, PATH=str(path))
        for name in ("git", "go", "node", "pnpm", "python3"):
            assert shutil.which(name, path=environment["PATH"]) is None

        def invoke(*args, success=True):
            result = subprocess.run([str(root / "pwnden"), "catalog", *args],
                                    cwd=work, env=environment, capture_output=True,
                                    text=True, timeout=240)
            if success and result.returncode:
                raise RuntimeError(f"catalog command failed: {result.stderr}")
            if not success and not result.returncode:
                raise RuntimeError("unpublished commit was accepted")
            assert list((root / "dist").iterdir()) == [], "catalog update retained generated output"
            assert not list(root.glob(".catalog-*")), "catalog update retained temporary lock files"
            return result

        result = invoke("update")
        selected = lock.read_bytes()
        assert re.fullmatch(rb"[a-f0-9]{40}\n", selected)
        assert selected != b"0" * 40 + b"\n"
        assert "Catalog pinned" in result.stdout
        before = lock.stat()
        result = invoke("update", "--revision", selected.decode().strip())
        assert lock.read_bytes() == selected and lock.stat().st_ino == before.st_ino
        assert "already pinned" in result.stdout
        invoke("update", "--revision", "0" * 40, success=False)
        assert lock.read_bytes() == selected, "failed fetch changed the catalog"
        print("Catalog smoke passed: published main, explicit revision, failed-fetch preservation and cleanup; no host SDKs.")


if __name__ == "__main__":
    main()
