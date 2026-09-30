# Execution verification

The platform verifies the problems it consumes. With `platform` and `challenges` checked out as siblings, run this command from the platform repository:

```sh
python3 tools/verify.py --challenges ../challenges
```

The host needs Python 3.11 or newer, the Go version declared in `go.mod`, Docker with Linux containers, and the Compose plugin. The verifier builds this checkout's CLI once, discovers the supplied problem manifests, and runs `validate`, `run`, `verify`, and `stop` for each one.

The CLI implements [problem contract version 1](https://github.com/pwnden/challenges/blob/main/docs/contract.md), checks each problem's version, and reads execution defaults and the attack rejection exit code from the challenges checkout's `contract.toml`. Problem format validation belongs to the challenges repository. The CLI checks real paths and Docker resource policy before execution, verifies flags and patch behavior, and cleans up its projects.

Start with the problems stopped before running the complete check. After a successful start, the verifier stops the problem even if verification fails. On interruption it signals the active CLI, waits for its cleanup, and stops the problem it started. A failed start performs its own cleanup; a pre-existing run remains owned by its original caller.

## GitHub Actions

`.github/workflows/verify.yml` checks out the platform revision under test and `pwnden/challenges` at `main`, runs Go tests and vet, and runs the same execution verifier. It uses read-only repository permissions and checkouts without persisted credentials. Action versions are pinned to exact commits.

The workflow runs on pushes to the platform's `main`, pull requests, manual dispatches, and daily at 18:00 UTC (03:00 Korea time). The daily run checks changes to the consumed problem repository. Publish both repositories' implementations before confirming GitHub-hosted execution. The platform workflow uses its own checked-out code for the runner.
