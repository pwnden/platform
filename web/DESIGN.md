---
name: pwnden
description: A subtly luminous blue terminal workspace for local security challenges.
colors:
  background: "#05090f"
  surface: "#080e18"
  surface-raised: "#0d1725"
  surface-hover: "#122239"
  foreground: "#d2dfef"
  muted: "#8a9db6"
  border: "#1b2d43"
  border-active: "#3a729e"
  accent: "#7ccaff"
  accent-surface: "#102b43"
  success: "#8fdbba"
  warning: "#e7c38e"
  danger: "#ff9eac"
  terminal-background: "#050a12"
  terminal-selection: "#234367"
  terminal-magenta: "#c5acff"
  terminal-cyan: "#8cdce6"
  terminal-bright-white: "#f1f6ff"
typography:
  display:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1.35rem"
    fontWeight: 600
    lineHeight: 1.65
  headline:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 600
    lineHeight: 1.65
  title:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.65
  body:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "15px"
    fontWeight: 400
    lineHeight: 1.65
  label:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "0.85rem"
    fontWeight: 400
    lineHeight: 1.65
  status:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "0.8rem"
    fontWeight: 400
    lineHeight: 1.65
  footnote:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.65
  content-heading:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1.05rem"
    fontWeight: 600
    lineHeight: 1.5
  brand:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "1.2rem"
    fontWeight: 500
    lineHeight: 1.65
  recovery:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1.25rem"
    fontWeight: 600
    lineHeight: 1.65
  code:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "0.9rem"
    fontWeight: 400
    lineHeight: 1.6
  mobile-input:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "16px"
    fontWeight: 400
    lineHeight: 1.65
rounded:
  control: "0.1875rem"
spacing:
  ui-space-1: "0.5rem"
  ui-space-2: "1rem"
  ui-space-3: "1.5rem"
  ui-space-4: "2rem"
components:
  button-secondary:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: "0.5rem 0.9rem"
  button-primary:
    backgroundColor: "{colors.accent-surface}"
    textColor: "{colors.accent}"
    rounded: "{rounded.control}"
    padding: "0.5rem 0.9rem"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.muted}"
    rounded: "{rounded.control}"
    padding: "0.5rem 0.9rem"
  button-danger:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.danger}"
    rounded: "{rounded.control}"
    padding: "0.5rem 0.9rem"
  button-row:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    width: "100%"
    padding: "0.5rem 0.9rem"
  button-row-selected:
    backgroundColor: "{colors.accent-surface}"
    textColor: "{colors.accent}"
    rounded: "{rounded.control}"
    width: "100%"
    padding: "0.5rem 0.9rem"
  input:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: "0.6rem 0.8rem"
    width: "100%"
  panel:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.foreground}"
  terminal:
    backgroundColor: "{colors.terminal-background}"
    textColor: "{colors.foreground}"
    padding: "{spacing.ui-space-2}"
---

# Design System: pwnden

## Overview

**Creative North Star: "Hacker Terminal Workspace"**

Blue-tinted black surfaces and restrained blue light give the player a working terminal character. Pretendard makes problem text and controls readable during extended challenge sessions. Code and terminal text retain monospace typography. The workspace fills the browser viewport, with reading and execution surfaces divided by thin borders.

**Key Characteristics:**

- Subtly luminous dark blue on blue-tinted black.
- Pretendard for body text and controls; bundled monospace for code, terminal output and the brand.
- Full-width workspace with compact framing and bounded terminal geometry.
- State expressed through labels, tonal changes and visible keyboard focus.

The frontmatter records implemented values. CSS variables in [theme.css](packages/ui/src/theme.css) remain the runtime source; extension tokens and component previews live in [.impeccable/design.json](.impeccable/design.json).

## Colors

The palette layers blue-black surfaces beneath pale text and a cool blue accent.

### Primary

- **Terminal Blue** (`accent`): primary actions, links, cursor and connected status.
- **Deep Blue Fill** (`accent-surface`): primary controls and selected problem rows.
- **Active Blue Stroke** (`border-active`): resting input borders, selected controls, hover borders and scrollbar thumbs.

### Neutral

- **Blue Black** (`background`): document canvas and text fields.
- **Workspace Navy** (`surface`): panels, sidebar, header and footer.
- **Raised Navy** (`surface-raised`) and **Hover Navy** (`surface-hover`): button surfaces and enabled hover feedback.
- **Pale Blue Text** (`foreground`) and **Slate Blue Text** (`muted`): content and supporting labels.
- **Divider Navy** (`border`): panel boundaries and resting secondary button borders.
- **Terminal Black** (`terminal-background`) and **Selection Blue** (`terminal-selection`): terminal canvas and text selection.

Semantic success, warning and danger colors support status and terminal output. Danger also marks alerts and stop actions. Terminal magenta, cyan and bright white complete the implemented ANSI palette; they belong to command output.

**The Blue Action Rule.** Use the accent, its dark fill and the active stroke together for primary and selected controls.

## Typography

**Body font stack:** Pretendard, system-ui, sans-serif. Pretendard Variable 1.3.9 supports the body, headings and controls. **Code stack:** JetBrains Mono, D2Coding, monospace. All fonts are bundled and served locally with `font-display: swap`; license files and source revisions are recorded in [fonts/README.md](packages/ui/src/fonts/README.md).

The display role is the empty workspace heading; headline is the selected problem title; title is the shared section heading. Body sets the root scale at (15px), and label covers fields and categories. Status and footnote roles carry secondary state. Headings use weight (600); body uses (400). Numeric text uses tabular figures; ligatures are disabled.

Problem descriptions render Markdown, wrap at word boundaries, use a reading measure of (72ch) and line height of (1.85). Korean words stay together; controls keep whole labels and wrap as units. Code preserves literal whitespace inside a scrollable block. The real terminal uses (14px) monospace type and line height of (1.4). Text inputs use (16px) at the narrow layout breakpoint.

**The Reading Rule.** Body text, headings and controls use Pretendard. Code, terminal output and the brand use the monospace stack.

## Layout

The player occupies (100dvh) beneath a compact brand header. Two nested `UISplit` controls divide the catalog, reading and terminal columns. The catalog starts at (20%) of the workspace and adjusts between (14%) and (36%). Reading starts at (50%) of the remaining width and adjusts between (30%) and (70%). Both boundaries support pointer dragging and keyboard arrows through Sectile separator semantics. The catalog header, search and category controls stay outside the scrolling results. The selected problem title and category stay outside the independently scrolling body. Column changes preserve the selected problem and terminal session.

At (48rem), panes stack in reading order and the resize handles are hidden. The catalog occupies (30rem), with its results scrolling between filters and pagination. The terminal has a minimum height of (32rem), and the page uses document scrolling. The selected problem header stays sticky at the top of its content. Panel padding changes from `ui-space-3` to `ui-space-2`. The flag field and submit button stack when their own panel is narrower than (30rem). The document supports widths from (320px).

Shared panel bodies use `ui-space-3`; headings, terminal controls and output use `ui-space-2`. Control groups use `ui-space-1`. Panel headings wrap actions as whole units. Navigation labels ellipsize within narrow panes and retain full titles; URLs and file paths can wrap while prose preserves words.

**The Bounded Terminal Rule.** Keep terminal measurement inside a positioned, bounded viewport. The terminal screen can shrink to (6rem) on short desktop windows, with its renderer positioned within that screen.

At desktop heights of (32rem) or less, catalog filters, rows and pagination scroll together beneath the list header so every control remains reachable. The terminal body also scrolls when its controls and error messages exceed the available height.

Source: [App.vue](apps/player/src/App.vue), [UIPanel.vue](packages/ui/src/UIPanel.vue), [PlayPanel.vue](features/play/src/PlayPanel.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

## Elevation & Depth

Depth comes from tonal surfaces and dividing strokes. The shared blue glow supports primary button hover, selected rows and focused fields. Its exact value is recorded in the sidecar. Keyboard focus uses a (2px) accent outline with a (3px) offset; inputs bring that outline to their edge.

Control color, border and shadow transitions take (140ms). Reduced-motion preferences remove CSS transitions and animations. The xterm renderer enables its blinking cursor when connected.

## Shapes

Controls have small, nearly square corners using `rounded.control`. Panels meet through straight borders. General status is text with semantic color; terminal connection status uses a meaningful plug symbol with an accessible state label. Section headings use a raised navy surface and a dividing stroke.

## Components

### Buttons

The shared button has a minimum height of (2.75rem), a thin border and line height of (1.4). Primary, secondary, ghost, danger and row variants use the assignments in the frontmatter. Enabled hover moves to the hover surface and active stroke; primary hover adds the glow. Compact buttons use a minimum height of (2.5rem), horizontal padding of (0.65rem) and label size of (0.85rem). Disabled or busy buttons are disabled, muted and rendered at (0.5) opacity; busy controls expose `aria-busy`.

Source: [UIButton.vue](packages/ui/src/UIButton.vue), [theme.css](packages/ui/src/theme.css).

### Inputs / Fields

An associated label precedes a full-width field. The resting field uses the active blue stroke, a dark canvas and an accent caret. Focus adds the outline and shared glow; placeholder text is muted. Disabled fields use (0.5) opacity. `UITextField` wraps Sectile through the public UI package. `UISelect` supplies labeled, controlled native category choices.

Source: [UITextField.vue](packages/ui/src/UITextField.vue).

### Panels

Each panel is a semantic section with a labelled heading and optional actions. A lower heading border separates the content; the header has a minimum height of (3.5rem). `headingLevel` places catalog and terminal headings at level 2 and reading section headings at level 3 beneath the selected problem title. Brief, materials, execution and submission, hints and walkthrough share raised heading bands. Execution and submission precede the learning sections. Panels flex to fill available space while permitting their contents to shrink. The terminal panel uses the terminal background.

Source: [UIPanel.vue](packages/ui/src/UIPanel.vue).

### Navigation

Problem rows are compact full-width title buttons, grouped under readable Korean category headings. Search matches all words across titles, identifiers and categories; a category filter narrows results. Pages contain at most (20) rows, with the result range and previous/next controls below. `aria-pressed` identifies the current selection; its accent fill, stroke and glow provide the visual state. Search and pagination preserve the current problem and terminal session. The responsive sidebar behavior follows Layout.

Source: [ProblemList.vue](features/catalog/src/ProblemList.vue), [App.vue](apps/player/src/App.vue).

### Status

Inline text communicates muted, info, success or danger tone. Labels describe actual client or execution state. Alert text uses danger. Supporting status text uses (0.8rem). `UIConnectionStatus` uses an authored plug SVG to distinguish connected, connecting, disconnected and error states through shape and semantic color. Its accessible label and tooltip name the state. Connecting animates the current mark; reduced-motion preferences stop it.

Source: [UIStatus.vue](packages/ui/src/UIStatus.vue), [UIConnectionStatus.vue](packages/ui/src/UIConnectionStatus.vue).

### Terminal

`UITerminal` owns xterm and its fit addon inside the UI package. It maps shared tokens to the terminal theme, enables screen reader mode and resizes after fonts load or column changes. Its public resize event contains only `cols` and `rows`. Selecting a problem prepares and connects its environment automatically. `UITerminalControls` keeps connection status, disconnect/reconnect or preparation cancellation, and environment stop together in the panel header. Below a pane width of (20rem), connection buttons retain accessible names and tooltips while their visible labels give way to icons. The status and controls stay together when the heading wraps. Disconnect retains the environment for the existing ten-minute inactivity window and waits for explicit reconnect; environment stop removes it. The renderer preserves output after shell exit. Failure and exit messages appear above the terminal; successful connection is conveyed in the header. The cleanup footnote stays below the viewport.

Source: [UITerminal.vue](packages/ui/src/UITerminal.vue), [UITerminalControls.vue](packages/ui/src/UITerminalControls.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

### Markdown

`UIMarkdown` renders source through md4x's WASM parser and an allowlisted Vue AST renderer. Source headings fit below the panel title. Inline code uses the accent and raised surface; code blocks use the terminal background. Tables scroll inside their own region. Quotes use a thin active stroke, and task markers are read-only. Links use the shared accent and keyboard focus. Images display alternative text. Loading is announced; failure offers retry and escaped source.

`UIReveal` uses native details/summary semantics with shared border, spacing and visible keyboard focus. File sources, individually numbered hints and answer-labeled walkthroughs open inside the reading pane. Closed spoilers have no content DOM. `UICode` displays escaped selectable source, preserves whitespace, and scrolls in both dimensions with a maximum height of (32rem). The catalog feature fetches revealed content on demand and keeps it across close/reopen within the selected problem. Text previews stop at (1 MiB); binary and larger materials direct players to the prepared terminal. Downloads are secondary actions.

Source: [UIMarkdown.vue](packages/ui/src/UIMarkdown.vue), [markdown.ts](packages/ui/src/markdown.ts).

## Do's and Don'ts

### Do:

- **Do** use shared color, spacing and corner tokens for controls and panels.
- **Do** use Pretendard for reading and controls, and bundled monospace for code, terminal output and the brand.
- **Do** keep input borders visible at rest and provide the shared keyboard focus outline.
- **Do** preserve the full viewport layout and bounded terminal screen when adapting widths.
- **Do** pair status colors with meaningful labels derived from actual state.
- **Do** honor reduced-motion preferences for CSS animation and transitions.
