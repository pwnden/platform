# Platform application interface

`internal/application` owns the platform's callable operations. CLI and future HTTP handlers consume these operations and format their own responses. The challenges checkout owns the versioned [problem contract](https://github.com/pwnden/challenges/blob/main/docs/contract.md); this application interface is maintained by the platform and follows its implementation releases.

## Construction and capabilities

`application.New(repo)` binds a service to an explicit challenges checkout. The path is required; construction performs no filesystem or Docker operations. `application.Open()` instead resolves the managed problem snapshot installed by `application.NewSetup().Setup(ctx)`. Each problem operation loads the current problem through the existing contract consumer and execution policy. Callers provide a non-nil `context.Context`. Repository-relative paths resolve using the process working directory, so callers keep that directory stable during operations.

Consumers depend on the capabilities they need. Core execution capabilities are:

```go
type Validator interface {
    Validate(context.Context, string) (Validation, error)
}

type Runner interface {
    Run(context.Context, string) (RunInfo, error)
    Stop(context.Context, string) (StopInfo, error)
}

type Verifier interface {
    Verify(context.Context, string) (Verification, error)
}
```

`Service` implements all three, plus `Catalog.List(ctx)`, `Details.Detail(ctx, slug)`, `Files.Download(ctx, slug, id)`, `Observer.Status(ctx, slug)` and `Player.Execute(ctx, slug, command)` / `Player.Submit(ctx, slug, flag)`. `SetupService` implements the separate `Bootstrapper.Setup(ctx)` capability. Operations return Go data rather than console output, HTTP status codes, or Docker configuration objects. Command parsing and transport serialization belong to adapters. New capabilities have their own interfaces and result types; existing consumers keep their current dependencies.

The player HTTP capability set, JSON DTOs, handlers, and local server are implemented in `internal/httpapi`, with the versioned [HTTP contract](web-api.md) and [OpenAPI document](openapi.json). Its backend consumes catalog, detail, file, observation, execution and submission capabilities directly. Browser response mapping includes only published endpoints and uses public messages for typed errors. The adapter serializes mutations and status reads by slug and owns cleanup for starts whose responses cannot be delivered. [Local server](local-server.md) documents its lifecycle.

## Inputs and results

Problem operations take a slug; `Execute` also takes an argument array and `Submit` a candidate flag. `List` and `Setup` take no slug. Problem lookup, contract version support, allowed paths, and Docker resource boundaries retain the existing runtime rules.

| Operation | Result fields | Success meaning |
| --- | --- | --- |
| `Validate` | `Slug`, `Kind`, `ContractVersion`, `FileCount`, `ServiceCount`, `Patched` | Inputs are loadable; service configuration and any patch configuration satisfy the execution policy. No service is started. |
| `Run` | `Slug`, `Kind`, `FileCount`, `Project`, `Endpoints` | Service startup completed and endpoint addresses were resolved; file problems require no service. |
| `Stop` | `Slug` | The problem's service and patch resources and saved state were removed. File problems require no service cleanup. |
| `Verify` | `Slug`, `Flag`, `Patched` | The declared automatic solution passed, along with patch attack, functional check, and patch cleanup when present. |
| `List` | Slice of `Problem`: `Slug`, `Title`, `Category`, `Kind` | Available declarations loaded; display metadata uses strings with the slug as a missing title fallback. Author metadata validation remains with the problem owner. |
| `Detail` | `Problem`, `Description`, `Files` | Optional problem README text and declared regular distribution files loaded. Each file has opaque `ID`, display `Name` and byte `Size`. |
| `Download` | `Name`, `Size`, `Content` | A current declared file is opened for reading. `Content` is an `io.ReadSeekCloser`; the caller closes it. |
| `Status` | `Slug`, `Kind`, `State`, `Endpoints` | Stored run state and Docker containers observed; no resources started, stopped or erased. |
| `Execute` | `Stdout`, `Stderr`, `ExitCode` | The requested toolbox command completed with exit code 0–124; runtime failures return an error. |
| `Submit` | `Slug`, `Accepted` | The candidate flag was compared with the file hash or recorded service flag. Incorrect flags are ordinary results. |
| `Setup` | `ProblemCount`, `ContractVersion` | Docker and Compose are available; the bundled snapshot passed integrity, compatibility, and execution checks; images were prepared and the managed snapshot activated. |

`Kind` is `"file"` or `"service"`. File results have no project or endpoints, and validation reports zero services. `FileCount` counts distribution entries, including entries that name directories. `ServiceCount` counts the vulnerable configuration's services. `Validation.Patched` indicates that a patch is declared; `Verification.Patched` indicates that its complete verification passed.

Each endpoint contains `Name`, `URL`, and `Published`. Published addresses come from Docker's actual mapped port; unpublished addresses refer to the container network. Successful `RunInfo` contains no generated flag. `Verification.Flag` is the recovered solution for the author verification workflow; a future player API defines its own response and access policy.

Any error returns the zero result; failed verification exposes no partial flag. `Run` rolls back a startup whose endpoint lookup fails, including its saved state. Existing running problems produce `already_running`; successful verification keeps the vulnerable run available. `Stop` permits missing distribution files and may be called repeatedly, but still requires enough valid problem and Compose declarations to identify and check cleanup resources.

## Errors

Use `errors.As(err, &applicationError)` with `var applicationError *application.Error`. Its fields are `Code`, `Operation`, `Slug`, and `Cause`. `Operation` identifies `configure`, `setup`, `list`, `detail`, `download`, `status`, `validate`, `run`, `stop`, `verify`, `exec`, or `submit`. `Unwrap` preserves underlying filesystem, contract, process, cancellation, and joined cleanup errors for `errors.Is` and `errors.As`.

| Code | Meaning |
| --- | --- |
| `invalid_argument` | Empty repository path, invalid slug, or unloadable declaration other than a missing input or unsupported version. |
| `not_found` | A repository, problem, or required declared input is missing during loading. |
| `incompatible_contract` | Repository or problem contract version is unsupported. |
| `already_running` | A service run is already recorded. |
| `not_running` | A service operation requires saved run state that is absent. |
| `validation_failed` | Resolved Compose validation could not complete or rejected the execution configuration. |
| `execution_failed` | Startup, endpoint discovery, or user toolbox execution failed. |
| `verification_failed` | Automatic solution or patch verification failed, including execution errors without a more specific code. |
| `cleanup_failed` | Resource teardown or state removal failed; inspect the cause for resource identities and all failures. |
| `canceled` | Work was canceled and cleanup did not fail. |
| `deadline_exceeded` | The caller deadline or toolbox runtime limit expired and cleanup did not fail. |
| `setup_required` | The managed problem snapshot is absent or incomplete; run setup. |
| `setup_failed` | Bundled content loading, integrity checking, Docker availability, image preparation, or managed activation failed. |

Classification uses typed errors and operation boundaries. Adapters map these codes to their own presentation. Cleanup failure takes precedence over cancellation or timeout because resources may remain. Both causes remain inspectable.

## Cancellation and resource ownership

The caller chooses the operation deadline. The application service adds no global timeout. The toolbox still applies the problem's execution timeout after image preparation. Work observes cancellation; startup failures and interrupted tool execution attempt independent cleanup before returning. Each Compose teardown has a two-minute limit and interrupted toolbox removal has a thirty-second limit. A stop involving both projects can therefore take longer than one teardown limit.

Explicit `Stop` completes cleanup with an independent context even when its caller is canceled. It reports success only after cleanup and state removal complete. Failed teardown retains saved state for another attempt. File preparation and cleanup create no Docker resources.

The service does not coordinate simultaneous callers. Adapters serialize mutating operations and `Status` for a given repository and slug; concurrent `Run`, `Stop`, and `Verify` calls for the same problem are outside this interface's supported usage. The HTTP adapter enforces that ownership. Detail and download readers use repository-confined filesystem access; editing declarations during execution remains unsupported.

## Details, files and observation

The platform presents the existing problem `README.md` as optional UTF-8 text, up to 1 MiB. A missing README produces an empty description. The browser displays it as text. This is a platform presentation convention; the challenges-owned version 1 execution contract remains intact.

`Detail` expands each declared `files` entry into regular files, including nested directories and repository-contained symlink targets. Directory cycles, special files and outside targets fail. IDs are SHA-256 identifiers of normalized repository-relative logical paths, stable across process restarts and content edits at the same path. They are lookup identifiers, not content checksums or access tokens. Downloads re-enumerate current declarations, select only a matching ID and open the target through `os.Root`. Names and sizes describe distribution files; solution and configuration files appear only if authors explicitly distribute them. Shared files inside the problem repository remain supported.

File status is `ready`: it needs no service run. A service with no recorded run is `stopped`. For a recorded run, `Status` checks the current execution policy and reads Compose containers. It reports `running` only when every declared service has running containers and every reported health check is healthy; missing, exited or unhealthy containers produce `unavailable`. This observation does not check application behavior or reconcile resources. Docker and corrupt-state errors return typed failures, retaining saved state. Endpoints are resolved only while running. A recorded unavailable run must be explicitly stopped before a new start.

## Managed setup and player commands

Setup uses the fixed archive identity embedded by the package builder. It activates the installed snapshot after all preparations succeed, preserves the current active snapshot on failure, and rejects switching snapshots while a previous service run is recorded. Sequential setup reuses the same snapshot and image caches. [Distribution](distribution.md) defines the storage and extraction boundaries.

`Execute` uses the declared solution image as the player's toolbox. It retains the same problem-directory mount permissions, network selection, image preparation, runtime deadline, and independent cleanup as automatic verification. It accepts direct command arguments and captures completed output. A service command checks the saved run and execution policy before using its project network.

`Submit` trims leading and trailing Unicode whitespace and uses the same flag comparison strategy as automatic verification. A service submission requires the current saved run flag. This operation performs no solution execution and writes no progress record.
