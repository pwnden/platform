# Platform distribution

The main entry point is a platform checkout with automated setup. Maintainers can also build native packages. Both flows use the same managed installation and execution policy.

## Clone and setup

Install Git and Docker with Linux containers, Compose, and Buildx, clone the
platform, and run `./pwnden setup`. The entry script builds a small native Go
checkout launcher inside Docker. That common launcher checks Docker, builds
the Vue frontend and Go platform, acquires the official revision in
`catalog.lock`, and invokes the generated executable's setup. Host Go, Node,
pnpm, and Python are unnecessary. Then run `./pwnden serve` and open its printed
full session URL to use the [local Vue player](local-server.md).

`catalog.lock` contains a full commit hash. The builder fetches exactly that commit from the official public problem repository, exports it with `git archive`, and embeds its archive checksum and revision in the executable. A missing commit fails acquisition. Publish the pinned problem commit before testing the public clone path. End users acquire no separate checkout and supply no problem path.

Builder images fix Node 24.21.0 and Go 1.27.1 to OCI image digests, with pnpm 12.8.1 pinned and Go toolchain downloads disabled. The frontend uses a frozen lockfile and its assets are copied into the Go embed directory. Docker's build cache reuses unchanged source and dependency layers. The context includes declared platform sources, frontend manifests/lockfile and the catalog lock; host dependency and generated asset directories are excluded. No host directory or Docker socket is mounted into the builder. Export writes generated files inside the platform checkout.

Each setup attempt creates `dist/build.*/payload/` containing `pwnden`,
`catalog.tar.gz`, and `distribution.json`. After build and problem setup succeed,
the Go checkout launcher atomically replaces `dist/active`. Failed attempts
remove their own new output and preserve the previous reference. Successful
previous builds remain under `dist/`. Arguments and streams pass to the native
player; cancellation gives it time to clean up before forcing an exit.
Generated output is ignored by Git. Generated paths reject symlink redirects.

Repeat `./pwnden setup` after source updates. Docker's cache reuses unchanged
layers; managed setup preserves installed edits for the same content identity.
Commands and setup run sequentially. POSIX `pwnden` and native Windows
`pwnden.ps1` only select the native build target, build the entry tool, and
invoke it. Build, preparation, activation and command routing live in Go.
Actual Windows and macOS execution checks remain a later stage.

```sh
python3 -B tools/test_bootstrap.py
python3 -B tools/smoke_package.py --checkout .
```

The first check covers activation, retry, failed build/preparation recovery and output boundaries with a Docker double. The second requires the published official commit and a real Docker daemon; it runs checkout setup and player workflows with Go, Git, Node, pnpm, and host Python absent from the application's PATH.

## Selecting the published catalog

Maintainers select the problem revision through the checkout entry:

```sh
./pwnden catalog update
```

The command fetches the published `pwnden/challenges` main branch and atomically updates `catalog.lock` to its full commit. To select a particular published commit, including an older revision, use `./pwnden catalog update --revision <full-40-character-commit>`. On native Windows the same arguments go to `./pwnden.ps1`.

Git runs in the existing pinned Docker toolchain; this operation needs no host Git, Go, Node, pnpm, Python or local challenges checkout. It builds only the catalog export target, without building the application or frontend. A fresh Git fetch runs on every update, so Docker cache reuse cannot hide changes to main. Failed acquisition, malformed output, cancellation or a detected edit to the lock during acquisition preserves the current file. Temporary exports are removed. An already selected commit leaves the file untouched.

Commit the resulting `catalog.lock` change with the platform. After receiving that platform revision, players run `./pwnden setup`, then `./pwnden serve`. Catalog selection and installation are separate operations: updating the lock leaves the active installation and running problems unchanged. Publish a challenges commit before selecting it; an unpublished commit fails explicitly. The normal update has a three-minute acquisition deadline. Invoke checkout operations sequentially.

Use `./pwnden catalog --help` for command help. Native packages retain their bundled revision; catalog selection belongs to the source checkout. `python3 -B tools/smoke_catalog.py` checks the real Docker entry with a temporary checkout, published and unpublished revisions, and a child PATH without host SDKs.

## Native packages

A native platform package contains the executable, `catalog.tar.gz`, and `distribution.json`. The frontend is embedded in the executable. Keep those files together for the first setup. Host Go, Git, Node, pnpm, Python, and a separate problem checkout are unnecessary. Docker Engine 28 or newer with Linux containers and the Compose plugin are prerequisites.

## Setup and managed content

`pwnden setup` checks Docker and Compose, verifies the bundled archive against a SHA-256 embedded in the executable, extracts the fixed problem snapshot, checks contract compatibility and the execution policy, and downloads/builds the declared images. It prepares vulnerable and patch images, toolboxes and the pinned platform connector image without starting problem services. [Network isolation](network-isolation.md) applies at runtime to every supported catalog version.

Installation data lives under the OS user configuration directory's `pwnden` folder. The active installation receipt identifies a content revision and archive digest. Commands resolve the managed problem directory from that receipt. Users choose problem names; paths and repository acquisition are managed by the platform.

The installed snapshot is the allowed problem root for host mounts and build inputs. The same real-path checks, host privilege exclusions, and project resource scopes apply. This is the existing Docker execution boundary for trusted official problems. Host paths outside that root are excluded by those mount and build rules.

Extraction accepts regular files, directories, and symlinks whose targets resolve inside the snapshot. It rejects path traversal, ambiguous paths, duplicate entries, hard links, special files, and links that escape the root. The private installation parent remains private; mounted problem files retain readable modes for container tools.

The active receipt is replaced only after preparation succeeds. Failed first preparation discards its new snapshot and leaves setup pending. Repeated setup reuses the installed snapshot and preserves edits. Docker image/build caches may remain after failed preparation and are reused on retry. Changing to another content revision requires stopping recorded service runs first. Setup and mutating problem operations are invoked sequentially.

The platform package carries a fixed content revision, so package use does not depend on a branch changing at setup time. An installed matching snapshot can be reused without the original archive. The executable checks the active problem contract when operations load a problem.

## Building packages

Maintainers need Go, Git, Docker and a problem checkout. From the platform repository:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
go run ./tools/package --challenges ../challenges --out dist --target linux/amd64
```

The builder exports the checkout's exact `HEAD` with `git archive`, compresses that snapshot, embeds its revision and digest into a static Go executable, and packages both together. Uncommitted and untracked problem files are excluded. The output includes a package SHA-256 file and metadata recording the content revision and digest.

Supported build targets are `linux`, `darwin`, and `windows`, each with `amd64` or `arm64`. Windows packages are ZIP; the others are tar.gz. Target support in the builder is distinct from actual host execution verification. The current host and CI exercise the native Linux package; Windows and macOS execution checks are a later stage.

```sh
python3 -B tools/smoke_package.py --package dist/pwnden-linux-amd64.tar.gz
```

The native Linux check extracts the package into its own directory, exposes Docker as the only executable on the child's PATH, and runs setup, repeat setup, listing, container tools, submitted flag checks, automatic verification, and service cleanup. It keeps the temporary distribution and state if service cleanup fails. Go, Git, and host Python are absent from the application process's PATH; Python is only the development check driver.

Platform CI verifies the consumed problems, builds the native Linux package, runs its package check, and checks the clone setup path against the pinned official commit. CI builds and tests locally on its runner.

## Development builds

Keep `platform` and `challenges` next to each other, then run `./pwnden dev`.
It builds the Go executable with the `development` build tag and starts Vite in
Docker. The tag provides source CLI commands without an embedded production web
build. The native CLI reads the live challenges catalog; problem execution keeps
its contract and mount-policy checks and prepares images when needed. The Go
server proxies frontend assets and HMR WebSockets on the same loopback origin as
the API and interactive terminal. Vue, TypeScript and CSS edits update through
HMR. Restart after Go or dependency changes; Docker caches unchanged inputs.
Verification runs separately. Ctrl+C stops the server and terminal sessions and
removes the Vite container and anonymous volumes; completed problem services retain
their state. Local problem edits need no commit.

Development builds live in `dist/dev.*/payload/`; `dist/development` identifies
the latest built executable. `dev` leaves the official `dist/active` reference
and managed installation unchanged. Source CLI checks use
`./pwnden --repo ../challenges <command>` with the development build. Ordinary
player commands prefer the official build. Missing or invalid referenced
executables fail explicitly.

The frontend tool image uses only workspace manifests and the frozen lockfile.
The Vite container mounts `platform/web` read-only, with host dependency directories
covered by anonymous Docker volumes. The launcher creates empty ignored
`node_modules` directories when needed as volume mount points. Dependencies and
Vite caches stay inside those disposable volumes. Polling detects changes across
Docker file sharing environments. The container uses the checkout user's UID/GID
where available, drops Linux capabilities and publishes only on loopback. Its
builder and runtime have no Docker socket. The native challenge runtime mounts
only validated problem paths as usual. Native packages keep the separate fixed content
identity flow. [Execution verification](verification.md) describes author checks.

```sh
go build -o dist/smoke-terminal ./tools/smoke_terminal
python3 -B tools/smoke_dev.py --checkout . --terminal-driver dist/smoke-terminal
```

This maintainer check runs `dev` twice with host SDKs absent from the player's
PATH. It checks cache reuse, live Vue SFC HMR through Go's WebSocket proxy, local
HTTP content and file bytes, real isolated terminal behavior and cleanup,
independent official activation and server shutdown. It also confirms that dev
performs neither full frontend verification nor eager problem-image preparation.
