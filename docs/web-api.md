# Local player HTTP contract

The platform owns the web client and HTTP adapter in the same repository. The HTTP contract is [OpenAPI 3.1.2](https://spec.openapis.org/oas/v3.1.2.html), recorded in [openapi.json](openapi.json). JSON DTOs, handlers, server lifecycle, and public error mapping live in `internal/httpapi`. `pwnden serve` opens the managed installation, serves the embedded Vue player and provides these operations. [Terminal protocol](terminal.md) defines WebSocket frames and session ownership. See [local server](local-server.md) for startup and verification.

The API version is independent of the challenges-owned problem contract. Version 1 uses `/api/v1`, snake_case JSON keys and `application/json`. API changes are maintained with the platform. Compatible additions stay in version 1; incompatible request/response semantics require another major API path. Clients tolerate additional response fields and use error codes rather than matching messages.

## Operations

Completed problems add `solved_at` (UTC RFC 3339) to summaries and details. Their
authenticated detail also includes `answer`, the player's previously accepted
input, for readonly restoration. Unsolved problems omit both fields. Listing
problems never includes accepted inputs. A correct submission returns success
after its durable write commits; storage failures return `503 storage_failed`.
See [durable state](storage.md) for ownership and catalog identity.

| Method and path | Application capability | Success |
| --- | --- | --- |
| `GET /api/v1/problems` | `Catalog.List` | `200`, problem summaries sorted by slug. |
| `GET /api/v1/workspaces` | `Workspaces.List` | `200`, retained problem environments, limit and idle duration. |
| `GET /api/v1/problems/{slug}/workspace` | `Workspaces.View` | `200`, authenticated NDJSON ready/heartbeat stream that prepares and retains the visible environment without a terminal. |
| `GET /api/v1/problems/{slug}` | `Details.Detail` | `200`, summary, description text and declared file metadata. |
| `GET /api/v1/problems/{slug}/guidance/{id}` | `Guidance.Guidance` | `200`, declared hint or walkthrough Markdown after explicit activation. |
| `GET /api/v1/problems/{slug}/files/{id}` | `Files.Download` | `200`, binary attachment for a current declared file ID. |
| `GET /api/v1/problems/{slug}/status` | `Observer.Status` | `200`, observed run status and owned local HTTP ingress endpoints. |
| `POST /api/v1/problems/{slug}/browser` | HTTP adapter browsing session | `200`, `slug`, endpoint `name`, wrapper `url` and original `target` origin. |
| `POST /api/v1/problems/{slug}/run` | `Runner.Run` | `200`, problem kind, file count and owned local HTTP ingress endpoints. |
| `DELETE /api/v1/problems/{slug}/run` | `Runner.Stop` | `200`, stopped slug after cleanup. |
| `POST /api/v1/problems/{slug}/submissions` | `Submit` | `200`, slug and `accepted` boolean. |
| `GET /api/v1/problems/{slug}/terminal` | `Workspaces.Attach` | `101`, same-origin WebSocket v2 with first-frame authentication; prepares stopped services automatically. |
| `DELETE /api/v1/problems/{slug}/terminal` | `Workspaces.EndTerminal` | `200`, toolbox removed while problem services remain. |

Problem summaries contain `slug`, `title`, `category`, and `kind`, plus `difficulty` when declared. Difficulty is an integer from 1 (Intro) to 5 (Expert), shared by summaries and detail. Installed catalogs without a declared difficulty omit it; the consumer preserves their unrated state. Missing display titles use the slug. Kind is `file` or `service`. Arrays are JSON arrays, including empty arrays; they never serialize as `null`.

Summaries include optional `search_text` containing the public problem brief.
The player searches this text alongside titles, identifiers and categories, so
CLI names such as `nmap` resolve without opening each problem. Empty briefs omit
the field. Hints, walkthroughs, solution files and runtime state are excluded.
Text uses the same UTF-8, 1 MiB and repository containment checks as descriptions.

Details add `description`, `files`, `tools`, `hint_count` and `walkthrough`. Contract 4 declares the ordered `tools` array through `[player].tools`: `web`, `files` or `terminal`. The player displays these tools and opens the first by default. Earlier catalogs derive their tools from declared resources. Contract 2 and later read the declared player brief; hint and walkthrough bodies are absent from this response. Contract 1 installations retain optional README text and have no guidance. Markdown is UTF-8, at most 1 MiB. Guidance IDs are `hint-1` through `hint-10` and `walkthrough`; only entries declared by that problem resolve. They use the same authentication, request-body and repository containment rules as other reads. The response contains `id` and Markdown `content`. It intentionally includes answer content on a walkthrough request.

Each file has `id`, `name` and byte `size`. Only declared distribution files and regular descendants of declared directories are listed. Repository-contained links and shared files are supported; outside paths, cycles and special files fail. IDs are stable path identifiers, not flag or content hashes. Download rechecks the current declared distribution, requires bearer authentication, streams `application/octet-stream`, and sets attachment `Content-Disposition` to the file basename. No host path or arbitrary relative path is accepted. The UI uses this same read for an escaped UTF-8 source preview capped at 1 MiB; binary or large material remains available through download. Downloads remain optional; tokens never appear in file links.

The workspace read authenticates with the bearer header and prepares the visible
environment without creating an interactive shell. It returns
`application/x-ndjson`: a `ready` message with public `status`, then a `heartbeat`
every twenty seconds. Preparation has the normal operation deadline; the active
stream lasts until disconnection or server shutdown. An invalidated workspace
emits `error` with `code: not_running`. Each connected view retains the environment;
the last view and terminal attachment leaving starts its ten-minute idle timer.
Returning cancels that timer. The shared environment limit is ten.

Status contains `slug`, `kind`, `state` and `endpoints`. A file problem is `ready`; an unrecorded service is `stopped`. A recorded service is `running` when every declared service has running containers with healthy or absent health checks; otherwise it is `unavailable` and requires explicit stop before another start. Endpoints appear only while running. Status survives server and page recreation through saved state and actual Docker observation. Failed Docker reads return an error; they do not erase state or clean up a run. The response contains no stored flag or project identifier.

Start and stop accept no body. File starts create no service resources and return an empty endpoint array. A service start preserves an already recorded run and returns a conflict. Endpoints contain `name` and `url`, preserving the problem's `http://` or `tcp://` scheme. HTTP URLs are owned loopback ingress origins; TCP URLs identify declared service addresses for tools inside the problem terminal. HTTP container addresses, run identities and Compose project identifiers stay inside the application layer. Stop is repeatable and reports success only after resource and state cleanup.

Submission accepts exactly one JSON object with a nonempty string `flag`, up to 4096 body bytes. Unknown request fields, trailing JSON values, malformed JSON, query parameters, and unexpected bodies are rejected. The media type must be `application/json` with an optional charset. Leading and trailing whitespace is trimmed by the application. An incorrect answer is `200 {"slug":"...","accepted":false}`. Service submissions require a recorded run. Submission history is a later feature.

The API exposes the player operations above. Author solution verification remains a CLI operation. Submission history is a subsequent capability.

## Common problem browser

Browser preparation accepts exactly one nonempty endpoint `name`, with the same
JSON and 4096-byte body rules as submission. The destination comes from the
running problem's declared HTTP services. Fixed-destination Docker streams reach
the service inside its isolated network. Each endpoint gets a
separate local proxy origin and a wrapper with native browser history controls;
the platform token stays on the API origin. Repeated opens reuse that wrapper.
Every proxied request rechecks current run status under the problem lock; stopped
or changed endpoints return 410. Workspace expiry closes its listeners within
one second of removal, and server shutdown closes proxies before environment
cleanup. The proxy preserves request methods, bodies, paths, query strings,
cookies and response security headers, and maps redirects back to its own origin.
Problem authors declare HTTP endpoints through the existing challenges contract.
Shared browsing code is maintained in the platform; see [frontend](frontend.md).

## Error responses

Every unsuccessful response has this shape:

```json
{"error":{"code":"not_running","message":"Start the problem first.","operation":"submit","slug":"note-vault"}}
```

`code` and `message` are required. `operation` and `slug` are optional. Application causes and process output stay in local diagnostics; they are not serialized. Public responses exclude challenge flags, file hashes, author solutions and host paths. Unknown errors produce a generic `internal_error`.

| HTTP status | Codes |
| --- | --- |
| `400` | `invalid_argument`, transport `invalid_request` |
| `401` | transport `unauthorized` |
| `403` | transport `forbidden` |
| `404` | `not_found` |
| `405` | transport `method_not_allowed` |
| `409` | `already_running`, `not_running`, `workspace_full`, `resource_limit`, `terminal_busy`, `workspace_busy` |
| `413` | transport `payload_too_large` |
| `415` | transport `unsupported_media_type` |
| `422` | `incompatible_contract`, `validation_failed` |
| `500` | `execution_failed`, `cleanup_failed`, `internal_error` |
| `503` | `setup_required`, `setup_failed`, `canceled` |
| `504` | `deadline_exceeded` |

Application error mapping follows typed errors, including cleanup precedence over cancellation. Transport failures use the same JSON envelope. A disconnected request may receive no response; its operation still follows application cancellation and independent cleanup rules.

## Local server ownership and access

The server binds to `127.0.0.1`, chooses an available port, and prints its actual URL with a random process-scoped token in the URL fragment. Fragments are absent from HTTP requests. The Vue entry point removes the fragment from the current history entry before rendering and injects it into the same-origin client in memory; API calls use the bearer header. Host, Origin, and browser fetch-site checks confine requests to the printed local origin; the server enables no cross-origin access. API authentication also applies to reads. Embedded HTML and static assets are available without a bearer header under the same origin checks. Responses use `Cache-Control: no-store`, and request bodies/tokens are excluded from diagnostics.

The HTTP adapter opens the managed installation itself. Requests select problem slugs; they cannot select a repository, host mount, image or Docker options. The backend capability set is `httpapi.Backend`, implemented by the existing application service. Handlers call it directly.

The server coordinates mutating calls and status reads in a single process. Calls for the same problem are serialized, including waiting cancellation; independent problems can proceed independently. Detail and file reads do not acquire the mutation lock. Queued requests are checked for cancellation before backend execution. Setup and CLI mutations are invoked outside an active server session. Cross-process coordination is a separate implementation feature.

Each ordinary API request and workspace preparation has a fifteen-minute operation deadline; the server itself runs until interrupted. A successful start whose response write or flush fails stops the run it just created, preserving cleanup errors. A canceled start that finishes successfully at the cancellation boundary also stops its new run before returning an error. A successful TCP flush cannot confirm that the browser processed the response. A pre-existing run remains owned by its existing session. Server shutdown stops accepting requests, cancels active work and waits for required cleanup, including handlers whose clients already disconnected. Visible problem environments remain available until explicit stop, idle expiry or server shutdown; startup failures and interrupted operations retain the application's existing cleanup ownership.
