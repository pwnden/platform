# Local web and API server

After managed setup, start the server from the platform checkout:

```sh
./pwnden serve
```

For source development, use `./pwnden dev`. Docker builds the Go development
executable and starts Vite with live frontend sources. This same Go server proxies
Vite assets and HMR WebSockets while handling authenticated API and terminal
requests itself. Open its printed full URL. Frontend edits update through HMR;
restart after Go or dependency changes. Verification runs separately, and problem
images are prepared when used. Local problem edits are read directly. Its source
root and resource identities are separate from the managed installation. Ctrl+C
removes the development frontend container and its dependency volumes. Native Windows
uses `./pwnden.ps1 dev`; actual Windows and macOS checks are deferred.

A native package uses `./pwnden serve` from its unpacked directory. `serve` accepts no repository, address, port, or positional arguments. It checks that the managed catalog can be loaded before opening a listener. Missing setup reports `setup_required`.

The server prints one private session URL such as `http://127.0.0.1:49152/#<token>`. Open the entire URL. The listener uses an available IPv4 loopback port. Its bearer token is regenerated for each process and must be kept private. The Vue entry point removes the fragment from the current history entry before rendering and injects the token into the API client in memory. It writes no cookie or browser storage. Reopening the printed full URL initializes another page; refreshing after fragment removal shows recovery instructions.

The root page serves the Vue player. Choose a problem, read its README description and download its declared distribution files. Service controls display observed run status: start a stopped service, open a published HTTP endpoint, submit a flag and stop it. File problems need no service startup. The [interactive terminal](terminal.md) opens the problem's toolbox shell; service problems must be running. Disconnect, shell exit and server shutdown remove the terminal container, while completed service runs remain available. Reopening the printed full URL and selecting a problem recovers service state and endpoints from Go, including after a server restart. TCP endpoints are displayed as addresses for terminal tools. Answer checking remains in Go through the implemented [API](web-api.md). Progress history is a subsequent capability. HTML, JavaScript and CSS are embedded in the executable; serving them requires neither a separate asset directory nor Node. See [frontend](frontend.md) for build preparation.

API calls require `Authorization: Bearer <token>`. Their Host must match the printed authority. When present, Origin must exactly match the printed HTTP origin; browser fetch-site values allow `same-origin` or top-level `none`. Reads are authenticated too. The server provides no CORS access. Requests cannot choose a problem root, image, mount, or Docker option.

Mutations and status reads for the same slug are serialized, including submissions. Independent problems can proceed concurrently. A queued request that is canceled never calls its backend. Detail and file reads are independent of the mutation lock. Setup and CLI mutations are run outside an active server session; cross-process coordination is a subsequent capability.

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

Unit tests cover embedded asset delivery, MIME and path/origin boundaries, declared files/directories and repository-contained links, authenticated attachments, description text, observed Docker states, strict JSON input, response privacy, queued cancellation, concurrent independent problems, failed delivery cleanup, and shutdown waiting for disconnected handlers. Frontend tests cover fragment removal, invalid credentials, client feature lifecycle, downloads and state recovery. The real Docker check retrieves HTML, JavaScript and CSS without exposing credentials, installs a managed catalog, downloads a declared file and compares its bytes with the bundled source, exercises both problem kinds, solves the web problem through its published endpoint, checks incorrect and correct submissions, restarts the server, verifies credential rotation and preserved observed state, detects an externally stopped owned container without erasing state, and stops repeatedly. Only Docker is present on the packaged child's PATH; Go, Git, Node, pnpm, and Python are absent. The smoke harness itself runs as a maintainer tool on Linux/WSL. Browser interaction checks and public clone download verification are distinct gates.
