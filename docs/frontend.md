# Frontend workspace

The browser client lives in `web/`. Go owns the local server, authentication,
problem execution and answer checking. Node and pnpm are build tools.

## Packages and ownership

| Package | Location | Responsibility |
| --- | --- | --- |
| `@pwnden/player` | `web/apps/player` | Session initialization and composition. |
| `@pwnden/ui` | `web/packages/ui` | Own button/input API, Sectile wrappers and theme tokens. |
| `@pwnden/api` | `web/packages/api` | Implement domain ports with the Go HTTP API. |
| `@pwnden/catalog` | `web/domains/catalog` | Summary/detail/file models and read/download port; pure TypeScript. |
| `@pwnden/play` | `web/domains/play` | Run/status/submission models and player port; pure TypeScript. |
| `@pwnden/catalog-feature` | `web/features/catalog` | Vue catalog, description text and authenticated file downloads. |
| `@pwnden/play-feature` | `web/features/play` | Vue observed execution state, controls and submissions. |

The app injects API implementations into features. Features import their domain
and the UI package. The API package imports domain ports and models. Domains
use ES libraries without browser, framework or transport dependencies. The UI
package alone imports Sectile; its public props and events belong to pwnden.
Progress and terminal packages are introduced with their actual capabilities.

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

`typescript` is an exact npm alias to `typescript-native-bridge`, also overridden
throughout the workspace. Both `tsc` and `vue-tsc` report `TNB ACTIVE` during
checking. Direct, peer and development dependencies use exact versions, and
internal dependencies use `workspace:0.1.0`. `saveExact` is enabled. pnpm 12
records package-manager resolution and application resolution as two YAML
documents in the committed lockfile; use the pinned pnpm to consume it.

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

The Vue app composes catalog and play features for the seven HTTP operations,
including detail, declared file downloads and observed run status. Descriptions
are displayed as text, with no HTML execution or active author links. Download
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

`./setup` builds the pinned Node frontend stage inside Docker, copies its
generated assets into the Go stage, and packages the executable and official
catalog. The executable needs neither Node nor a separate asset directory.
Source Go builds and tests must first prepare the ignored embed directory:

```sh
docker build --file Dockerfile.web --target assets --output type=local,dest=internal/playerweb/dist .
go test ./...
```

The embed directives make a missing frontend build a compile error. The
bootstrap context excludes host `node_modules` and generated frontend directories.
A separate Vite server does not provide authenticated access to Go's API: the
server allows its own local origin. Container terminal access and submission
history remain subsequent implementation stages. Reopening a full server URL and
selecting the problem recovers its current run from Go, without browser persistence.
