# pwnden platform

Local wargames with prepared problem environments and flag checks.

## Get started

Download and unpack the platform package for your system. Keep its files together. Docker with Linux containers and Compose must be installed and running.

From the unpacked folder:

```sh
./pwnden setup
./pwnden list
./pwnden run note-vault
```

Setup prepares the included problems and their execution tools. Host Go, Git, Python, and a separate problem checkout are unnecessary. The platform manages problem files and paths.

Open the URL printed by `run`. Note Vault is an introductory web problem: log in as `guest` / `guest` and find the administrator's recovery key, formatted as `pwnden{...}`.

```sh
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

File problem tools need no service startup. Their problem directory is read-only by default, and their toolbox has no network access. `exec` preserves command arguments and returns the command's exit status. It runs completed commands; an interactive terminal is a later feature.

`./pwnden --help` lists commands. Windows packages use `pwnden.exe`. The current verified package runs on Linux; Windows and macOS actual host checks are a later stage.

## Development

Development commands require Go 1.27.1, Docker with Compose, and a local problem checkout.

```sh
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
