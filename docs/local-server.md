# Local web and API server

After managed setup, start the server from the platform checkout:

```sh
./pwnden serve
```

A native package uses `./pwnden serve` from its unpacked directory. `serve` accepts no repository, address, port, or positional arguments. It checks that the managed catalog can be loaded before opening a listener. Missing setup reports `setup_required`.

The server prints one private session URL such as `http://127.0.0.1:49152/#<token>`. Open the entire URL. The listener uses an available IPv4 loopback port. Its bearer token is regenerated for each process and must be kept private. The Vue entry point removes the fragment from the current history entry before rendering and injects the token into the API client in memory. It writes no cookie or browser storage. Reopening the printed full URL initializes another page; refreshing after fragment removal shows recovery instructions.

The root page serves the Vue player. Choose a problem, start or stop it, open a published HTTP endpoint and submit a flag. TCP endpoints are displayed as addresses for terminal tools. Answer checking remains in Go through the implemented [API](web-api.md). File access, progress history and an interactive terminal are subsequent capabilities. HTML, JavaScript and CSS are embedded in the executable; serving them requires neither a separate asset directory nor Node. See [frontend](frontend.md) for build preparation.

API calls require `Authorization: Bearer <token>`. Their Host must match the printed authority. When present, Origin must exactly match the printed HTTP origin; browser fetch-site values allow `same-origin` or top-level `none`. Reads are authenticated too. The server provides no CORS access. Requests cannot choose a problem root, image, mount, or Docker option.

Mutations for the same slug are serialized, including submissions. Independent problems can proceed concurrently. A queued request that is canceled never calls its backend. Setup and CLI mutations are run outside an active server session; cross-process coordination is a subsequent capability.

SIGINT and SIGTERM stop the listener and cancel active requests. Shutdown allows ten seconds for HTTP connections to drain, closes remaining connections after that limit, and still waits for independent problem cleanup. Each API operation has a fifteen-minute deadline. HTTP headers and request bodies have short read deadlines. The server has no overall fifteen-minute lifetime limit.

An interrupted new start, or a failed response write/flush after startup, triggers independent stop while the problem mutation lock is held. Cleanup failures retain their application causes and are reported to local diagnostics without process output, flags, tokens, or host paths. Completed runs survive server shutdown and can be submitted or stopped after restarting the server. `already_running` preserves an existing run.

## Verification

Maintainers use Go and Python for these checks; the packaged server requires Docker rather than host SDKs:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
go test ./...
go vet ./...
go test -race ./internal/httpapi
go run ./tools/package --challenges ../challenges --out dist --target linux/amd64
python3 -B tools/smoke_http.py --package dist/pwnden-linux-amd64.tar.gz
```

Unit tests cover embedded asset delivery, MIME and path/origin boundaries, access checks, strict JSON input, response privacy, queued cancellation, concurrent independent problems, failed delivery cleanup, and shutdown waiting for disconnected handlers. Frontend tests cover fragment removal and invalid credentials. The real Docker check retrieves HTML, JavaScript and CSS without exposing credentials, installs a managed catalog, exercises both problem kinds, solves the web problem through its published endpoint, checks incorrect and correct submissions, restarts the server, verifies credential rotation and preserved run state, and stops repeatedly. Only Docker is present on the packaged child's PATH; Go, Git, Node, pnpm, and Python are absent. The smoke harness itself runs as a maintainer tool on Linux/WSL. Browser interaction checks and public clone download verification are distinct gates.
