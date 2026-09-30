# Command interface

The CLI consumes the [platform application interface](application-interface.md). The packaged executable selects commands and formats results. End users run `pwnden setup` once, then choose problems by name. [Distribution](distribution.md) describes the package and managed installation.

```text
pwnden [--repo PATH] <command> [command arguments]
```

Global options precede the command. Each command owns its arguments and flags; command flags precede positional arguments, following Go's standard `flag` parsing. Commands use the managed installation by default. `--repo PATH` is a developer override for an explicit checkout. The selected handler constructs the application service after parsing its arguments.

| Command | Invocation | Result |
| --- | --- | --- |
| `setup` | `pwnden setup` | Docker and Compose check, bundled problem installation, and image preparation. |
| `list` | `pwnden list` | Available problem names, categories, and titles. |
| `run` | `pwnden run <slug>` | File distribution count, or service project and actual endpoint addresses. |
| `exec` | `pwnden exec <slug> -- <command> [arguments]` | A completed command's stdout, stderr, and exit status from the declared toolbox. |
| `submit` | `pwnden submit <slug> <flag>` | `Correct.` or `Incorrect.` for the submitted flag. |
| `stop` | `pwnden stop <slug>` | Confirmation after problem resources and state are removed. |
| `validate` | `pwnden validate <slug>` | Contract compatibility and file/service count after execution policy checks. |
| `verify` | `pwnden verify <slug>` | Recovered solution flag and optional patch verification result; this is an author workflow. |

`pwnden --help` lists registered commands and global options. `pwnden <command> --help` shows that command's usage without loading a repository. Help and argument parsing diagnostics use stderr. Successful operation output uses stdout. Application failures print to stderr with the `pwnden:` prefix and exit 1; successful operations and help exit 0. Incorrect submissions exit 1 without an error banner. Completed toolbox commands preserve stdout, stderr, and their exit status from 0 through 124; Docker failures, signals, and timeouts are application failures. Application errors include the operation and code; their underlying causes remain available to Go callers.

`exec` uses the problem's declared toolbox image and mount permissions. File toolboxes have no network access. Service toolboxes use the recorded run's project network and require a prior `run`. Arguments execute directly; use `sh -c` explicitly for a shell command. Interactive stdin and a terminal are outside this completed-command interface. `submit` checks the file flag hash or the current recorded service flag and does not execute a solution or record progress history.

The executable forwards SIGINT and SIGTERM through the caller context and applies a fifteen-minute operation deadline. Independent cleanup may continue after cancellation or the deadline, following the application interface's resource rules.

If `run` starts a service but cannot write its result to stdout, the command attempts to stop that new run before returning the output error. Any cleanup error is retained. A pre-existing run is preserved when startup reports `already_running`.

## Adding a command

`internal/cli.Command` contains `Name`, `Summary`, and `Run(context.Context, Invocation) error`. `Invocation` contains the parsed repository path, untouched command arguments, stdout, and stderr. `cli.New(commands...)` copies registrations and rejects invalid names, duplicate names, and missing handlers. Names are lowercase words/digits separated by single hyphens, beginning with a letter.

Add a handler and its registration to `DefaultCommands`. The handler defines its own positional argument count, options, help, application calls, and output. Commands can accept no slug or multiple arguments without changing the dispatcher. The dispatcher handles global options and lookup only. Returning `flag.ErrHelp` marks a successful help request. Other handler errors pass through unchanged.

New platform functions define their own capability interfaces and result types when needed. Existing command signatures remain intact. A command invokes the application service in process; an HTTP adapter can consume the same service directly. These are compiled command registrations.

Source development can use `go run ./cmd/pwnden --repo PATH <command>` with the Go version declared in `go.mod`. End-user packages contain their own executable and bundled content identity.
