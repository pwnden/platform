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

Component insets use one padding value for all four sides. Shared control
insets derive from the control height and content line height. Panel headings
and bodies share their panel inset. Sibling spacing uses gap; nested contours
derive their inner radius from the outer radius and the complete inset.
`pnpm check:spacing` enforces padding declarations in owned Vue and CSS sources.

Loading transitions retain control dimensions and sibling positions. Keep labels
stable and express progress within an existing icon box when the control has one.
