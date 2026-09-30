# Frontend conventions

Follow the repository abbreviation convention. The boundary checker checks
common abbreviations in source identifiers; negative test fixtures retain the
invalid spelling they are testing.

`tools/boundaries.mjs` owns allowed dependencies. Domain packages use pure
TypeScript and own their ports. The app injects `packages/api` implementations
into features. Features consume their domain and `packages/ui`. Only
`packages/ui` imports Sectile and it exports its own component props and events.

Use public workspace package exports for imports between packages. Each direct,
peer, development and optional dependency has an exact version. Run
`pnpm verify` with the versions pinned in this workspace, or run the root
`Dockerfile.web` verification target.
