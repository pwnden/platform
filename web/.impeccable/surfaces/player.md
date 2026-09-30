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

At 76rem the terminal follows the reading column. At 48rem the problem list moves above the content, with bounded scrolling; the page uses the full width and vertical document scroll. The terminal viewport has a bounded independent layout so terminal row measurements cannot grow the page.

## Validation

Inspect actual 1920px desktop and 390px iframe renderings via Browser Plugin. Keep synthetic visual fixtures separate from runtime problem data. Build and behavior checks run with pinned Docker toolchains.
