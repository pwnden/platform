# Browser terminal

Select a problem and choose **터미널 연결** in the local player. File problems
need no service. A stopped service offers **문제 실행 후 터미널 연결**: the play
feature starts the problem and refreshes its observed state before the terminal
connects. An already-running service connects directly. Failed or canceled
preparation keeps the shell closed. Connection errors provide recovery guidance,
and input receives focus only after the shell is ready. Both desktop column
boundaries support resizing; the existing connection receives new dimensions.
The terminal opens interactive Bash with Readline inside the declared solution
image at `/challenge`. The image must provide `/bin/bash`, `stty` and the
`C.UTF-8` locale. Startup enables the TTY's `iutf8` mode and sets `LANG` and
`LC_ALL` to `C.UTF-8` so line editing and terminal erase use UTF-8 characters.
Bash starts with `--noprofile --norc` and `INPUTRC=/dev/null` for consistent
default Readline bindings. The Docker Engine allocates the TTY. The official Docker CLI connection
helper preserves the same context, TLS and endpoint selection as Compose.

## Input and editing

xterm translates keyboard and IME input into terminal bytes; the transport
preserves those bytes. Bash/Readline owns completion, editing and command
history. Tabs complete commands and paths at the shell prompt. Arrow keys browse
history and move within a line. Home/End, Delete, Ctrl+A/E, Ctrl+U/K/W/Y,
Ctrl+R and Ctrl+L use the standard Readline bindings. Ctrl+C interrupts the
foreground process; Ctrl+Z and `fg` use Bash job control. Ctrl+D deletes at the
cursor or closes the shell on an empty line. UTF-8 Backspace erases the input
character and stops at the beginning of the editable line.

Bracketed paste keeps pasted input literal until the player confirms it. Programs
running in the terminal own their own key bindings, including literal Tab where
appropriate. Browser and operating system reserved shortcuts remain subject to
their normal restrictions. Clipboard and IME handling use xterm's native paths.

Command history lives within the connection; `HISTFILE=/dev/null` keeps it out of
the problem mount. Reconnecting starts a fresh shell and history.

The toolbox mounts only the allowed problem directory with its declared
read-only/writable setting, drops all capabilities and uses `no-new-privileges`.
Writable mounts use the host UID/GID where available. File problems use network
`none`; service problems use the current project's declared solve network.
Clients choose a slug and dimensions; Docker options stay inside the runtime.

## Session ownership

`application.Terminals.OpenTerminal(ctx, slug, cols, rows)` returns a
`TerminalSession`: `io.ReadWriteCloser`, `Resize(ctx, cols, rows)` and
`Wait(ctx) (exitCode, error)`. Output is merged raw TTY stdout/stderr. The owner
closes the session. Repeatable close removes its container and anonymous volumes,
with independent cleanup limited to thirty seconds. Cancellation triggers close.
An interrupted create response attempts cleanup using the random owned name.
The session uses the caller's deadline rather than the automatic solution timeout.

HTTP permits one terminal per problem. Creation shares the mutation lock;
streaming releases it, allowing submissions and status reads. HTTP problem stop
cancels and waits for its terminal before stopping services. Exit, disconnect
and server shutdown close the session. Cleanup failure remains a server failure;
service stop reports it rather than claiming success. Setup and CLI mutations
are performed outside an active server. Reconnection opens a new shell.

## WebSocket protocol v1

Connect to `GET /api/v1/problems/{slug}/terminal` with exact same-origin `Origin`
and subprotocol `pwnden.terminal.v1`. Query parameters, encoded paths and request
bodies are rejected. Native browser WebSocket does not supply a bearer header;
the first text frame within five seconds authenticates the session:

```json
{"type":"authenticate","token":"PROCESS_SESSION_TOKEN","cols":80,"rows":24}
```

Authentication precedes container creation. The initial page fragment and
same-tab credential lifetime are described in [local-server.md](local-server.md).
Tokens stay outside WebSocket URLs, diagnostics and terminal output. Columns are integers 2–500 and rows
1–200. Unknown fields, duplicate keys and invalid messages fail. Incoming messages
are limited to 16 KiB. The existing fifteen-minute HTTP deadline limits sessions.

| Direction | Frame | Meaning |
| --- | --- | --- |
| Server → client | Text `{"type":"ready"}` | Shell accepts input. |
| Client → server | Binary | Raw input bytes, including Ctrl+C (byte 3). |
| Server → client | Binary | Raw output, at most 16 KiB per frame. |
| Client → server | Text `{"type":"ack"}` | Previous output frame has rendered. |
| Client → server | Text `{"type":"resize","cols":120,"rows":40}` | Resize the daemon TTY. |
| Server → client | Text `{"type":"exit","code":7}` | Shell exit code. |
| Server → client | Text `{"type":"error","code":"..."}` | Public failure code, then disconnect. |

The server waits fifteen seconds for each output acknowledgement and limits
writes to five seconds. Missing acknowledgements end with `output_backpressure`.
Client buffered input is limited to 64 KiB (`input_backpressure`); paste is split
into 16 KiB frames. Other codes include `unauthorized`, `terminal_busy`,
`invalid_argument` and application error codes. Closing the socket terminates
the owned session.

## Frontend and styles

Pure TypeScript `domains/terminal` owns the ports and byte events; `packages/api`
implements WebSocket. The app injects it into `features/terminal`, which owns
connection controls and unmount cleanup. `packages/ui` alone wraps xterm and fit,
owns their CSS and exports its own props, events and handle.

Each HTML response issues a fresh style nonce in CSP and a meta element. The UI
uses xterm's documented `documentOverride` with a document proxy to assign that
nonce only to its generated styles. Global DOM methods remain unchanged.
The viewport's styles receive the nonce before insertion into xterm's own
containers, including styles created through the native document in xterm 6.
Scripts retain `default-src 'self'`; arbitrary inline styles remain blocked.
WebSocket connections are explicitly permitted only to the printed server host.

## Verification

Go tests cover authentication before creation, origin/protocol, bytes, resize,
duplicate sessions, exit codes, invalid messages, disconnect, problem stop and
server cancellation. Frontend tests cover ports, connection cancellation,
render acknowledgements, input limits and feature lifetime.

The maintainer integration driver checks real Docker while the packaged player's
PATH contains only Docker:

```sh
go build -o dist/smoke-terminal ./tools/smoke_terminal
python3 -B tools/smoke_http.py --package dist/pwnden-linux-amd64.tar.gz --terminal-driver dist/smoke-terminal
```

It inspects mount/privilege/network policies, runs the file solver, accesses the
service internally, checks UTF-8 erase, path completion, history, cursor editing,
Readline shortcuts, bracketed paste, size/Ctrl+C/exit codes, and verifies cleanup on exit,
disconnect and problem stop. Browser interaction and Windows/macOS execution
are separate validation scopes.
