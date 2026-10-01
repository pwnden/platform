---
name: pwnden
description: A subtly luminous blue terminal workspace for local security challenges.
colors:
  background: "#05090f"
  surface: "#080e18"
  surface-raised: "#0d1725"
  surface-hover: "#122239"
  surface-pressed: "#19324b"
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
  syntax-keyword: "#c5acff"
  syntax-function: "#8cdce6"
typography:
  display:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1.35rem"
    fontWeight: 600
    lineHeight: 1.65
  headline:
    fontFamily: "'Pretendard', system-ui, sans-serif"
    fontSize: "1rem"
    fontWeight: 600
    lineHeight: 1.4
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
  flush: "0px"
  inline: "0.125rem"
  control: "0.25rem"
  surface: "0.5rem"
spacing:
  ui-inset-control: "calc((2.75rem - 1.4rem) / 2 - 1px)"
  ui-inset-compact: "calc((2.5rem - 1.19rem) / 2 - 1px)"
  ui-space-1: "0.5rem"
  ui-space-2: "1rem"
  ui-space-3: "1.5rem"
  ui-space-4: "2rem"
components:
  button-secondary:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: "{spacing.ui-inset-control}"
  button-primary:
    backgroundColor: "{colors.accent-surface}"
    textColor: "{colors.accent}"
    rounded: "{rounded.control}"
    padding: "{spacing.ui-inset-control}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.muted}"
    rounded: "{rounded.control}"
    padding: "{spacing.ui-inset-control}"
  button-danger:
    backgroundColor: "{colors.surface-raised}"
    textColor: "{colors.danger}"
    rounded: "{rounded.control}"
    padding: "{spacing.ui-inset-control}"
  button-row:
    backgroundColor: "transparent"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    width: "100%"
    padding: "{spacing.ui-inset-control}"
  button-row-selected:
    backgroundColor: "{colors.accent-surface}"
    textColor: "{colors.accent}"
    rounded: "{rounded.control}"
    width: "100%"
    padding: "{spacing.ui-inset-control}"
  input:
    backgroundColor: "{colors.background}"
    textColor: "{colors.foreground}"
    rounded: "{rounded.control}"
    padding: "{spacing.ui-inset-control}"
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

At (48rem), panes stack in reading order and the resize handles are hidden. The catalog occupies (30rem), with its results scrolling between filters and pagination. The terminal has a minimum height of (32rem), and the page uses document scrolling. The selected problem header stays sticky at the top of its content. Reading panel padding changes from `ui-space-3` to `ui-space-2`. The execution panel uses `ui-space-2` at all widths. The flag field and submit button stack at full width when their own panel is narrower than (24rem). The document supports widths from (320px).

Each container uses the same inset on all four sides. Panel headings and bodies share `--ui-panel-inset`: reading sections use `ui-space-3` on desktop and `ui-space-2` on narrow screens; execution, catalog and terminal sections use `ui-space-2`. Catalog filters, results and pagination use the same inset. The embedded terminal receives its inset once from the panel body. Control groups use `ui-space-1` as gap. Panel headings wrap actions as whole units. Navigation labels ellipsize within narrow panes and retain full titles; URLs and file paths can wrap while prose preserves words.

The selected problem title and its category badge share one header row. The
title uses the shared section scale; a long title ellipsizes and retains its
full-title tooltip. `--ui-workspace-header-size` derives the minimum height from
the compact control, two equal workspace insets and a divider. Category badges
use the control radius and equal padding, with accessible category context.

The execution section shows service entry points and the flag form. Connecting
the terminal prepares the environment; its header owns connection and immediate
environment termination. Entry links use the shared control recipe through
`UILink`. New-tab navigation has an external-link icon and accessible description. A divider
separates the flag form; its field and submit button share the default height.
Status and an icon-only refresh action remain together in the section header.
Status refresh retains the body, entry links, editable flag input and submission
result throughout observation. Progress appears in the existing header icon box
and respects reduced-motion preferences. A completed state change or observation
failure updates the content; unknown state disables submission and removes stale
entry points.

**The Equal Inset Rule.** Top, right, bottom and left padding use one value. Control padding derives from border-box height, content line height and border thickness. Sibling rhythm uses gap. Responsive layouts replace the entire inset. `pnpm check:spacing` validates owned CSS and Vue style declarations through the existing CSS parser and runs as part of `pnpm verify`.

**The Stable Loading Rule.** Loading transitions retain control dimensions and
sibling positions. Material download retains its label and uses the existing
icon box for progress. The control stays disabled and semantically busy during
the request. Reduced-motion preferences keep the progress icon static.

**The Bounded Terminal Rule.** Keep terminal measurement inside a positioned, bounded viewport. The terminal screen can shrink to (6rem) on short desktop windows, with its renderer positioned within that screen.

At desktop heights of (32rem) or less, catalog filters, rows and pagination scroll together beneath the list header so every control remains reachable. The terminal body also scrolls when its controls and error messages exceed the available height.

Source: [App.vue](apps/player/src/App.vue), [UIPanel.vue](packages/ui/src/UIPanel.vue), [PlayPanel.vue](features/play/src/PlayPanel.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

## Elevation & Depth

Depth comes from tonal surfaces and dividing strokes. The shared blue glow supports primary button hover. Keyboard focus uses the shared (2px) accent outline inside the control boundary so scroll containers retain the indicator. Inline links and disclosure labels use an external (2px) offset. Disclosure focus follows its label and chevron. Pointer focus follows the browser's `:focus-visible` semantics; editable fields also show an accent border while focused.

Control color, border and shadow transitions take (140ms) with ease-out timing. Enabled hover uses the hover surface and active stroke on devices that support hover. Pressing a control uses the pressed surface and accent border; selected rows keep their accent fill after release. Expanded disclosures use the accent label and rotated chevron. Disabled controls retain their resting appearance at reduced opacity. Forced-color mode uses the system Highlight for focus and selection borders. Reduced-motion preferences remove CSS transitions and animations. The xterm renderer enables its blinking cursor when connected. These states belong to the shared UI package.

## Shapes

Buttons, inputs, disclosures and switch tracks share `rounded.control`. Inline code, links and disclosure focus labels use `rounded.inline`; standalone code blocks and material lists use `rounded.surface`. Adjoining workspace panels and interior file rows use `rounded.flush`. General status is text with semantic color; terminal connection uses a rectangular switch with a square thumb and accessible state text. Section headings use a raised navy surface and a dividing stroke.

Nested contours derive the inner radius from the outer radius minus the complete border and padding inset, clamped to zero. `theme.css` provides `.ui-radius-outer` and `.ui-radius-inner`, with inherited `--ui-radius-outer` and `--ui-radius-inset` values. The calculation stays on the child so local overrides work. The switch applies it to its track and thumb. Material lists apply it only to the first and last exposed edges; preview content follows those corners while focus indicators remain visible.

## Components

### Buttons

The shared button has a minimum height of (2.75rem), a thin border, font size of (1rem) and line height of (1.4). Its four-sided inset is `(height - line height) / 2 - border`. Primary, secondary, ghost, danger and row variants use the assignments in the frontmatter. Hover, pressing, selection and keyboard focus follow Elevation & Depth. Compact buttons use a fixed height of (2.5rem), label size of (0.85rem), content line height of (1.19rem) and the same inset calculation. Icon-only controls center their icon in a square. Disabled or busy buttons are disabled, muted and rendered at (0.5) opacity; busy controls expose `aria-busy`.

Source: [UIButton.vue](packages/ui/src/UIButton.vue), [theme.css](packages/ui/src/theme.css).

### Inputs / Fields

An associated label precedes a full-width field. Its height and four-sided inset follow default controls, with a fixed (1.4rem) content line box. The resting field uses the active blue stroke, a dark canvas and an accent caret. Hover and focus accent the border; keyboard focus adds the shared inset outline. Placeholder text is muted. Invalid fields use the danger border. Disabled fields use (0.5) opacity. `UITextField` wraps Sectile through the public UI package. `UISelect` places the selected label and chevron in an inset grid. A native select covers the complete field, preserving click, keyboard and accessibility behavior; focus and disabled states apply to its visible wrapper.

Source: [UITextField.vue](packages/ui/src/UITextField.vue).

### Panels

Each panel is a semantic section with a labelled heading and optional actions. A lower heading border separates the content; the header has a minimum height of (3.5rem). `headingLevel` places catalog and terminal headings at level 2 and reading section headings at level 3 beneath the selected problem title. Brief, materials, execution and submission, hints and walkthrough share raised heading bands. Execution and submission precede the learning sections. Panels flex to fill available space while permitting their contents to shrink. The terminal panel uses the terminal background.

The execution panel groups its current state and refresh action in the header. Refresh queries server state. Service actions occupy the body for service problems, while ready file problems begin directly with the submission form. The form receives a divider only when body content precedes it.

Source: [UIPanel.vue](packages/ui/src/UIPanel.vue), [PlayPanel.vue](features/play/src/PlayPanel.vue).

### Navigation

Problem rows are compact full-width title buttons, grouped under readable Korean category headings. Search matches all words across titles, identifiers and categories; a category filter narrows results. Pages contain at most (20) rows, with the result range and previous/next controls below. `aria-pressed` identifies the current selection through accent fill and stroke. Keyboard focus remains a separate indicator. Search and pagination preserve the current problem and terminal session. The responsive sidebar behavior follows Layout.

Source: [ProblemList.vue](features/catalog/src/ProblemList.vue), [App.vue](apps/player/src/App.vue).

### Status

Inline text communicates muted, info, success or danger tone. Labels describe actual client or execution state. Alert text uses danger. Supporting status text uses (0.8rem). The terminal connection state is shown beside its switch as an explicit label with semantic color.

Source: [UIStatus.vue](packages/ui/src/UIStatus.vue), [UITerminalControls.vue](packages/ui/src/UITerminalControls.vue).

### Terminal

`UITerminal` owns xterm and its fit addon inside the UI package. It maps shared tokens to the terminal theme, enables screen reader mode and resizes after fonts load or column changes. Its public resize event contains only `cols` and `rows`. Selecting a problem prepares and connects its environment automatically. `UITerminalControls` places a compact rectangular connection switch and an icon-only environment-stop button in the panel header. The switch has a (1.75rem) square thumb with breathing room. Its usable track width is exactly two thumb widths; checked state translates the thumb by (100%) of its width. The connected chain, disconnected chain and preparation progress icons occupy the thumb. Control names and state remain in accessible text and tooltips. The switch represents connection intent: connected and preparing states are on; preparing exposes `aria-busy` and can be cancelled by switching off. Disconnected and error states are off. `UISwitch` owns the Sectile wrapper, shared hover/pressed/focus styles and reduced-motion behavior. Both controls use the shared compact hit height and wrap as complete units within narrow panes. Switching off retains the environment for the existing ten-minute inactivity window and waits for switching on; environment stop immediately removes it. Switching on after environment stop prepares a fresh environment. The renderer preserves output after shell exit. Failure and exit messages appear above the terminal; successful connection is conveyed in the header. The cleanup footnote stays below the viewport.

Source: [UITerminal.vue](packages/ui/src/UITerminal.vue), [UISwitch.vue](packages/ui/src/UISwitch.vue), [UITerminalControls.vue](packages/ui/src/UITerminalControls.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

### Markdown

`UIMarkdown` renders source through md4x's WASM parser and an allowlisted Vue AST renderer. Source headings fit below the panel title. Inline code uses the accent and raised surface; code blocks use the terminal background. Tables scroll inside their own region. Quotes use a thin active stroke, and task markers are read-only. Links use the shared accent and keyboard focus. Images display alternative text. Loading is announced; failure offers retry and escaped source.

`UIReveal` uses native details/summary semantics with shared border, spacing and compact keyboard focus. Individually numbered hints and answer-labeled walkthroughs open inside the reading pane. Closed spoilers have no content DOM. `UICode` displays escaped selectable source, preserves indentation and blank lines, and scrolls in both dimensions with a maximum height of (32rem). Both Markdown fences and material previews use Shiki with a shared blue-black theme. Fences supply their language; material filenames supply their extension. Keywords use syntax magenta, functions use syntax cyan, strings use success, numbers use warning, types use accent and comments use muted. Grammars and the JavaScript regex engine load on demand. Vue renders tokens as text spans with fixed CSS classes under the existing CSP. Unknown languages, initialization failure and large sources remain readable as escaped text. The catalog feature fetches revealed content on demand and keeps it across close/reopen within the selected problem. Text previews stop at (1 MiB); binary and larger materials direct players to the prepared terminal.

`UIFile` groups a material's filename, compact byte size and secondary download action in one header. The filename leads; its parent directory is muted. Long paths ellipsize and retain the full path in accessible labels and a tooltip. The disclosure and download are separate native buttons. Opening source places the scrollable code directly beneath its header within one file-list frame. Loading, notices and retry remain inside that preview. Below a container width of (28rem), size moves beneath the filename; below (20rem), download uses its named icon. Download stays in the header throughout preview loading and expansion. The UI package owns presentation; the catalog feature owns file identity, fetching and cache.

Source: [UIMarkdown.vue](packages/ui/src/UIMarkdown.vue), [markdown.ts](packages/ui/src/markdown.ts), [UICode.vue](packages/ui/src/UICode.vue), [syntax.ts](packages/ui/src/syntax.ts), [UIFile.vue](packages/ui/src/UIFile.vue).

## Do's and Don'ts

### Do:

- **Do** use shared color, spacing and corner tokens for controls and panels.
- **Do** use Pretendard for reading and controls, and bundled monospace for code, terminal output and the brand.
- **Do** keep input borders visible at rest and provide the shared keyboard focus outline.
- **Do** preserve the full viewport layout and bounded terminal screen when adapting widths.
- **Do** pair status colors with meaningful labels derived from actual state.
- **Do** honor reduced-motion preferences for CSS animation and transitions.
