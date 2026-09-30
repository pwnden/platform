# Command interface

The CLI consumes the [platform application interface](application-interface.md). The executable selects commands and formats results. Run the commands from the platform checkout using `go run ./cmd/pwnden`, or substitute a built `pwnden` executable.

```text
pwnden [--repo PATH] <command> [command arguments]
```

Global options precede the command. Each command owns its arguments and flags; command flags precede positional arguments, following Go's standard `flag` parsing. The current four commands require `--repo PATH` and exactly one problem slug. `--repo` is parsed globally and the application service is constructed by the selected problem handler.

| Command | Invocation | Result |
| --- | --- | --- |
| `validate` | `pwnden --repo PATH validate <slug>` | Contract compatibility and file/service count after execution policy checks. |
| `run` | `pwnden --repo PATH run <slug>` | File distribution count, or service project and actual endpoint addresses. |
| `verify` | `pwnden --repo PATH verify <slug>` | Recovered solution flag and optional patch verification result; this is an author workflow. |
| `stop` | `pwnden --repo PATH stop <slug>` | Confirmation after problem resources and state are removed. |

`pwnden --help` lists registered commands and global options. `pwnden <command> --help` shows that command's usage without loading a repository. Help and argument parsing diagnostics use stderr. Successful operation output uses stdout. The executable prints errors to stderr with the `pwnden:` prefix and exits 1; successful operations and help exit 0. Application errors include the operation and code; their underlying causes remain available to Go callers.

The executable forwards SIGINT and SIGTERM through the caller context and applies a fifteen-minute operation deadline. Independent cleanup may continue after cancellation or the deadline, following the application interface's resource rules.

If `run` starts a service but cannot write its result to stdout, the command attempts to stop that new run before returning the output error. Any cleanup error is retained. A pre-existing run is preserved when startup reports `already_running`.

## Adding a command

`internal/cli.Command` contains `Name`, `Summary`, and `Run(context.Context, Invocation) error`. `Invocation` contains the parsed repository path, untouched command arguments, stdout, and stderr. `cli.New(commands...)` copies registrations and rejects invalid names, duplicate names, and missing handlers. Names are lowercase words/digits separated by single hyphens, beginning with a letter.

Add a handler and its registration to `DefaultCommands`. The handler defines its own positional argument count, options, help, application calls, and output. Commands can accept no slug or multiple arguments without changing the dispatcher. The dispatcher handles global options and lookup only. Returning `flag.ErrHelp` marks a successful help request. Other handler errors pass through unchanged.

New platform functions define their own capability interfaces and result types when needed. Existing command signatures remain intact. A command invokes the application service in process; an HTTP adapter can consume the same service directly. These are compiled command registrations.
