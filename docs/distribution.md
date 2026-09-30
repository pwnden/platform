# Platform distribution

End users receive one platform package. It contains the native `pwnden` executable, `catalog.tar.gz`, and `distribution.json`. Keep those files together for the first setup. Go, Git, Python, and a separate problem checkout are unnecessary on the host. Docker with Linux containers and the Compose plugin are prerequisites.

## Setup and managed content

`pwnden setup` checks Docker and Compose, verifies the bundled archive against a SHA-256 embedded in the executable, extracts the fixed problem snapshot, checks contract compatibility and the existing execution policy, and downloads/builds the declared images. It prepares vulnerable and patch images and toolboxes without starting problem services.

Installation data lives under the OS user configuration directory's `pwnden` folder. The active installation receipt identifies a content revision and archive digest. Commands resolve the managed problem directory from that receipt. Users choose problem names; paths and repository acquisition are managed by the platform.

The installed snapshot is the allowed problem root for host mounts and build inputs. The same real-path checks, host privilege exclusions, and project resource scopes apply. This is the existing Docker execution boundary for trusted official problems. Host paths outside that root are excluded by those mount and build rules.

Extraction accepts regular files, directories, and symlinks whose targets resolve inside the snapshot. It rejects path traversal, ambiguous paths, duplicate entries, hard links, special files, and links that escape the root. The private installation parent remains private; mounted problem files retain readable modes for container tools.

The active receipt is replaced only after preparation succeeds. Failed first preparation discards its new snapshot and leaves setup pending. Repeated setup reuses the installed snapshot and preserves edits. Docker image/build caches may remain after failed preparation and are reused on retry. Changing to another content revision requires stopping recorded service runs first. Setup and mutating problem operations are invoked sequentially.

The platform package carries a fixed content revision, so package use does not depend on a branch changing at setup time. An installed matching snapshot can be reused without the original archive. The executable checks the active problem contract when operations load a problem.

## Building packages

Maintainers need Go, Git, and a problem checkout. From the platform repository:

```sh
go run ./tools/package --challenges ../challenges --out dist --target linux/amd64
```

The builder exports the checkout's exact `HEAD` with `git archive`, compresses that snapshot, embeds its revision and digest into a static Go executable, and packages both together. Uncommitted and untracked problem files are excluded. The output includes a package SHA-256 file and metadata recording the content revision and digest.

Supported build targets are `linux`, `darwin`, and `windows`, each with `amd64` or `arm64`. Windows packages are ZIP; the others are tar.gz. Target support in the builder is distinct from actual host execution verification. The current host and CI exercise the native Linux package; Windows and macOS execution checks are a later stage.

```sh
python3 -B tools/smoke_package.py --package dist/pwnden-linux-amd64.tar.gz
```

The native Linux check extracts the package into its own directory, exposes Docker as the only executable on the child's PATH, and runs setup, repeat setup, listing, container tools, submitted flag checks, automatic verification, and service cleanup. It keeps the temporary distribution and state if service cleanup fails. Go, Git, and host Python are absent from the application process's PATH; Python is only the development check driver.

Platform CI verifies the consumed problems, builds the native Linux package, and runs the same package check. Public distribution publication is separate from these local build outputs.

## Development builds

Source builds can use `pwnden --repo PATH <command>` to select an explicit checkout. Ordinary unconfigured `go build` executables contain no bundled content identity, so creating an end-user distribution uses the package builder. [Execution verification](verification.md) describes the author and runtime verification procedures.
