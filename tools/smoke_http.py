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


def interrupted(_signum, _frame):
    raise KeyboardInterrupt


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--package", type=Path, required=True)
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
        assert api(origin, token, "POST", "/problems/rotor-lock/run")["endpoints"] == []
        assert not api(origin, token, "POST", "/problems/rotor-lock/submissions", {"flag": "wrong"})["accepted"]
        assert api(origin, token, "POST", "/problems/rotor-lock/submissions", {"flag": file_flag})["accepted"]
        owned_run = True
        run = api(origin, token, "POST", "/problems/note-vault/run")
        assert "project" not in run and len(run["endpoints"]) == 1
        endpoint = run["endpoints"][0]["url"].rstrip("/")
        flag = solve_web(endpoint)
        assert api(origin, token, "POST", "/problems/note-vault/run", expected=409)["error"]["code"] == "already_running"
        assert not api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": "wrong"})["accepted"]
        assert api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": flag})["accepted"]
        stop(server)
        server = None
        # Completed runs retain their state; a new process rotates credentials.
        server, origin, next_token = start(binary, environment)
        assert next_token != token
        assert api(origin, token, "GET", "/problems", expected=401)["error"]["code"] == "unauthorized"
        assert api(origin, next_token, "POST", "/problems/note-vault/submissions", {"flag": flag})["accepted"]
        for _ in range(2):
            assert api(origin, next_token, "DELETE", "/problems/note-vault/run") == {"slug": "note-vault"}
        owned_run = False
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
