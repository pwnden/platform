"""Check Vite's HTTP/WebSocket HMR protocol through the Go development origin."""

import base64
import hashlib
import json
import os
from pathlib import Path
import re
import socket
import struct
import tempfile
import time
from urllib.parse import urlsplit
from urllib.request import build_opener, ProxyHandler


def check_hmr(origin, checkout):
    client = build_opener(ProxyHandler({}))

    def fetch(path):
        with client.open(origin + path, timeout=10) as response:
            assert response.headers.get("Cache-Control") == "no-store"
            return response.read().decode()

    page = fetch("/")
    assert '/@vite/client' in page and '/src/main.ts' in page
    nonce = re.search(r'name="pwnden-style-nonce" content="([a-f0-9]{64})"', page)
    assert nonce and f'property="csp-nonce" nonce="{nonce[1]}"' in page
    vite = fetch("/@vite/client")
    match = re.search(r'const wsToken = "([A-Za-z0-9_-]+)";', vite)
    assert match, "Vite WebSocket token missing"
    parsed = urlsplit(origin)
    with tempfile.NamedTemporaryFile(mode="w+", prefix="pwnden-hmr-", suffix=".vue",
                                     dir=Path(checkout) / "web/apps/player/src", encoding="utf-8") as fixture:
        content = '<template><p>initial</p></template>\n'
        fixture.write(content)
        fixture.flush()
        path = "/src/" + Path(fixture.name).name
        # Let the polling watcher finish the file-add event before registering
        # this module. The following mutation must exercise an existing SFC.
        time.sleep(0.6)
        assert "initial" in fetch(path)
        with socket.create_connection((parsed.hostname, parsed.port), timeout=10) as connection:
            key = base64.b64encode(os.urandom(16)).decode()
            request = (f"GET /__vite_hmr?token={match[1]} HTTP/1.1\r\n"
                       f"Host: {parsed.netloc}\r\nOrigin: {origin}\r\n"
                       "Connection: Upgrade\r\nUpgrade: websocket\r\n"
                       f"Sec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n"
                       "Sec-WebSocket-Protocol: vite-hmr\r\n\r\n")
            connection.sendall(request.encode())
            with connection.makefile("rb") as stream:
                assert b"101 Switching Protocols" in stream.readline()
                headers = {}
                while (line := stream.readline()) != b"\r\n":
                    assert line, "HMR handshake closed"
                    name, value = line.decode().split(":", 1)
                    headers[name.lower()] = value.strip()
                accept = base64.b64encode(hashlib.sha1((key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11").encode()).digest()).decode()
                assert headers["sec-websocket-accept"] == accept
                assert headers["sec-websocket-protocol"] == "vite-hmr"

                def message():
                    header = stream.read(2)
                    assert len(header) == 2 and header[0] == 0x81 and not header[1] & 0x80
                    length = header[1] & 0x7F
                    if length == 126:
                        length = struct.unpack("!H", stream.read(2))[0]
                    elif length == 127:
                        length = struct.unpack("!Q", stream.read(8))[0]
                    assert length < 1 << 20
                    return json.loads(stream.read(length))

                assert message()["type"] == "connected"
                fixture.seek(0)
                fixture.write(content.replace("initial", "updated"))
                fixture.truncate()
                fixture.flush()
                os.fsync(fixture.fileno())
                for _ in range(16):
                    update = message()
                    assert update["type"] != "error", update
                    if update["type"] == "update" and any(item["path"] == path for item in update["updates"]):
                        break
                else:
                    raise AssertionError("Vue SFC update was not delivered")
                assert "updated" in fetch(path)
