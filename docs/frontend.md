# Frontend workspace

The browser client lives in `web/`. Go owns the local server, authentication,
problem execution and answer checking. Node and pnpm are build tools.

## Packages and ownership

| Package | Location | Responsibility |
| --- | --- | --- |
| `@pwnden/player` | `web/apps/player` | Session initialization and composition. |
| `@pwnden/ui` | `web/packages/ui` | Own controls, panels, tabs, terminal, Markdown, code and iframe API; Sectile and xterm wrappers, theme tokens and bundled fonts. |
| `@pwnden/api` | `web/packages/api` | Implement domain ports with the Go HTTP API and WebSocket. |
| `@pwnden/catalog` | `web/domains/catalog` | Summary/detail/file models and read/download/guidance port; pure TypeScript. |
| `@pwnden/play` | `web/domains/play` | Run/status/submission models and player port; pure TypeScript. |
| `@pwnden/catalog-feature` | `web/features/catalog` | Player brief, bounded source preview, optional authenticated downloads, progressive hints and walkthrough. |
| `@pwnden/play-feature` | `web/features/play` | Observed execution state, submissions and local problem web documents. |
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
body and monospace font stacks, focus treatment, spacing and component variants.
`UIPanel` provides a labelled section with an optional actions slot; `UIStatus`
renders a textual status. `UISwitch` wraps Sectile's native switch semantics with
a softly rounded rectangular track and square thumb. The track's usable width
is twice the thumb width, and the thumb travels by its own width. It exposes a
controlled boolean value, stable accessible label, optional compact presentation
and a thumb slot receiving checked and busy state. Pending state remains operable;
callers use `disabled` when changes are unavailable. The player tool strip uses
`UIIconButton` for the selected tool's actions, with accessible names and tooltips.
Features supply actual state and behavior through their public component handles.

`--ui-control-size-compact` fixes the visible border-box height of compact
buttons and switch tracks. Switch padding derives from this shared height and
the square thumb size, retaining two thumb widths of usable track space.

All component padding uses one inset for top, right, bottom and left. Default
and compact controls derive their inset from the height, content line height
and border. Panel headings and bodies share `--ui-panel-inset`; responsive
changes replace the whole inset. Sibling spacing uses `gap`. Embedded terminal
output receives its inset from the panel body.

`UISelect` presents the selected label and chevron in a grid with equal outer
insets. A native select covers the full field, retaining its label, option
selection, keyboard navigation and mouse behavior. Focus and disabled styles
apply to the visible field. `pnpm check:spacing`, included in `pnpm verify`,
checks owned Vue style blocks and CSS with the existing Vue compiler's CSS
parser, rejecting unequal padding and directional overrides.

Corner tokens distinguish controls (`--ui-radius-control`), content surfaces
(`--ui-radius-surface`), inline elements (`--ui-radius-inline`) and adjoining
workspace panels (`--ui-radius-flush`). `--ui-radius` remains a control alias.
For nested contours, a container sets `--ui-radius-outer` and
`--ui-radius-inset` (border plus padding). `.ui-radius-outer` uses the outer
value; `.ui-radius-inner` computes `max(0px, outer - inset)` on the child,
so locally overridden values remain effective. The switch uses this relation
for its track and thumb. The framed web document derives its inner corners
from the containing surface and border.

The selected problem header uses the same compact title scale and control-based
height as its neighboring workspace headers. Its title stays on one line with a
full-title tooltip; the category appears in a right-aligned `UIBadge`, with a
screen-reader category prefix. The body scrolls independently below it.

The execution panel groups status and an icon-only refresh action in the header.
Refresh queries the current server state. The panel shows TCP addresses and the
flag form; the terminal header owns connection and environment termination.
Connecting prepares the environment automatically. HTTP entry points appear
in the right-side web tool.
The flag field and confirmation form are a separate group below a divider.
File problems proceed directly to submission. Both form controls share the
default height; narrow containers stack them at full width.
Refresh retains the existing body, flag input and submission result
while the header icon rotates inside its fixed box. The flag field remains
editable; submitting and duplicate refresh requests wait for the observation.
Reduced-motion preferences keep the refresh icon static. Failed observation
clears stale endpoints and exposes recovery; a confirmed state change updates
the displayed content.

Shared interaction styles live in `packages/ui/src/theme.css`. Buttons, fields,
links, disclosures, code scroll regions and split handles share focus tokens and
140ms state transitions. Keyboard focus stays inside bounded controls; disclosure
focus outlines the label and chevron. Hover is gated by the device's hover
capability. Pressed, selected, expanded, disabled and invalid states have their
own semantic styles. Reduced-motion and forced-color preferences are supported.

`ProblemFiles` displays its selected source automatically on entering Files.
A single file has a path label; multiple files have a named selector. Source
fills the remaining pane height. File sizes are metadata for the preview limit.
The catalog feature owns selection, fetching, limits, retry, download and cache.
Its public handle exposes the selected filename, download progress and download
action for app composition. The common tab strip places the download icon at its
right edge. Progress occupies the same icon box; repeated requests stay disabled.
Reduced-motion preferences keep the indicator static.

The wide layout places the problem list, reading/execution controls and tool pane
in adjacent columns. At 76rem the tool pane follows the reading column; at 48rem
the list moves above the content and the full-width document scrolls vertically.
The terminal frame constrains xterm independently of its measured row height.

The tool pane defaults to Terminal. Files appears for declared distribution
materials; Web appears once a published local HTTP entry point is observed and
remains available for recovery after environment termination. `UITabs` owns
Sectile keyboard navigation, linked tab/panel semantics, selected styling and
inactive-panel exclusion. Panels stay mounted. Switching tools preserves the
terminal attachment, source previews and each opened web document. It does not
start the environment inactivity timer. Hidden browser pages and problem changes
follow the existing terminal lifecycle.
The tab strip shares one content divider. Full-height tab targets use equal
workspace insets and a selected bottom indicator. Hover and pressed states affect
the strip surface; keyboard focus outlines the label separately from selection.
Selected-tool icon actions occupy the strip's right edge, outside the tablist's
arrow-key navigation. Terminal refresh, file download, browser navigation, web
refresh and the new-tab link share the compact square control class and radius.

`ProblemDetail` loads metadata once and emits the detail to the app. The app
passes declared files to `ProblemFiles` and observed run status to `ProblemWeb`;
features remain independent. Terminal preparation and reattachment refresh status
after an existing request finishes, so an early observation cannot hide a newly
prepared web service. Hidden terminals keep their dimensions and do not take focus
when an attachment finishes preparing.

`ProblemWeb` selects platform-provided `http://127.0.0.1:<port>` web origins.
Documents load on first use, retain state when switching tools or endpoints, and
reload only on an explicit web refresh. Environment termination or unknown state
removes old frames. A restarted endpoint loads its new address. Multiple HTTP
services have a labeled selector. The common tab strip offers back, forward,
reload and a native new-tab link for browser developer tools. An accessible
address field edits paths and query strings within the selected problem origin.
Reload refreshes the current page and retains its history. These actions preserve
service state. `UILink` keeps `noopener noreferrer` on the new-tab link.

The platform provides a separate loopback browsing proxy per declared HTTP endpoint,
with a trusted wrapper and a sandboxed inner problem frame. The wrapper reads the
inner frame's native Navigation API and validates commands from the player using
the exact parent origin and window. The player validates the wrapper origin,
window and session channel; displayed URLs remain within the selected problem.
No problem-side bridge or document injection is required. Common browsing is
platform-owned, so authors use existing HTTP declarations and exercise resources.
The API client requires the wrapper and target to share the same canonical
problem loopback origin, separate from the player origin. It also checks the
requested slug, endpoint name and wrapper path before preparing a frame.
The proxy reaches the isolated service through a fixed-destination Docker exec
stream with a platform-owned connector; problem containers have no published
ports or external network. See [network isolation](network-isolation.md).
The proxy streams problem content and preserves methods, bodies, paths, queries,
cookies and security headers, while mapping target-origin redirects. Requests use
the browser-visible Host so generated absolute links and Origin checks agree.
Every proxied request rechecks the current endpoint; workspace expiry and server
shutdown close listeners and upstream connections. Login cookies use the same
loopback hostname and retain the existing browser SameSite relationship.

The iframe has a title, no referrer and sandbox capabilities for scripts,
origin-based storage, forms, dialogs, popups and downloads. Top-level navigation
stays outside those capabilities. Platform credentials stay on the platform
origin. Production and development share a
`frame-src http://127.0.0.1:*` policy; the UI accepts only observed local entry
points, and the platform retains `frame-ancestors 'none'`.

Native history controls require the Navigation API in current browsers. The
wrapper reports unsupported navigation or inaccessible frames and retains the
new-tab path for developer tools. Target frame restrictions remain effective;
the platform preserves their security headers. This tool does not reproduce a
browser's full developer tools or external-site browsing.

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

## Syntax highlighting

The UI package owns exact dependencies `shiki@4.4.3` and `@shikijs/langs@4.4.3`.
`UICode` renders both Markdown fences and text material previews. A fence's
language hint takes priority; material filenames select their language by
extension. Common security challenge languages load individually on demand,
using a shared highlighter and the JavaScript regex engine.

The fixed theme maps tokens to CSS classes; Vue escapes their text. Token spans
use the existing stylesheet and CSP. The engine needs no additional WASM payload.
See [Shiki's engine documentation](https://shiki.style/guide/regex-engines).
Inline Markdown code stays plain. Unknown languages, loading failure and sources
above 131,072 UTF-16 code units retain escaped selectable source. Async results
are discarded after a source change or component unmount.

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
header. Play loads status on selection, refreshes it after submission and terminal
lifecycle events, and provides
a manual refresh. Failed observation clears stale endpoints; unavailable
resources direct the player to the terminal header's environment termination
and reconnect controls. Those controls remain available after failed
observation; unknown state disables submission.
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
edits update through HMR, including workspace UI and feature packages. The Vite
watcher includes domain, feature and shared package directories so changes to
type-only Vue props imports invalidate the compiler's type cache. Go proxies
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
