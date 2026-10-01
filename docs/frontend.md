# Frontend workspace

The browser client lives in `web/`. Go owns the local server, authentication,
problem execution and answer checking. Node and pnpm are build tools.

## Packages and ownership

| Package | Location | Responsibility |
| --- | --- | --- |
| `@pwnden/player` | `web/apps/player` | Session initialization and composition. |
| `@pwnden/ui` | `web/packages/ui` | Own button/input/panel/status/terminal/Markdown/reveal/code API, Sectile and xterm wrappers, theme tokens and bundled fonts. |
| `@pwnden/api` | `web/packages/api` | Implement domain ports with the Go HTTP API and WebSocket. |
| `@pwnden/catalog` | `web/domains/catalog` | Summary/detail/file models and read/download/guidance port; pure TypeScript. |
| `@pwnden/play` | `web/domains/play` | Run/status/submission models and player port; pure TypeScript. |
| `@pwnden/catalog-feature` | `web/features/catalog` | Player brief, bounded source preview, optional authenticated downloads, progressive hints and walkthrough. |
| `@pwnden/play-feature` | `web/features/play` | Vue observed execution state, controls and submissions. |
| `@pwnden/terminal` | `web/domains/terminal` | Connection/session/dimensions/byte-event ports; pure TypeScript. |
| `@pwnden/terminal-feature` | `web/features/terminal` | Connection controls, input/output flow and session lifetime. |

The app injects API implementations into features. Features import their domain
and the UI package. The API package imports domain ports and models. Domains
use ES libraries without browser, framework or transport dependencies. The UI
package alone imports Sectile; its public props and events belong to pwnden.
Progress packages are introduced with their actual capabilities.

Use public package exports between packages and relative imports inside one
package. `web/tools/boundaries.mjs` keeps the allowlist independently of package
manifests. It checks manifests, static/type imports, re-exports, literal dynamic
imports, import types, Vue scripts/styles, relative paths, source symlinks,
package internals and package/module cycles. Resolution uses public workspace
exports; compiler/Vite aliases and computed module loaders require an explicit
policy change. Domains also reject ambient reference directives. Negative
fixtures exercise these restrictions. This is a development dependency check,
not a sandbox for arbitrary code in build tools.

Project-owned abbreviations use whole lowercase or whole uppercase. The checker
recognizes common abbreviations in source identifiers; third-party names retain
their upstream spelling. Repository guidance records the general convention.

## Pinned build tools

| Tool | Version |
| --- | --- |
| Node LTS | `24.21.0` |
| pnpm | `12.8.1` |
| Vue / compiler / renderer | `3.5.43` |
| Vite / Vue plugin | `8.3.1` / `6.0.9` |
| TypeScript native bridge | `6.0.3-bridge.18.tsgo.7.0.2` |
| vue-tsc | `3.3.11` |
| Sectile Vue | `0.18.2` |
| Vitest | `5.0.2` |
| xterm / fit addon | `6.0.0` / `0.11.0` |
| md4x | `0.0.30` |

`typescript` is an exact npm alias to `typescript-native-bridge`, also overridden
throughout the workspace. Both `tsc` and `vue-tsc` report `TNB ACTIVE` during
checking. Direct, peer and development dependencies use exact versions, and
internal dependencies use `workspace:0.1.0`. `saveExact` is enabled. pnpm 12
records package-manager resolution and application resolution as two YAML
documents in the committed lockfile; use the pinned pnpm to consume it.

## Visual system

The player fills the browser viewport with a blue-black terminal workspace.
The design contract lives in [DESIGN.md](../web/DESIGN.md), with product context
in [PRODUCT.md](../web/PRODUCT.md). `@pwnden/ui` owns the shared palette,
monospace typography, focus treatment, spacing and component variants.
`UIPanel` provides a labelled section with an optional actions slot; `UIStatus`
renders a textual status with a decorative dot. Features supply actual state.

The wide layout places the problem list, reading/execution controls and terminal
in adjacent columns. At 76rem the terminal follows the reading column; at 48rem
the list moves above the content and the full-width document scrolls vertically.
The terminal frame constrains xterm independently of its measured row height.

JetBrains Mono and D2Coding WOFF2 files are bundled in the UI package, served
from the same origin and embedded into the Go executable. Font sources and
license terms are recorded in `web/packages/ui/src/fonts/README.md`; Vite emits
both OFL license files under `/assets/licenses/`. Go serves WOFF2 as `font/woff2`.

`UIMarkdown` accepts a `source` string. The UI package lazily imports the
md4x standalone module and initializes its embedded WASM once. It renders the
AST through Vue nodes: headings, lists, tables, fenced code, emphasis, quotes
and read-only task markers share the theme. Raw HTML displays as escaped text;
author components and attributes are excluded. HTTP/HTTPS links open with
`noopener noreferrer`, and document anchors use instance-prefixed heading IDs.
Relative and unsupported links remain text; images display their alternative
text. Problem-local image/file URL resolution is not part of this component.
Loading is announced, and parser failure preserves escaped source with retry.

The page CSP explicitly allows `script-src 'self' 'wasm-unsafe-eval'` for this
parser. JavaScript eval and inline scripts remain blocked. WASM is bundled into
a same-origin JavaScript chunk; no CDN, native addon or additional host tool is
needed at runtime.

## Verification

From the repository root, Docker runs the same checks as frontend CI:

```sh
docker build --file Dockerfile.web --target verify .
```

With the pinned development tools installed:

```sh
cd web
pnpm install --frozen-lockfile
pnpm verify
```

Verification checks boundaries, every package's types, negative dependency
fixtures, HTTP request/response mapping, native button attributes, Sectile input
label/value rendering, feature lifecycle/state recovery/file download and a
production build. Feature tests compile real client Vue templates and use a
custom renderer in Node; UI components are doubled there and tested separately.
These checks do not replace browser interaction or layout validation. Build
output is ignored under `web/dist/player`. CI builds and tests; it uploads no artifacts.

To export verified static assets locally:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=dist/web .
```

## Integration status

The Vue app composes catalog, play and terminal features for HTTP operations and a WebSocket upgrade,
including detail, declared file downloads and observed run status. Descriptions
are rendered as Markdown through the UI-owned AST renderer. Download
ports return plain `Uint8Array`; the catalog feature creates the browser Blob URL
and revokes it after initiating the download. The bearer token stays in the fetch
header. Play loads status on selection, refreshes it after mutations and provides
a manual refresh. Failed observation clears stale endpoints; an unavailable
recorded service offers stop for cleanup. Service stop also remains available
after failed observation; unknown state permits no new start or submission.
The API client uses a fixed same-origin
`/api/v1` base, keeps the fragment token in memory and uses a bearer header. It follows the server's
snake_case DTOs and tolerates extra response fields. Flags are checked on Go;
an incorrect answer is an ordinary rejected submission. Requests do not follow
redirects or persist credentials. The app removes the initial token fragment
before rendering and provides recovery text for missing sessions.

Go embeds the built HTML and assets through `internal/playerweb`. `/` serves the
Vue app and `/assets/` serves exact embedded files without directory listings,
redirects or route fallbacks. Host/origin checks and security headers apply to
these requests. The Vue entry point owns fragment initialization. Reloading
after fragment removal requires reopening the server's printed full URL.

`./pwnden setup` builds the pinned Node frontend inside Docker, copies its assets
into the Go stage and packages the official catalog. The executable needs neither
Node nor a separate asset directory. `./pwnden dev` builds Go with the `development`
tag and runs Vite in Docker against the live workspace. Vue, TypeScript and CSS
edits update through HMR, including workspace UI and feature packages. Go proxies
Vite assets and WebSockets on the player's origin; API and terminal requests keep
their existing authentication and origin checks. Vite's CSP nonce also authorizes
its generated HMR styles. Restart after Go or dependency changes. Full verification
runs separately through `pnpm verify` or the `Dockerfile.web` verification target.
Source Go builds and tests must first prepare the ignored embed directory:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
go test ./...
```

The embed directives make a missing frontend build a compile error. The
bootstrap context excludes host `node_modules` and generated frontend directories.
The development frontend is accessed through Go's printed URL so requests use
the server's own local origin. [Container terminal access](terminal.md) uses
first-frame authentication, a render acknowledgement per output frame and cleanup
on disconnect/unmount. Its UI styles use a page-specific CSP nonce through xterm's
document override. Submission history remains a subsequent stage. Reopening a full server URL and
selecting the problem recovers its current run from Go, without browser persistence.
