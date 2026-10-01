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

The server prints one private session URL such as `http://127.0.0.1:49152/#<token>`. Open the entire URL. The listener uses an available IPv4 loopback port. Its bearer token is regenerated for each process and must be kept private. The Vue entry point removes the fragment from the current history entry before rendering, retains the credential in origin-scoped tab `sessionStorage`, and injects it into the API client. Reloads and development HMR in the same tab keep the connection. A fresh full URL replaces the saved credential. An HTTP 401 or terminal authentication rejection clears that credential and presents connection recovery. An uninitialized tab also displays recovery; it does not claim the session has expired. When tab storage is disabled, the full URL works for that page and must be reopened after a reload. Credentials are never sent to a separate frontend origin or stored in cookies or `localStorage`.

The root page serves the Vue player. Choose a problem and read its description,
hints and walkthrough in the reading pane. Submit the discovered flag there.
The right-side tool pane provides Terminal, Files for declared materials, and Web
for published local HTTP services. Files supports source preview and optional
download. Web shows the problem in an iframe and offers a new-tab link for browser
developer tools. Switching tools retains the terminal, source and web state.

The [interactive terminal](terminal.md) automatically prepares the selected
problem and connects its toolbox shell. Its connection switch controls attachment;
the power button immediately terminates the whole problem environment. Problem
switching, page refresh and hidden browser tabs retain the same shell and service
state for ten inactive minutes. Returning cancels expiry. Up to ten problem
environments are retained; the player explicitly ends one when capacity is reached.
Shell exit removes the toolbox while a visible problem retains its services.
Environment stop and normal server shutdown remove both. TCP endpoints are
displayed as addresses for terminal tools. Answer checking remains in Go through
the implemented [API](web-api.md). Progress history is a subsequent capability.
HTML, JavaScript and CSS are embedded in the executable; serving them requires
neither a separate asset directory nor Node. See [frontend](frontend.md) for build
preparation.

API calls require `Authorization: Bearer <token>`. Their Host must match the printed authority. When present, Origin must exactly match the printed HTTP origin; browser fetch-site values allow `same-origin` or top-level `none`. Reads are authenticated too. The server provides no CORS access. Requests cannot choose a problem root, image, mount, or Docker option.

Mutations and status reads for the same slug are serialized, including submissions. Independent problems can proceed concurrently. A queued request that is canceled never calls its backend. Detail and file reads are independent of the mutation lock. Setup runs before starting a server. An installation lock excludes another server and standalone run, exec, verify and stop mutations during the server session.

SIGINT and SIGTERM stop the listener and cancel active requests. Shutdown allows ten seconds for HTTP connections to drain, closes remaining connections after that limit, and still waits for independent problem cleanup. Each API operation has a fifteen-minute deadline. HTTP headers and request bodies have short read deadlines. The server has no overall fifteen-minute lifetime limit.

An interrupted new start, or a failed response write/flush after startup, triggers independent stop while the problem mutation lock is held. Cleanup failures retain their application causes and are reported to local diagnostics without process output, flags, tokens, or host paths. Normal shutdown cleans all server-managed problem environments. Ownership records persist before creation; after a crash, the next startup cleans recorded environments from the same repository before admitting requests. Failed cleanup retains records and reports failure. `already_running` preserves an existing run.

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

Unit tests cover embedded asset delivery, MIME and path/origin boundaries, declared files/directories and repository-contained links, authenticated attachments, description text, observed Docker states, strict JSON input, response privacy, queued cancellation, concurrent independent problems, failed delivery cleanup, and shutdown waiting for disconnected handlers. Frontend tests cover fragment removal, invalid credentials, client feature lifecycle, downloads and state recovery. The real Docker check retrieves HTML, JavaScript and CSS without exposing credentials, installs a managed catalog, downloads a declared file and compares its bytes with the bundled source, exercises both problem kinds, solves the web problem through its published endpoint, checks incorrect and correct submissions, restarts the server, verifies credential rotation and full shutdown cleanup, retains shell state across attachment changes, and exercises crash recovery, detects an externally stopped owned container without erasing state, and stops repeatedly. Only Docker is present on the packaged child's PATH; Go, Git, Node, pnpm, and Python are absent. The smoke harness itself runs as a maintainer tool on Linux/WSL. Browser interaction checks and public clone download verification are distinct gates.
