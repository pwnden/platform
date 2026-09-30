# Player workspace

## Mode

Operate: read a challenge, inspect files, use the isolated terminal and submit a flag.

## Direction

User-selected hacker terminal language: subtly luminous dark blue on blue-tinted black, monospace typography for Latin and Korean, full browser viewport.

## First viewport

A compact header and footer frame a full-width workspace. A persistent problem sidebar leads to a reading and execution column, with a large terminal occupying the remaining width and height. Empty state teaches selection. Each state indicator reflects actual client, problem or terminal state.

## Interaction

Selecting a problem highlights its row and opens the associated workspace. Selecting the current problem keeps its terminal session. Focus and primary actions use the blue accent. The empty prompt softly pulses; reduced-motion preferences stop its animation.

## Responsive behavior

Catalog, description and terminal columns have two draggable and keyboard-accessible boundaries. Resizing preserves the terminal connection and refits its rows and columns. At 48rem the panes stack in reading order, with bounded catalog scrolling and full-width document scrolling. Korean prose keeps words together; control labels remain whole. The terminal viewport has a bounded independent layout so terminal row measurements cannot grow the page.

## Validation

Inspect actual 1920px desktop and 390px iframe renderings via Browser Plugin. Keep synthetic visual fixtures separate from runtime problem data. Build and behavior checks run with pinned Docker toolchains.
