# Local web and API server

After managed setup, start the server from the platform checkout:

```sh
./pwnden serve
```

A native package uses `./pwnden serve` from its unpacked directory. `serve` accepts no repository, address, port, or positional arguments. It checks that the managed catalog can be loaded before opening a listener. Missing setup reports `setup_required`.

The server prints one private session URL such as `http://127.0.0.1:49152/#<token>`. Open the entire URL. The listener uses an available IPv4 loopback port. Its bearer token is regenerated for each process and must be kept private. The URL fragment reaches the bootstrap script, which removes it from the current history entry and sets `window.pwndenSession` to `{token, apiBase}` in memory. The script writes no cookie or browser storage. Reopening the printed full URL initializes another page; a refresh of the URL after fragment removal does not restore credentials.

The current root page initializes the session and shows a placeholder. The Vue problem screen, file access, progress history, and interactive terminal are subsequent capabilities. The implemented [API](web-api.md) lists problems, starts and stops them, and checks submitted flags. Go serves both the bootstrap assets and API; the frontend build will supply its static assets to this server.

API calls require `Authorization: Bearer <token>`. Their Host must match the printed authority. When present, Origin must exactly match the printed HTTP origin; browser fetch-site values allow `same-origin` or top-level `none`. Reads are authenticated too. The server provides no CORS access. Requests cannot choose a problem root, image, mount, or Docker option.

Mutations for the same slug are serialized, including submissions. Independent problems can proceed concurrently. A queued request that is canceled never calls its backend. Setup and CLI mutations are run outside an active server session; cross-process coordination is a subsequent capability.

SIGINT and SIGTERM stop the listener and cancel active requests. Shutdown allows ten seconds for HTTP connections to drain, closes remaining connections after that limit, and still waits for independent problem cleanup. Each API operation has a fifteen-minute deadline. HTTP headers and request bodies have short read deadlines. The server has no overall fifteen-minute lifetime limit.

An interrupted new start, or a failed response write/flush after startup, triggers independent stop while the problem mutation lock is held. Cleanup failures retain their application causes and are reported to local diagnostics without process output, flags, tokens, or host paths. Completed runs survive server shutdown and can be submitted or stopped after restarting the server. `already_running` preserves an existing run.

## Verification

Maintainers use Go and Python for these checks; the packaged server requires Docker rather than host SDKs:

```sh
go test ./...
go vet ./...
go test -race ./internal/httpapi
go run ./tools/package --challenges ../challenges --out dist --target linux/amd64
python3 -B tools/smoke_http.py --package dist/pwnden-linux-amd64.tar.gz
```

Unit tests cover access checks, strict JSON input, response privacy, queued cancellation, concurrent independent problems, failed delivery cleanup, and shutdown waiting for disconnected handlers. The real Docker check installs a managed catalog, exercises both problem kinds, solves the web problem through its published endpoint, checks incorrect and correct submissions, restarts the server, verifies credential rotation and preserved run state, and stops repeatedly. Only Docker is present on the packaged child's PATH; Go, Git, Node, pnpm, and Python are absent. The smoke harness itself runs as a maintainer tool on Linux/WSL. Public clone download verification remains the separate distribution gate.
