# pwnden platform

Local wargames with prepared problem environments and flag checks.

## Get started

Install Git and Docker with Linux containers, Compose, and Buildx, then:

```sh
git clone https://github.com/pwnden/platform.git
cd platform
./pwnden setup
./pwnden serve
```

Setup builds the Vue frontend and Go platform inside Docker, acquires the pinned official problems, and prepares their execution tools. Host Go, Node, pnpm, Python, and a separate problem checkout are unnecessary. The checked-in `pwnden` file is a launcher; generated executables stay under ignored `dist/` paths.

The pinned problem commit in `catalog.lock` must be published in the official problem repository before a fresh clone can complete setup. Missing commits fail explicitly. Open the full URL printed by `serve` to use the [local Vue player](docs/local-server.md): choose a problem, read its description, download its files, check its execution state, run or stop a service and submit a flag. Go serves the embedded frontend and [web API](docs/web-api.md) from the same local origin.

Note Vault is an introductory web problem: run it in the player, open its published HTTP endpoint, log in as `guest` / `guest` and find the administrator's recovery key, formatted as `pwnden{...}`. You can also use the CLI:

```sh
./pwnden run note-vault
./pwnden submit note-vault 'pwnden{your_answer}'
./pwnden stop note-vault
```

Each service run uses a new flag. `stop` removes its service resources and saved run state.

## File problems

Rotor Lock is an introductory reversing problem. Analyze its input checker and find an input that unlocks it. Commands execute inside the prepared problem toolbox:

```sh
./pwnden exec rotor-lock -- cat files/checker.py
./pwnden exec rotor-lock -- python3 files/checker.py 'your_candidate'
./pwnden submit rotor-lock 'pwnden{your_answer}'
```

File problem tools need no service startup. Their problem directory is read-only by default, and their toolbox has no network access. `exec` preserves command arguments and returns the command's exit status. The web player's [interactive terminal](docs/terminal.md) opens a shell in the same isolated toolbox, with input, resize and cleanup on disconnect.

`./pwnden --help` lists commands. Linux/WSL is the current execution target for checkout verification. Windows and macOS actual host checks are a later stage.

## Built packages

Maintainers can also build a platform package containing the native executable and its problem catalog. From an unpacked package, run `./pwnden setup` and then the same player commands. Keep the package files together. Windows packages use `pwnden.exe`. See [distribution](docs/distribution.md) for build and verification commands.

## Development

For development, keep the local challenges checkout next to platform and run:

```sh
./pwnden dev
```

This command builds the Go development executable and starts Vite inside Docker,
with the live local problems exposed through Go. Open its printed full URL.
Vue, TypeScript and CSS edits update through HMR. Restart after Go or dependency
changes; unchanged Docker layers use the build cache. Verification is a separate
maintainer command. Problem images are prepared when used. Ctrl+C closes terminals
and removes the Vite container and temporary dependency volumes. Local problem
edits need no commit or package build. The official installation stays separate.
Host Go, Node and pnpm are unnecessary. On native Windows, use `./pwnden.ps1`
with the same commands; actual Windows and macOS execution checks are deferred.

Maintainer validation commands below require Go 1.27.1, Docker with Compose and
Buildx, and the local problem checkout.

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
go run ./cmd/pwnden --repo ../challenges validate <slug>
go run ./cmd/pwnden --repo ../challenges run <slug>
go run ./cmd/pwnden --repo ../challenges verify <slug>
go run ./cmd/pwnden --repo ../challenges stop <slug>
```

Run these commands from this repository. `--repo` names the root of the challenges checkout. The [problem contract](https://github.com/pwnden/challenges/blob/main/docs/contract.md) defines the metadata, execution, solution and patch results, and resource lifecycle.

The challenges checkout owns the complete specification in `docs/contract.md` and its machine-readable version, defaults, and result code in `contract.toml`. It validates problem metadata in its own CI. This runner supports contract version 1. It checks the repository version and each problem's `schema` before reading execution fields, and reads solution defaults and the attack rejection exit code from the TOML file. Unsupported versions produce a compatibility error. Supporting a new version requires implementing its rules in the runner.

`validate` checks contract compatibility, repository paths, and the resolved Compose execution policy. `run` starts service challenges with a new flag; file challenges have no service to start. `verify` runs the declared solution in its toolbox image, checks the flag, and, when configured, checks the patched version. `stop` removes the challenge's Compose containers, networks, and volumes. The generated flag stays in the local user cache until `stop`; that cache is never mounted into a problem container.

For endpoints with Compose port mappings, `run` prints the actual host address reported by Docker. Challenges can bind `127.0.0.1::8000` to let Docker choose a free local port. Endpoints without published ports are marked `container network` and are accessible through their Compose service names from the solution container.

The runner checks resolved bind mount sources before `up`. Only paths inside the supplied challenges repository are accepted. It also confines build contexts, Dockerfiles, local build cache paths, file-backed configs, and secrets to that repository. Named volumes and bridge networks are project-scoped. Compose services cannot use automatic Docker API socket access, and the toolbox does not mount a Docker socket. Compose files are authored by the challenge owner.

Run every problem's execution, solution, patch, and cleanup check from this repository:

```sh
python3 tools/verify.py --challenges ../challenges
```

See [execution verification and CI](docs/verification.md) for prerequisites and workflow behavior.

The [application interface](docs/application-interface.md) defines shared callable operations, typed results, error codes, and cancellation and cleanup rules for command and HTTP adapters.

See the [command interface](docs/cli.md) for argument rules, help, and adding commands. `go run ./cmd/pwnden --help` lists the current commands.

See [distribution](docs/distribution.md) for managed setup, package building, and end-user package checks.
