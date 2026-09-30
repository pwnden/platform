# Command interface

The CLI consumes the [platform application interface](application-interface.md).
A cloned platform starts with `./pwnden setup`, which builds inside Docker and
prepares its managed problems. `./pwnden dev` builds and serves the live local
problem checkout. The tracked entry script calls a common Go checkout launcher;
player commands reach the generated executable. Native packages invoke
`pwnden setup` directly. [Distribution](distribution.md) describes the lifecycle.
`serve` provides the local Go server, embedded Vue player and [HTTP API](web-api.md).

```text
pwnden [--repo PATH] <command> [command arguments]
```

Global options precede the command. Each command owns its arguments and flags; command flags precede positional arguments, following Go's standard `flag` parsing. Commands use the managed installation by default. `--repo PATH` is a developer override for an explicit checkout. The selected handler constructs the application service after parsing its arguments.

| Command | Invocation | Result |
| --- | --- | --- |
| `setup` | `pwnden setup` | Docker and Compose check, bundled problem installation, and image preparation. |
| `dev` | `pwnden dev` | In a source checkout, Docker build and local problem preparation, then the development web/API server. |
| `serve` | `pwnden serve` | Local Go web/API server for the managed installation; prints its private session URL. |
| `list` | `pwnden list` | Available problem names, categories, and titles. |
| `run` | `pwnden run <slug>` | File distribution count, or service project and actual endpoint addresses. |
| `exec` | `pwnden exec <slug> -- <command> [arguments]` | A completed command's stdout, stderr, and exit status from the declared toolbox. |
| `submit` | `pwnden submit <slug> <flag>` | `Correct.` or `Incorrect.` for the submitted flag. |
| `stop` | `pwnden stop <slug>` | Confirmation after problem resources and state are removed. |
| `validate` | `pwnden validate <slug>` | Contract compatibility and file/service count after execution policy checks. |
| `verify` | `pwnden verify <slug>` | Recovered solution flag and optional patch verification result; this is an author workflow. |

`pwnden --help` lists registered commands and global options. `pwnden <command> --help` shows that command's usage without loading a repository. Help and argument parsing diagnostics use stderr. Successful operation output uses stdout. Application failures print to stderr with the `pwnden:` prefix and exit 1; successful operations and help exit 0. Incorrect submissions exit 1 without an error banner. Completed toolbox commands preserve stdout, stderr, and their exit status from 0 through 124; Docker failures, signals, and timeouts are application failures. Application errors include the operation and code; their underlying causes remain available to Go callers.

`exec` uses the problem's declared toolbox image and mount permissions. File toolboxes have no network access. Service toolboxes use the recorded run's project network and require a prior `run`. Arguments execute directly; use `sh -c` explicitly for a shell command. Interactive stdin and a terminal are outside this completed-command interface. `submit` checks the file flag hash or the current recorded service flag and does not execute a solution or record progress history.

The executable forwards SIGINT and SIGTERM through the caller context. Ordinary commands have a fifteen-minute operation deadline. Persistent commands `serve` and `dev` run until interrupted; development preparation and individual API requests have their own fifteen-minute deadlines. Independent cleanup may continue after cancellation or a deadline, following the application interface's resource rules. See [local server](local-server.md) for authentication and shutdown behavior.

If `run` starts a service but cannot write its result to stdout, the command attempts to stop that new run before returning the output error. Any cleanup error is retained. A pre-existing run is preserved when startup reports `already_running`.

## Adding a command

`internal/cli.Command` contains `Name`, `Summary`, `Run(context.Context, Invocation) error`, and `Persistent`. `Persistent` opts into a signal-controlled lifetime instead of the ordinary command deadline. `Invocation` contains the parsed repository path, untouched command arguments, stdout, and stderr. `cli.New(commands...)` copies registrations and rejects invalid names, duplicate names, and missing handlers. Names are lowercase words/digits separated by single hyphens, beginning with a letter.

Add a handler and its registration to `DefaultCommands`. The handler defines its own positional argument count, options, help, application calls, and output. Commands can accept no slug or multiple arguments without changing the dispatcher. The dispatcher handles global options and lookup only. Returning `flag.ErrHelp` marks a successful help request. Other handler errors pass through unchanged.

New platform functions define their own capability interfaces and result types when needed. Existing command signatures remain intact. A command invokes the application service in process; an HTTP adapter can consume the same service directly. These are compiled command registrations.

Use `./pwnden dev` for source development. Explicit source CLI checks can use
`./pwnden --repo ../challenges <command>` with the latest development build.
Maintainers can also prepare the embedded frontend as described in
[frontend](frontend.md) and use `go run ./cmd/pwnden --repo PATH <command>`.
End-user packages contain their executable, embedded frontend and content identity.
