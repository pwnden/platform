"""Verify checkout dev startup, cache reuse, HTTP and terminal without host SDKs."""

import argparse
import json
import os
from pathlib import Path
import re
import shutil
import signal
import subprocess
import tempfile
import time
from urllib.parse import urlsplit
from urllib.error import URLError
from urllib.request import urlopen

from smoke_http import api, download
from smoke_hmr import check_hmr


def interrupted(_signum, _frame):
    raise KeyboardInterrupt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--checkout", type=Path, default=Path.cwd())
    parser.add_argument("--terminal-driver", type=Path, required=True)
    args = parser.parse_args()
    root = args.checkout.resolve()
    repo = root.parent / "challenges"
    driver = args.terminal_driver.resolve()
    original = (root / "dist/active").read_bytes() if (root / "dist/active").exists() else None
    with tempfile.TemporaryDirectory(prefix=".smoke-dev-", dir=root / "dist") as temporary:
        work = Path(temporary)
        tool_path = work / "path"
        tool_path.mkdir()
        for name in ("docker", "sh", "dirname", "uname", "mkdir", "mktemp", "rm"):
            executable = shutil.which(name)
            if not executable:
                parser.error(f"development entry requires {name}")
            (tool_path / name).symlink_to(Path(executable).resolve())
        environment = dict(os.environ, PATH=str(tool_path),
                           XDG_CONFIG_HOME=str(work / "config"), XDG_CACHE_HOME=str(work / "cache"))
        for name in ("go", "git", "node", "pnpm", "python3"):
            assert shutil.which(name, path=environment["PATH"]) is None
        signal.signal(signal.SIGINT, interrupted)
        signal.signal(signal.SIGTERM, interrupted)
        for attempt in range(2):
            output = work / f"stdout-{attempt}"
            diagnostic = work / f"stderr-{attempt}"
            server = None
            origin = None
            try:
                with output.open("w") as stdout, diagnostic.open("w") as stderr:
                    server = subprocess.Popen([str(root / "pwnden"), "dev"], env=environment,
                                              cwd=work, stdout=stdout, stderr=stderr)
                deadline = time.monotonic() + 300
                while time.monotonic() < deadline:
                    text = output.read_text()
                    match = re.search(r"^http://127\.0\.0\.1:\d+/#([a-f0-9]{64})$", text, re.MULTILINE)
                    if match:
                        parsed = urlsplit(match.group())
                        origin, token = f"http://{parsed.netloc}", parsed.fragment
                        break
                    if server.poll() is not None:
                        raise RuntimeError("pwnden dev exited before startup; no credentials printed\n" + diagnostic.read_text())
                    time.sleep(0.05)
                if origin is None:
                    raise RuntimeError("pwnden dev startup timed out")
                for path in ("/", "/challenges", "/challenges/diagnostic-port?q=nmap"):
                    with urlopen(origin + path, timeout=10) as response:
                        assert response.status == 200
                        assert response.headers.get_content_type() == "text/html"
                        assert b"<!doctype html>" in response.read().lower()
                problems = api(origin, token, "GET", "/problems")
                assert {"rotor-lock", "note-vault"} <= {item["slug"] for item in problems["problems"]}
                check_hmr(origin, root)
                assert "pnpm verify" not in diagnostic.read_text(), "dev ran full verification"
                assert "Preparing local problems" not in output.read_text(), "dev eagerly prepared problem images"
                detail = api(origin, token, "GET", "/problems/rotor-lock")
                assert detail["description"] == (repo / "challenges/rotor-lock/README.md").read_text()
                file = next(item for item in detail["files"] if item["name"] == "files/checker.py")
                assert download(origin, token, "rotor-lock", file) == (repo / "challenges/rotor-lock/files/checker.py").read_bytes()
                result = subprocess.run([str(driver)], input=json.dumps({"origin": origin, "token": token,
                                        "root": str(repo), "slug": "rotor-lock", "mode": "exit"}),
                                        capture_output=True, text=True, timeout=120)
                if result.returncode:
                    raise RuntimeError(result.stderr)
                if attempt == 1:
                    assert "CACHED" in diagnostic.read_text(), "unchanged source did not reuse Docker cache"
                after = (root / "dist/active").read_bytes() if (root / "dist/active").exists() else None
                assert original == after, "dev changed the official build reference"
            finally:
                if server is not None:
                    was_running = server.poll() is None
                    if was_running:
                        server.send_signal(signal.SIGTERM)
                    try:
                        server.wait(timeout=60)
                    except subprocess.TimeoutExpired:
                        server.kill()
                        server.wait()
                        raise RuntimeError("development shutdown did not finish")
                    if was_running and server.returncode != 143:
                        raise RuntimeError("development entry did not preserve signal exit status")
                    if origin:
                        try:
                            api(origin, token, "GET", "/problems")
                        except URLError:
                            pass
                        else:
                            raise RuntimeError("development server survived entry shutdown")
            print(f"Development smoke passed: {'first' if attempt == 0 else 'cached'} startup, HTTP, live HMR, terminal, shutdown; no host SDKs.", flush=True)


if __name__ == "__main__":
    main()
