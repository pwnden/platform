# Platform application interface

`internal/application` owns the platform's callable operations. CLI and future HTTP handlers consume these operations and format their own responses. The challenges checkout owns the versioned [problem contract](https://github.com/pwnden/challenges/blob/main/docs/contract.md); this application interface is maintained by the platform and follows its implementation releases.

## Construction and capabilities

`application.New(repo)` binds a service to one challenges checkout. The path is required; construction performs no filesystem or Docker operations. Each operation loads the current problem through the existing contract consumer and execution policy. Callers provide a non-nil `context.Context` and the problem's slug. Repository-relative paths resolve using the process working directory, so callers keep that directory stable during operations.

Consumers depend on the capabilities they need:

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

`Service` implements all three. Operations return Go data rather than console output, HTTP status codes, or Docker configuration objects. Command parsing and transport serialization belong to adapters. New capabilities can have their own interfaces and result types; existing consumers keep their current dependencies.

## Inputs and results

All four operations take a slug. Problem lookup, contract version support, allowed paths, and Docker resource boundaries retain the existing runtime rules.

| Operation | Result fields | Success meaning |
| --- | --- | --- |
| `Validate` | `Slug`, `Kind`, `ContractVersion`, `FileCount`, `ServiceCount`, `Patched` | Inputs are loadable; service configuration and any patch configuration satisfy the execution policy. No service is started. |
| `Run` | `Slug`, `Kind`, `FileCount`, `Project`, `Endpoints` | Service startup completed and endpoint addresses were resolved; file problems require no service. |
| `Stop` | `Slug` | The problem's service and patch resources and saved state were removed. File problems require no service cleanup. |
| `Verify` | `Slug`, `Flag`, `Patched` | The declared automatic solution passed, along with patch attack, functional check, and patch cleanup when present. |

`Kind` is `"file"` or `"service"`. File results have no project or endpoints, and validation reports zero services. `FileCount` counts distribution entries, including entries that name directories. `ServiceCount` counts the vulnerable configuration's services. `Validation.Patched` indicates that a patch is declared; `Verification.Patched` indicates that its complete verification passed.

Each endpoint contains `Name`, `URL`, and `Published`. Published addresses come from Docker's actual mapped port; unpublished addresses refer to the container network. Successful `RunInfo` contains no generated flag. `Verification.Flag` is the recovered solution for the author verification workflow; a future player API defines its own response and access policy.

Any error returns the zero result; failed verification exposes no partial flag. `Run` rolls back a startup whose endpoint lookup fails, including its saved state. Existing running problems produce `already_running`; successful verification keeps the vulnerable run available. `Stop` permits missing distribution files and may be called repeatedly, but still requires enough valid problem and Compose declarations to identify and check cleanup resources.

## Errors

Use `errors.As(err, &applicationError)` with `var applicationError *application.Error`. Its fields are `Code`, `Operation`, `Slug`, and `Cause`. `Operation` is `configure`, `validate`, `run`, `stop`, or `verify`. `Unwrap` preserves underlying filesystem, contract, process, cancellation, and joined cleanup errors for `errors.Is` and `errors.As`.

| Code | Meaning |
| --- | --- |
| `invalid_argument` | Empty repository path, invalid slug, or unloadable declaration other than a missing input or unsupported version. |
| `not_found` | A repository, problem, or required declared input is missing during loading. |
| `incompatible_contract` | Repository or problem contract version is unsupported. |
| `already_running` | A service run is already recorded. |
| `not_running` | A service operation requires saved run state that is absent. |
| `validation_failed` | Resolved Compose validation could not complete or rejected the execution configuration. |
| `execution_failed` | Startup or endpoint discovery failed. |
| `verification_failed` | Automatic solution or patch verification failed, including execution errors without a more specific code. |
| `cleanup_failed` | Resource teardown or state removal failed; inspect the cause for resource identities and all failures. |
| `canceled` | Work was canceled and cleanup did not fail. |
| `deadline_exceeded` | The caller deadline or toolbox runtime limit expired and cleanup did not fail. |

Classification uses typed errors and operation boundaries. Adapters map these codes to their own presentation. Cleanup failure takes precedence over cancellation or timeout because resources may remain. Both causes remain inspectable.

## Cancellation and resource ownership

The caller chooses the operation deadline. The application service adds no global timeout. The toolbox still applies the problem's execution timeout after image preparation. Work observes cancellation; startup failures and interrupted tool execution attempt independent cleanup before returning. Each Compose teardown has a two-minute limit and interrupted toolbox removal has a thirty-second limit. A stop involving both projects can therefore take longer than one teardown limit.

Explicit `Stop` completes cleanup with an independent context even when its caller is canceled. It reports success only after cleanup and state removal complete. Failed teardown retains saved state for another attempt. File preparation and cleanup create no Docker resources.

The service does not coordinate simultaneous callers. Adapters serialize mutating operations for a given repository and slug; concurrent `Run`, `Stop`, and `Verify` calls for the same problem are outside this interface's supported usage. Adding a concurrent HTTP adapter requires enforcing that ownership before exposing these operations.
