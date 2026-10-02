# Platform integration verification

Challenges completes [authoring, execution verification and publication](https://github.com/pwnden/challenges/blob/main/docs/publishing.md) in its own repository using Python and Docker. Platform independently consumes published catalogs and verifies their integration with its own runtime, CLI and player. These platform checks require the platform's development toolchain; challenge authors finish their publication gates with the challenges tools.

With `platform` and a published `challenges` catalog checked out as siblings, run this command from the platform repository:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
python3 tools/verify.py --challenges ../challenges
```

The host needs Python 3.11 or newer, the Go version declared in `go.mod`, Docker Engine 28 or newer with Linux containers, and the Compose plugin. The first command builds embedded frontend assets with pinned tools inside Docker. The verifier builds this checkout's CLI once, discovers the supplied problem manifests, and runs `validate`, `run`, `verify`, and `stop` for each one.

Compose must support `start --wait` so containers can be created and their mounts
checked before startup. CI installs checksum-verified Compose 5.1.3 and builds the
selected basic and pcap CLI images from the challenges checkout. For local checks,
prepare those images using the catalog's `images/cli/README.md` first.

The CLI implements [problem contract version 5](https://github.com/pwnden/challenges/blob/main/docs/contract.md), retains version 1, 2, 3 and 4 execution compatibility with the current isolation policy, checks each problem's version, and reads execution defaults and the attack rejection exit code from the challenges checkout's `contract.toml`. Problem format and learning-content validation belong to the challenges repository. The CLI checks real paths and Docker resource policy before execution, verifies flags and patch behavior, and cleans up its projects.

Start with the problems stopped before running the complete check. After a successful start, the verifier stops the problem even if verification fails. On interruption it signals the active CLI, waits for its cleanup, and stops the problem it started. A failed start performs its own cleanup; a pre-existing run remains owned by its original caller.

## Actual network isolation

On Linux/WSL with a locally accessible Docker bridge, run:

```sh
PWNDEN_TEST_CHALLENGES="$(realpath ../challenges)" go test ./internal/httpapi -run '^TestIsolatedNoteVaultDocker$' -count=1 -v
```

The test copies Note Vault into a temporary catalog and uses a separate cache.
It checks Internet IPv4, IPv6 and DNS destinations, a reachable host listener
and a reachable service on another network, from both command execution and an
interactive PTY. Same-problem HTTP remains reachable. The full browser ingress
path preserves login cookies, flag recovery and submission; vulnerable and
patched verification and container/network teardown must pass. Positive controls
establish that host and peer destinations exist before checking their denial.
The test leaves the player's running projects untouched. See [network isolation](network-isolation.md).

## GitHub Actions

`.github/workflows/verify.yml` checks out the platform revision under test and `pwnden/challenges` at `main`, builds embedded player assets in Docker, runs Go tests and vet, and runs the same execution verifier. It uses read-only repository permissions and checkouts without persisted credentials. Action versions are pinned to exact commits.

The workflow runs on pushes to the platform's `main`, pull requests, manual dispatches, and daily at 18:00 UTC (03:00 Korea time). The daily run checks changes to the published problem repository. Each repository owns its implementation and CI; the platform workflow uses its own checked-out code for the consumer runtime. GitHub-hosted execution is confirmed by the respective workflow runs after publication.

After problem verification, the workflow builds a native Linux platform package and runs `tools/smoke_package.py`. That check uses the package with only Docker on its child PATH, exercising managed setup and player commands without Go, Git, or a separately supplied problem path. See [distribution](distribution.md) for local package checks.

`tools/smoke_http.py` also exercises the packaged local server with only Docker on its child PATH: embedded HTML/script/style delivery, authenticated catalog, both problem kinds, web problem solving, incorrect and correct submissions, preserved state and rotated credentials after server restart, and repeatable stop. This driver checks HTTP behavior; browser interaction is a separate check. See [local server](local-server.md) for the local command and unit/concurrency coverage.

The HTTP catalog check compares the API response with all manifests in the
installed catalog. The same smoke also solves Note Vault through the common
browser proxy and compares its flag with the common HTTP ingress endpoint, checking wrapper
isolation and authenticated preparation. It exercises HTTP; native iframe
history and visual interaction require a Browser Plugin check.

The catalog comparison includes newly added problems. Rotor Lock and Note Vault
exercise representative file and service flows. New declarations join catalog
and execution checks automatically. Run `python3 -B tools/test_smoke_http.py` to
check discovery coverage for additional, missing, unexpected and duplicate
problems without Docker.

The workflow also runs `tools/test_bootstrap.py` for activation and output-boundary failures, then runs `tools/smoke_package.py --checkout .` with only Docker and ordinary shell utilities on the child PATH. The checkout check builds inside Docker and fetches the exact official problem commit from `catalog.lock`. That commit must already be published; unpublished content fails this gate rather than switching revisions.

`tools/smoke_catalog.py` checks catalog selection through the checkout entry in an isolated temporary checkout. With host Git, Go, Node, pnpm and Python absent from the child PATH, it fetches published main, reselects the explicit commit, rejects an unpublished revision and verifies lock preservation and temporary-output cleanup. The workflow runs this check before problem execution. It leaves the checked-in catalog and active installation untouched.
