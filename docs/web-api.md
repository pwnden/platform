# Local player HTTP contract

The platform owns the web client and HTTP adapter in the same repository. The initial API contract is [OpenAPI 3.1.2](https://spec.openapis.org/oas/v3.1.2.html), recorded in [openapi.json](openapi.json). JSON DTOs, handlers, server lifecycle, and public error mapping live in `internal/httpapi`. `pwnden serve` opens the managed installation, serves the embedded Vue player and provides these operations. Interactive terminal access is a subsequent implementation stage. See [local server](local-server.md) for startup and verification.

The API version is independent of the challenges-owned problem contract. Version 1 uses `/api/v1`, snake_case JSON keys and `application/json`. API changes are maintained with the platform. Compatible additions stay in version 1; incompatible request/response semantics require another major API path. Clients tolerate additional response fields and use error codes rather than matching messages.

## Operations

| Method and path | Application capability | Success |
| --- | --- | --- |
| `GET /api/v1/problems` | `Catalog.List` | `200`, problem summaries sorted by slug. |
| `POST /api/v1/problems/{slug}/run` | `Runner.Run` | `200`, problem kind, file count and published endpoints. |
| `DELETE /api/v1/problems/{slug}/run` | `Runner.Stop` | `200`, stopped slug after cleanup. |
| `POST /api/v1/problems/{slug}/submissions` | `Submit` | `200`, slug and `accepted` boolean. |

Problem summaries contain `slug`, `title`, `category`, and `kind`. Missing display titles use the slug. Kind is `file` or `service`. Arrays are JSON arrays, including empty arrays; they never serialize as `null`.

Start and stop accept no body. File starts create no service resources and return an empty endpoint array. A service start preserves an already recorded run and returns a conflict. Endpoints contain `name` and `url` for published entry points, preserving the problem's `http://` or `tcp://` scheme. HTTP URLs are browser links; TCP URLs identify addresses for terminal tools. Private network endpoints and Compose project identifiers stay inside the application layer. Stop is repeatable and reports success only after resource and state cleanup.

Submission accepts exactly one JSON object with a nonempty string `flag`, up to 4096 body bytes. Unknown request fields, trailing JSON values, malformed JSON, query parameters, and unexpected bodies are rejected. The media type must be `application/json` with an optional charset. Leading and trailing whitespace is trimmed by the application. An incorrect answer is `200 {"slug":"...","accepted":false}`. Service submissions require a recorded run. Submission history is a later feature.

The initial API exposes the player operations above. Author solution verification remains a CLI operation. Interactive terminal streaming is added after its application session contract is implemented. Description/file access and persisted player status are subsequent capabilities with their own response contracts.

## Errors

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
| `409` | `already_running`, `not_running` |
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

The server coordinates mutating calls in a single process. Calls for the same problem are serialized, including waiting cancellation; independent problems can proceed independently. Queued requests are checked for cancellation before backend execution. Setup and CLI mutations are invoked outside an active server session. Cross-process coordination is a separate implementation feature.

Each API request has a fifteen-minute operation deadline; the server itself runs until interrupted. A successful start whose response write or flush fails stops the run it just created, preserving cleanup errors. A canceled start that finishes successfully at the cancellation boundary also stops its new run before returning an error. A successful TCP flush cannot confirm that the browser processed the response. A pre-existing run remains owned by its existing session. Server shutdown stops accepting requests, cancels active work and waits for required cleanup, including handlers whose clients already disconnected. Completed service runs remain available until explicit stop; startup failures and interrupted operations retain the application's existing cleanup ownership.
