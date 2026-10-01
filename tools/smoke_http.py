"""Verify the packaged local player API with real Docker and no host SDKs."""

import argparse
from http.cookiejar import CookieJar
import hashlib
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
import tomllib
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
        flag = json.load(response)["body"]
    with client.open(endpoint + "/notes/1", timeout=10) as response:
        assert flag not in response.read().decode()
    with client.open(endpoint + "/notes/2", timeout=10) as response:
        assert flag in response.read().decode(), "browser walkthrough failed to retrieve current flag"
    return flag


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
            assert "localStorage" not in content and "sessionStorage" in content


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


def installed_catalog(work):
    catalogs = list((work / "config" / "pwnden" / "catalogs").glob("*/catalog"))
    assert len(catalogs) == 1
    return catalogs[0]


def check_catalog(work, problems):
    expected = {manifest.parent.name for manifest in installed_catalog(work).glob("challenges/*/challenge.toml")}
    actual = [problem["slug"] for problem in problems]
    assert expected and set(actual) == expected and len(actual) == len(expected), "HTTP catalog differs from installed problems"


def terminal(driver, origin, token, work, slug, mode):
    settings = {"origin": origin, "token": token, "root": str(installed_catalog(work)), "slug": slug, "mode": mode}
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
        for mutation in (("stop", "note-vault"), ("exec", "rotor-lock", "--", "true"), ("serve",)):
            blocked = subprocess.run([str(binary), *mutation], env=environment, capture_output=True, text=True, timeout=10)
            assert blocked.returncode != 0 and not blocked.stdout, "CLI bypassed installation ownership"
        check_assets(origin, token)
        problems = api(origin, token, "GET", "/problems")["problems"]
        check_catalog(work, problems)
        assert api(origin, "incorrect", "GET", "/problems", expected=401)["error"]["code"] == "unauthorized"
        detail = api(origin, token, "GET", "/problems/rotor-lock")
        assert detail["description"] and len(detail["files"]) == 1
        with tarfile.open(binary.with_name("catalog.tar.gz"), "r:gz") as archive:
            name = next(name for name in archive.getnames() if name.endswith("/contract.toml"))
            with archive.extractfile(name) as source:
                content_expected = tomllib.loads(source.read().decode())["version"] >= 2
        assert detail["walkthrough"] == content_expected, "catalog content availability disagrees with contract"
        if content_expected:
            assert detail["hint_count"] == 3
            assert "go run" not in detail["description"] and file_flag not in detail["description"]
            for slug in ("rotor-lock", "note-vault"):
                brief = api(origin, token, "GET", f"/problems/{slug}")
                assert brief["hint_count"] == 3 and brief["walkthrough"]
                assert "go run" not in brief["description"] and "solve/README.md" not in brief["description"]
                for id in ("hint-1", "hint-2", "hint-3", "walkthrough"):
                    content = api(origin, token, "GET", f"/problems/{slug}/guidance/{id}")
                    assert content["id"] == id and content["content"].strip()
                assert api(origin, token, "GET", f"/problems/{slug}/guidance/hint-4", expected=404)["error"]["code"] == "not_found"
                assert api(origin, "incorrect", "GET", f"/problems/{slug}/guidance/walkthrough", expected=401)["error"]["code"] == "unauthorized"
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
        if args.terminal_driver:
            terminal(args.terminal_driver.resolve(), origin, token, work, "note-vault", "disconnect")
            assert api(origin, token, "GET", "/problems/note-vault/status")["state"] == "running"
            api(origin, token, "DELETE", "/problems/note-vault/run")
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
        browser = api(origin, token, "POST", "/problems/note-vault/browser", {"name": run["endpoints"][0]["name"]})
        wrapper = urlsplit(browser["url"])
        assert browser["target"] == endpoint and wrapper.hostname == "127.0.0.1" and wrapper.port
        assert f"http://{wrapper.netloc}" == endpoint and endpoint != origin
        with build_opener(ProxyHandler({})).open(browser["url"], timeout=10) as response:
            assert token not in response.read().decode()
        assert solve_web(f"http://{wrapper.netloc}") == flag, "common browser proxy changed the exercise"
        assert api(origin, token, "POST", "/problems/note-vault/run", expected=409)["error"]["code"] == "already_running"
        assert not api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": "wrong"})["accepted"]
        assert api(origin, token, "POST", "/problems/note-vault/submissions", {"flag": flag})["accepted"]
        if args.terminal_driver:
            terminal(args.terminal_driver.resolve(), origin, token, work, "note-vault", "disconnect")
        stop(server)
        server = None
        # Normal shutdown cleans owned runs; a new process rotates credentials.
        server, origin, next_token = start(binary, environment)
        assert next_token != token
        assert api(origin, token, "GET", "/problems", expected=401)["error"]["code"] == "unauthorized"
        assert api(origin, next_token, "GET", "/problems/note-vault/status")["state"] == "stopped"
        assert api(origin, next_token, "GET", "/problems/rotor-lock")["files"] == detail["files"]
        assert api(origin, next_token, "POST", "/problems/note-vault/submissions", {"flag": flag}, expected=409)["error"]["code"] == "not_running"
        api(origin, next_token, "POST", "/problems/note-vault/run")
        if args.terminal_driver:
            for slug in ("rotor-lock", "note-vault"):
                terminal(args.terminal_driver.resolve(), origin, next_token, work, slug, "retain")
            workspaces = api(origin, next_token, "GET", "/workspaces")
            assert workspaces["limit"] == 10 and workspaces["idle_seconds"] == 600
            assert {entry["slug"] for entry in workspaces["workspaces"]} == {"rotor-lock", "note-vault"}
            # Kill only this isolated test server, then recover its journal.
            server.kill()
            server.communicate(timeout=10)
            server = None
            server, origin, next_token = start(binary, environment)
            assert api(origin, next_token, "GET", "/workspaces")["workspaces"] == []
            assert api(origin, next_token, "GET", "/problems/note-vault/status")["state"] == "stopped"
            catalogs = list((work / "config" / "pwnden" / "catalogs").glob("*/catalog"))
            identity = hashlib.sha256(str(catalogs[0].resolve()).encode()).hexdigest()
            leftovers = subprocess.run([docker, "ps", "-aq", "--filter", f"label=pwnden.repository={identity}"], check=True, capture_output=True, text=True, timeout=30).stdout.strip()
            assert not leftovers, "recovery left an owned toolbox behind"
            api(origin, next_token, "POST", "/problems/note-vault/run")
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
