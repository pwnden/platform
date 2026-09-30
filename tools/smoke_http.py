"""Verify the packaged local player API with real Docker and no host SDKs."""

import argparse
from http.cookiejar import CookieJar
import json
import os
from pathlib import Path
import queue
import re
import shutil
import signal
import subprocess
import tempfile
import tarfile
import threading
from urllib.error import HTTPError
from urllib.parse import urlsplit
from urllib.request import HTTPCookieProcessor, ProxyHandler, Request, build_opener

from smoke_package import extract_package


def command(binary, environment, *args):
    result = subprocess.run([str(binary), *args], env=environment,
                            capture_output=True, text=True, timeout=300)
    if result.returncode:
        raise RuntimeError(f"{args[0]} failed: {result.stderr}")
    return result.stdout.strip()


def start(binary, environment):
    server = subprocess.Popen([str(binary), "serve"], env=environment,
                              stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    announcement = queue.Queue()
    threading.Thread(target=lambda: announcement.put(server.stdout.readline()), daemon=True).start()
    try:
        parsed = urlsplit(announcement.get(timeout=10).strip())
        if parsed.hostname != "127.0.0.1" or not parsed.port or len(parsed.fragment) != 64:
            raise RuntimeError("server did not print a loopback session URL")
        return server, f"http://{parsed.netloc}", parsed.fragment
    except BaseException:
        stop(server)
        raise


def stop(server):
    if server.poll() is None:
        server.send_signal(signal.SIGTERM)
    output, diagnostic = server.communicate(timeout=300)
    if server.returncode:
        raise RuntimeError(f"serve failed: {diagnostic}")
    if diagnostic or output:
        raise RuntimeError("unexpected server diagnostics")


def api(origin, token, method, path, data=None, expected=200):
    headers = {"Authorization": f"Bearer {token}", "Origin": origin,
               "Sec-Fetch-Site": "same-origin"}
    content = None
    if data is not None:
        content = json.dumps(data).encode()
        headers["Content-Type"] = "application/json"
    client = build_opener(ProxyHandler({}))
    try:
        response = client.open(Request(origin + "/api/v1" + path, content, headers, method=method), timeout=180)
    except HTTPError as failure:
        response = failure
    with response:
        result = json.load(response)
        if response.status != expected or response.headers.get("Cache-Control") != "no-store":
            raise AssertionError(f"{method} {path}: HTTP {response.status}, {result}")
        return result


def solve_web(endpoint):
    client = build_opener(ProxyHandler({}), HTTPCookieProcessor(CookieJar()))
    login = Request(endpoint + "/login", json.dumps({"username": "guest", "password": "guest"}).encode(),
                    {"Content-Type": "application/json"})
    with client.open(login, timeout=10) as response:
        assert response.status == 200
    with client.open(endpoint + "/api/notes/2", timeout=10) as response:
        return json.load(response)["body"]


def check_assets(origin, token):
    client = build_opener(ProxyHandler({}))
    with client.open(origin + "/", timeout=10) as response:
        page = response.read().decode()
        assert response.headers.get("Content-Type") == "text/html; charset=utf-8"
        assert response.headers.get("Cache-Control") == "no-store"
        assert "default-src 'self'" in response.headers.get("Content-Security-Policy", "")
    assert 'id="app"' in page and 'type="module"' in page
    nonce = re.search(r'name="pwnden-style-nonce" content="([a-f0-9]{64})"', page)
    assert nonce
    assert "/session.js" not in page and token not in page
    assets = re.findall(r'(?:src|href)="(/assets/[^" ]+)"', page)
    assert len(assets) >= 2
    for path in assets:
        with client.open(origin + path, timeout=10) as response:
            content = response.read().decode()
            media = "text/javascript" if path.endswith(".js") else "text/css"
            assert response.headers.get("Content-Type") == media + "; charset=utf-8"
            assert response.headers.get("Cache-Control") == "no-store"
            assert response.headers.get("X-Content-Type-Options") == "nosniff"
        assert content and token not in content
        if path.endswith(".js"):
            assert "history.replaceState" in content and "/api/v1" in content
            assert "localStorage" not in content and "sessionStorage" not in content


def download(origin, token, slug, file):
    headers = {"Authorization": f"Bearer {token}", "Origin": origin,
               "Sec-Fetch-Site": "same-origin"}
    client = build_opener(ProxyHandler({}))
    with client.open(Request(f"{origin}/api/v1/problems/{slug}/files/{file['id']}", headers=headers), timeout=30) as response:
        content = response.read()
        assert response.headers.get("Content-Type") == "application/octet-stream"
        assert response.headers.get_content_disposition() == "attachment"
        assert response.headers.get_filename() == "checker.py"
        assert response.headers.get("Cache-Control") == "no-store"
        assert response.headers.get("X-Content-Type-Options") == "nosniff"
        assert len(content) == file["size"]
        return content


def interrupted(_signum, _frame):
    raise KeyboardInterrupt


def terminal(driver, origin, token, work, slug, mode):
    catalogs = list((work / "config" / "pwnden" / "catalogs").glob("*/catalog"))
    assert len(catalogs) == 1
    settings = {"origin": origin, "token": token, "root": str(catalogs[0]), "slug": slug, "mode": mode}
    result = subprocess.run([str(driver)], input=json.dumps(settings), capture_output=True, text=True, timeout=120)
    if result.returncode:
        raise RuntimeError(result.stderr)
    print(result.stdout.strip(), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
    parser.add_argument("--terminal-driver", type=Path)
    args = parser.parse_args()
    package = args.package.resolve()
    docker = shutil.which("docker")
    if not docker:
        parser.error("Docker is required")
    signal.signal(signal.SIGINT, interrupted)
    signal.signal(signal.SIGTERM, interrupted)
    work = Path(tempfile.mkdtemp(prefix=".http-smoke-", dir=package.parent))
    server = None
    preserve = False
    owned_run = False
    try:
        binary = extract_package(package, work)
        path = work / "path"
        path.mkdir()
        (path / "docker").symlink_to(Path(docker).resolve())
        environment = dict(os.environ, PATH=str(path), XDG_CONFIG_HOME=str(work / "config"),
                           XDG_CACHE_HOME=str(work / "cache"))
        for name in ("go", "git", "node", "pnpm", "python3"):
            assert shutil.which(name, path=environment["PATH"]) is None
        command(binary, environment, "setup")
        # CLI mutations complete before opening the HTTP session.
        file_flag = command(binary, environment, "exec", "rotor-lock", "--", "python3", "solve/solve.py")
        server, origin, token = start(binary, environment)
        check_assets(origin, token)
        problems = api(origin, token, "GET", "/problems")["problems"]
        assert {p["slug"] for p in problems} == {"note-vault", "rotor-lock"}
        assert api(origin, "incorrect", "GET", "/problems", expected=401)["error"]["code"] == "unauthorized"
        detail = api(origin, token, "GET", "/problems/rotor-lock")
        assert "Rotor Lock" in detail["description"] and len(detail["files"]) == 1
        file = detail["files"][0]
        assert file["name"] == "files/checker.py" and len(file["id"]) == 64
        with tarfile.open(binary.with_name("catalog.tar.gz"), "r:gz") as archive:
            name = next(name for name in archive.getnames() if name.endswith("rotor-lock/files/checker.py"))
            with archive.extractfile(name) as source:
                assert download(origin, token, "rotor-lock", file) == source.read()
        assert api(origin, "incorrect", "GET", f"/problems/rotor-lock/files/{file['id']}", expected=401)["error"]["code"] == "unauthorized"
        assert api(origin, token, "GET", "/problems/rotor-lock/files/" + "0" * 64, expected=404)["error"]["code"] == "not_found"
        assert api(origin, token, "GET", "/problems/rotor-lock/status") == {"slug": "rotor-lock", "kind": "file", "state": "ready", "endpoints": []}
        if args.terminal_driver:
            for mode in ("exit", "disconnect"):
                terminal(args.terminal_driver.resolve(), origin, token, work, "rotor-lock", mode)
        assert api(origin, token, "GET", "/problems/note-vault")["files"] == []
        assert api(origin, token, "GET", "/problems/note-vault/status")["state"] == "stopped"
        assert api(origin, token, "POST", "/problems/rotor-lock/run")["endpoints"] == []
        assert not api(origin, token, "POST", "/problems/rotor-lock/submissions", {"flag": "wrong"})["accepted"]
        assert api(origin, token, "POST", "/problems/rotor-lock/submissions", {"flag": file_flag})["accepted"]
        owned_run = True
        run = api(origin, token, "POST", "/problems/note-vault/run")
        assert "project" not in run and len(run["endpoints"]) == 1
        endpoint = run["endpoints"][0]["url"].rstrip("/")
        observed = api(origin, token, "GET", "/problems/note-vault/status")
        assert observed["state"] == "running" and observed["endpoints"] == run["endpoints"]
        flag = solve_web(endpoint)
        assert api(origin, token, "POST", "/problems/note-vault/run", expected=409)["error"]["code"] == "already_running"
        assert not api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": "wrong"})["accepted"]
        assert api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": flag})["accepted"]
        if args.terminal_driver:
            terminal(args.terminal_driver.resolve(), origin, token, work, "note-vault", "disconnect")
        stop(server)
        server = None
        # Completed runs retain their state; a new process rotates credentials.
        server, origin, next_token = start(binary, environment)
        assert next_token != token
        assert api(origin, token, "GET", "/problems", expected=401)["error"]["code"] == "unauthorized"
        assert api(origin, next_token, "GET", "/problems/note-vault/status") == observed
        assert api(origin, next_token, "GET", "/problems/rotor-lock")["files"] == detail["files"]
        assert api(origin, next_token, "POST", "/problems/note-vault/submissions", {"flag": flag})["accepted"]
        # Stop only this smoke run's service outside the platform, then observe
        # the actual container state without erasing its recorded ownership.
        receipts = list((work / "cache" / "pwnden" / "runs").glob("*.json"))
        assert len(receipts) == 1
        project = json.loads(receipts[0].read_text())["project"]
        assert re.fullmatch(r"pwnden-note-vault-[a-f0-9]{8}", project)
        containers = subprocess.run([docker, "ps", "--filter", f"label=com.docker.compose.project={project}",
                                     "--format", "{{.ID}}"], check=True, capture_output=True, text=True, timeout=30).stdout.split()
        assert len(containers) == 1
        subprocess.run([docker, "stop", containers[0]], check=True, capture_output=True, timeout=30)
        unavailable = api(origin, next_token, "GET", "/problems/note-vault/status")
        assert unavailable["state"] == "unavailable" and unavailable["endpoints"] == []
        assert receipts[0].exists()
        assert api(origin, next_token, "POST", "/problems/note-vault/run", expected=409)["error"]["code"] == "already_running"
        for _ in range(2):
            assert api(origin, next_token, "DELETE", "/problems/note-vault/run") == {"slug": "note-vault"}
        if args.terminal_driver:
            api(origin, next_token, "POST", "/problems/note-vault/run")
            terminal(args.terminal_driver.resolve(), origin, next_token, work, "note-vault", "stop")
        owned_run = False
        assert api(origin, next_token, "GET", "/problems/note-vault/status")["state"] == "stopped"
        assert api(origin, next_token, "POST", "/problems/note-vault/submissions", {"flag": flag}, expected=409)["error"]["code"] == "not_running"
        print("HTTP player smoke check passed with Docker and no host SDKs.", flush=True)
    finally:
        if server is not None:
            try:
                stop(server)
            except BaseException:
                preserve = True
        if owned_run and not preserve:
            try:
                command(binary, environment, "stop", "note-vault")
                owned_run = False
            except BaseException:
                preserve = True
        if preserve or owned_run:
            print(f"Cleanup incomplete; keep package and state at {work}", flush=True)
            raise RuntimeError("HTTP smoke cleanup failed")
        shutil.rmtree(work)


if __name__ == "__main__":
    main()
