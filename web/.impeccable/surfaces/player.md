# Player workspace

## Mode

Operate: understand a challenge's objective and context, inspect materials, use its service and isolated terminal, submit a flag, and learn through progressive hints and a complete explanation.

## Direction

User-selected hacker terminal language: subtly luminous dark blue on blue-tinted black, monospace typography for Latin and Korean, full browser viewport.

## First viewport

A compact header and footer frame a full-width workspace. A persistent problem sidebar leads to a reading and execution column, with a large terminal occupying the remaining width and height. Empty state teaches selection. Each state indicator reflects actual client, problem or terminal state.

## Interaction

Selecting a problem highlights its row and opens the associated workspace. Selecting the current problem keeps its terminal session. Focus and primary actions use the blue accent. The empty prompt softly pulses; reduced-motion preferences stop its animation.

The brief describes the player's situation and starting actions. Materials open as escaped source with Shiki syntax colors in the reading pane; downloading is optional. Markdown code fences use the same code renderer. Hints open individually in increasing specificity. The walkthrough is labeled as containing answers and loads only when opened. All documents stay within the workspace, with independent loading, error and retry states. Large or binary materials use the prepared terminal. Player steps use actual UI labels and the prepared environment.

The terminal header groups a plug-shaped state indicator, disconnect/reconnect or preparation cancellation, and environment stop in one row. State is communicated by shape and semantic color with an accessible label. Disconnect retains the existing environment for its ten-minute inactivity window and waits for explicit reconnect. Successful connection is conveyed in the header; error and exit messages appear above the terminal. Narrow panes keep controls together using named icon buttons.

## Responsive behavior

Catalog, description and terminal columns have two draggable and keyboard-accessible boundaries. Resizing preserves the terminal connection and refits its rows and columns. At 48rem the panes stack in reading order, with bounded catalog scrolling and full-width document scrolling. Korean prose keeps words together; control labels remain whole. The terminal viewport has a bounded independent layout so terminal row measurements cannot grow the page.

## Validation

Inspect actual 1920px desktop and 390px iframe renderings via Browser Plugin. Keep synthetic visual fixtures separate from runtime problem data. Build and behavior checks run with pinned Docker toolchains.
