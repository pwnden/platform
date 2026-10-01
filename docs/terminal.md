# Browser terminal

Selecting a problem prepares its services and toolbox, then automatically connects
the terminal. Reentering within ten inactive minutes attaches to the same shell:
working directory, environment variables, history, temporary files and background
jobs remain available. A hidden tab, problem switch, refresh or lost connection
detaches the browser. Returning cancels expiry; leaving starts another ten minutes.
Visible attachments stay active even without keyboard input.

The header groups a compact rectangular connection switch and a power-icon
**환경 종료** button. The visible track border and button border share the exact
compact height, as do their hit areas, with softly rounded control corners.
Track padding follows this height and the thumb size. The square thumb has
surrounding space; the track's usable
width is twice the thumb width, with a one-thumb-width transition between ends.
Connected and disconnected chain icons distinguish on and off, with a progress
icon during preparation. The switch uses Sectile's native button semantics and `role="switch"`,
with the stable accessible name **터미널 연결**. It is on when connected or
preparing an attachment; preparing is explicitly labeled and exposes `aria-busy`.
Disconnected and failed states are off. Position, icon and color distinguish
these states. Accessible names, hidden status text and tooltips retain their
meaning. Controls wrap as complete units in narrow panes.

Turning the connection switch off detaches the browser and starts the existing
inactivity timer while retaining the shell and services. A manually disconnected
terminal stays detached across tab visibility changes until the switch is turned
on. Turning it off during preparation cancels the pending attachment.
**환경 종료** immediately stops and removes the shell environment and services.
Turning the switch back on prepares a new environment. Successful connection is
indicated in the header; body messages report failures and shell exit.

Up to ten problem environments are retained per server. Each environment includes
its service containers and one toolbox; the limit counts problems, not containers.
At capacity, existing work remains intact. The terminal shows the retained
environments and lets the player end one before retrying.

## Input and isolation

xterm translates keyboard and IME input into terminal bytes. Bash/Readline owns
completion, history and editing: Tab, arrows, Home/End, Delete, Ctrl+A/E,
Ctrl+U/K/W/Y/R/L, Ctrl+C/Z and `fg` keep their normal shell behavior. Ctrl+D
deletes at the cursor or exits on an empty line. Bracketed paste waits for
confirmation. Browser and operating system reserved shortcuts follow their own rules.

Clipboard shortcuts follow the browser's operating system, independently of the
server or container host:

| Browser OS | Copy selection | Paste |
| --- | --- | --- |
| Windows | Ctrl+C, Ctrl+Shift+C, Ctrl+Insert | Ctrl+V, Ctrl+Shift+V, Shift+Insert |
| macOS | Command+C | Command+V |
| Linux | Ctrl+Shift+C, Ctrl+Insert | Ctrl+Shift+V, Shift+Insert |

Windows Ctrl+C copies while text is selected and interrupts otherwise. macOS and
Linux Ctrl+C interrupts regardless of selection. Copy remains available after
shell exit. Paste uses native browser clipboard events and xterm's existing
newline normalization and bracketed-paste handling. Read permission is not
requested. The explicit Ctrl+Shift+C shortcut suppresses browser developer tools,
copies through a native copy event and falls back to clipboard write. If both
are unavailable, the terminal shows a context-menu copy instruction.

The declared solution image supplies Bash, `stty` and `C.UTF-8`. Startup enables
TTY `iutf8`, uses `LANG=LC_ALL=C.UTF-8`, `--noprofile --norc` and
`INPUTRC=/dev/null`. History remains in the retained shell with
`HISTFILE=/dev/null`. The Docker Engine owns the PTY; the Docker CLI helper
preserves context, TLS and endpoint selection.

The toolbox mounts only the declared repository-contained problem directory,
honors its writable setting, drops all capabilities and uses
`no-new-privileges`. File problems use network `none`; service problems use
their declared solve network. Clients select a slug and dimensions.

## Ownership and cleanup

The application workspace manager owns the Docker stream and shell independently
of WebSocket requests. One browser attachment receives input per problem; another
tab receives `terminal_busy`. The connection also records presence after shell
exit, keeping a visible service problem active.

`exit` and empty-line Ctrl+D report the exit code and remove the toolbox.
Services remain while their problem is visible. Reconnect creates a fresh shell.
Browser disconnect preserves the shell. `DELETE /problems/{slug}/terminal`
explicitly removes only the toolbox. **문제 환경 종료** and
`DELETE /problems/{slug}/run` remove the toolbox and services. After inactivity
expires, or on normal server shutdown, the manager performs the same full cleanup.

The installation lock excludes another server and standalone `run`, `exec`,
`verify` and `stop` operations while the server owns the installation.
Read-only commands remain available. Setup is performed before starting the server.
Ownership is persisted before creating resources in the user configuration
directory under `pwnden/workspaces`. After a process crash, startup cleans only
recorded problems from that repository using repository/problem Docker labels and
their Compose projects. Failed cleanup retains ownership and blocks startup;
independent environments are still attempted.

## WebSocket protocol v2

Connect to `GET /api/v1/problems/{slug}/terminal` with exact same-origin
`Origin` and subprotocol `pwnden.terminal.v2`. Authentication arrives in the
first text frame within five seconds:

```json
{"type":"authenticate","token":"PROCESS_SESSION_TOKEN","cols":80,"rows":24}
```

Credentials stay out of URLs and diagnostics. Query parameters, encoded paths,
bodies, duplicate keys, unknown fields and invalid messages are rejected.
Columns are integers 2–500 and rows 1–200; input frames are at most 16 KiB.
Environment preparation has a fifteen-minute limit. Active attachments have
heartbeat checks and no fixed lifetime deadline.

| Direction | Frame | Meaning |
| --- | --- | --- |
| Server → client | Text `{"type":"ready","reused":true}` | Ready; `reused` identifies an existing shell. |
| Client → server | Binary | Raw input bytes. |
| Server → client | Binary | Snapshot followed by live output, at most 16 KiB per frame. |
| Client → server | Text `{"type":"ack"}` | Previous output rendered. |
| Client → server | Text `{"type":"resize","cols":120,"rows":40}` | Resize the daemon PTY. |
| Server → client | Text `{"type":"exit","code":7}` | Shell exited; connection still records visible presence. |
| Server → client | Text `{"type":"error","code":"..."}` | Failure followed by disconnect. |

## Output restoration

The server continuously reads detached output into a VT screen, using the pinned
Charmbracelet emulator. Its primary scrollback holds the latest 500 lines; older
history is discarded. Reattachment restores primary/alternate screen content,
cursor position and terminal modes, then streams live output in order. Pending
UTF-8 and ANSI sequence prefixes carry across attachment boundaries.
Queries while detached are answered by the emulator; attached xterm supplies its
own responses.

Each attachment has sixteen pending frames of at most 16 KiB. A slow attachment
is closed without terminating its shell. Snapshot chunks use the same render
acknowledgement path as live output. Acknowledgements have fifteen seconds,
writes five seconds; heartbeat detects abandoned connections. Rendering changes
are confined to the browser terminal.

Input uses a separate sixteen-frame queue and a server-owned writer. A blocked
program cannot prevent browser detach. A full queue reports input backpressure;
its capacity remains bounded independently of the browser's network buffer.

The server's style nonce is assigned to xterm's own generated style elements;
global DOM methods are unchanged. JavaScript eval and arbitrary inline styles
remain blocked. WebSocket connections target the printed server origin only.

## Verification

```sh
go test ./...
go vet ./...
go test -race ./internal/application -run '^TestWorkspace'
go test -race ./internal/httpapi
go build -o dist/smoke-terminal ./tools/smoke_terminal
python3 -B tools/smoke_http.py --package dist/pwnden-linux-amd64.tar.gz --terminal-driver dist/smoke-terminal
```

Unit tests cover timer cancellation/reset, active retention, ten-environment cap,
exit, cleanup errors, installation locks, crash recovery and split UTF-8/ANSI
screen restoration. Frontend tests cover automatic attachment, stale callbacks,
visibility, capacity recovery, keyboard byte transport and resize.
Clipboard tests cover browser OS detection, native copy/paste delegation,
selection-dependent Windows interruption, Linux/macOS control keys, IME key
pass-through, copy-event listener cleanup and clipboard failure handling.

Actual Docker checks cover shell state retention, detached output, Readline,
mount/network/privilege policy, automatic service startup, terminal-only stop,
full stop, normal shutdown and crash recovery. Browser Plugin screen/IME checks
and Windows/macOS host execution are separate validation scopes.
