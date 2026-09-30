# pwnden platform

Local command-line runner for challenges in the separate `challenges` repository. Requires Go 1.27.1 and Docker Engine with the Compose plugin.

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
