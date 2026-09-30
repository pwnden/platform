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

## Native packages

A native platform package contains the executable, `catalog.tar.gz`, and `distribution.json`. The frontend is embedded in the executable. Keep those files together for the first setup. Host Go, Git, Node, pnpm, Python, and a separate problem checkout are unnecessary. Docker with Linux containers and the Compose plugin are prerequisites.

## Setup and managed content

`pwnden setup` checks Docker and Compose, verifies the bundled archive against a SHA-256 embedded in the executable, extracts the fixed problem snapshot, checks contract compatibility and the existing execution policy, and downloads/builds the declared images. It prepares vulnerable and patch images and toolboxes without starting problem services.

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
It builds current platform/frontend sources inside Docker and calls the native
CLI with the live local challenges path. `Service.Prepare` applies the same
contract and execution policy and prepares images without starting services.
The same Go HTTP server serves the embedded frontend and local problem operations.
Local problem edits need no commit. Platform/frontend changes take effect after
stopping and rerunning `dev`; Docker caches unchanged inputs. Ctrl+C stops the
server and its terminal sessions; completed problem services retain their state.

Development builds live in `dist/dev.*/payload/`; `dist/development` identifies
the latest built executable. `dev` leaves the official `dist/active` reference
and managed installation unchanged. Source CLI checks use
`./pwnden --repo ../challenges <command>` with the development build. Ordinary
player commands prefer the official build. Missing or invalid referenced
executables fail explicitly.

The development build prepares the live checkout without host bind mounts or
a Docker socket in the builder. The native runtime mounts only validated
problem paths as usual. Native packages keep the separate fixed content
identity flow. [Execution verification](verification.md) describes author checks.

```sh
go build -o dist/smoke-terminal ./tools/smoke_terminal
python3 -B tools/smoke_dev.py --checkout . --terminal-driver dist/smoke-terminal
```

This maintainer check runs `dev` twice with host SDKs absent from the player's
PATH. It checks cache reuse, local HTTP content and file bytes, real isolated
terminal behavior and cleanup, independent official activation and server shutdown.
