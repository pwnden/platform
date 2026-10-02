# Browser terminal

Selecting a problem prepares its environment through the independent workspace
view stream. The challenges-owned tool declaration determines whether Terminal
is displayed. Opening its tab for the first time starts the shell automatically.
Reentering within ten inactive minutes attaches to the same shell: working
directory, environment variables, history, temporary files and background jobs
remain available. A hidden page or problem switch releases problem presence and
terminal attachment; returning cancels expiry. Tool switches within the selected
problem keep visited tools mounted. Web-only problems create no terminal.

The common tool tab strip contains one **터미널 새로고침** icon at its right
edge. Refresh closes only the browser attachment and reconnects to the existing
shell. Files, directory, history and background jobs remain available; the server
restores output. After shell exit, refresh prepares a new shell. During preparation,
the icon is busy and disabled; a viewport overlay announces progress while
preserving the terminal geometry. Failures and shell exit appear as overlays inside the screen.
Cleanup follows the existing ten-minute inactivity policy and server shutdown.
An unavailable service is recovered after leaving the problem and allowing its
inactivity cleanup to finish.

Up to ten problem environments are retained per server. An environment includes
its service containers and, when used, one toolbox; the limit counts problems, not containers.
At capacity, existing work remains intact. The fixed submission bar offers a retained-environment recovery overlay even
when the problem has no Terminal tool.

## Input and isolation

The prompt shows the current shell directory, for example `/challenge #`.
The path uses cyan from the shared terminal palette. Bash updates it after `cd` and
uses `#` for root or `$` for other users. Color sequences are enclosed in
Readline nonprinting delimiters so cursor movement and line wrapping use the
visible prompt width. The platform supplies `PS1` inside the toolbox; players
can change it with normal Bash assignments. Existing retained shells keep their
current prompt until replaced.

xterm translates keyboard and IME input into terminal bytes. Bash/Readline owns
completion, history and editing: Tab, arrows, Home/End, Delete, Ctrl+A/E,
Ctrl+U/K/W/Y/R/L, Ctrl+C/Z and `fg` keep their normal shell behavior. Ctrl+D
deletes at the cursor or exits on an empty line. Bracketed paste waits for
confirmation. Browser and operating system reserved shortcuts follow their own rules.

Filename completion ignores case: `cat readme` followed by Tab completes to
`cat README.md` when that filename is the unique match. The filesystem still
uses the actual filename. Tab invokes Readline's `menu-complete` to cycle forward
through multiple matches; Shift+Tab invokes `menu-complete-backward` to cycle
backward. Native Readline cycling includes the original input after the last
candidate, then begins again. A missing match leaves the input intact.

Other shell bindings retain Bash's native behavior. Readline supplies editing
and history, Bash supplies builtins, expansions, pipelines, redirection,
functions and programmable completion, and the Docker PTY supplies job control.
Available external programs come from the declared solution image.

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
the platform's container-local `INPUTRC=/tmp/pwnden.inputrc`, containing
case-insensitive completion and these native function bindings:

```text
set completion-ignore-case on
"\C-i": menu-complete
"\e[Z": menu-complete-backward
```

Startup passes this configuration as a positional argument to Bash and writes
it with private permissions in the toolbox, preserving the repository mount
policy. Only configuration creation changes the umask; the interactive shell
retains its ordinary file permissions.
History remains in the retained shell with
`HISTFILE=/dev/null`. The Docker Engine owns the PTY; the Docker CLI helper
preserves context, TLS and endpoint selection.

For a disposable Docker completion regression check, run
`PWNDEN_TEST_CHALLENGES=/absolute/path/to/challenges go test ./internal/runtime -run '^TestTerminalFilenameCompletionDocker$' -count=1 -v`.
It creates its own toolbox, checks relative and absolute case-insensitive
completion, directory prompt and directory updates, directory completion, forward and reverse candidate cycling,
wraparound, missing matches, UTF-8 deletion and command history, then
removes that toolbox. Existing player environments remain available.

The toolbox mounts only the declared repository-contained problem directory,
honors its writable setting, drops all capabilities and uses
`no-new-privileges`. File problems use network `none`; service problems use
their declared solve network with internal, isolated IPv4 and IPv6 gateway modes.
Internet, host services and other problem networks are blocked. Clients select a
slug and dimensions. See [network isolation](network-isolation.md).

## Ownership and cleanup

The application workspace manager owns the Docker stream and shell independently
of WebSocket requests. One browser attachment receives input per problem; another
tab receives `terminal_busy`. The independent workspace stream records problem presence without a shell;
terminal attachments also retain their environment for compatibility.

`exit` and empty-line Ctrl+D report the exit code and remove the toolbox.
Services remain while their problem is visible. Reconnect creates a fresh shell.
Browser disconnect preserves the shell. `DELETE /problems/{slug}/terminal`
explicitly removes only the toolbox. `DELETE /problems/{slug}/run` removes
the toolbox and services. After inactivity
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
