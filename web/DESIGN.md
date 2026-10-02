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
  difficulty-intro: "#CBD5E1"
  difficulty-easy: "#64D6A4"
  difficulty-medium: "#70A5FF"
  difficulty-hard: "#B79AFF"
  difficulty-expert: "#F27C9B"
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

The player occupies (100dvh) beneath a compact brand header. Two nested `UISplit` controls divide the catalog, reading and tool columns. The catalog starts at (20%) of the workspace and adjusts between (14%) and (36%). Reading starts at (50%) of the remaining width and adjusts between (30%) and (70%). Both boundaries support pointer dragging and keyboard arrows through Sectile separator semantics. The catalog header, search and category controls stay outside the scrolling results. The selected problem title and category stay outside the independently scrolling body. Column changes preserve the selected problem, terminal session and web document.

At (48rem), panes stack in reading order and the resize handles are hidden. The catalog occupies (30rem), with its results scrolling between filters and pagination. The tool pane occupies (36rem), and the page uses document scrolling. The selected problem header stays sticky at the top of its content. Reading panel padding changes from `ui-space-3` to `ui-space-2`. The execution panel uses `ui-space-2` at all widths. The flag field and submit button stack at full width when their own panel is narrower than (20rem). The document supports widths from (320px).

Each container uses the same inset on all four sides. Panel headings and bodies share `--ui-panel-inset`: reading sections use `ui-space-3` on desktop and `ui-space-2` on narrow screens; execution, catalog and terminal sections use `ui-space-2`. Catalog filters, results and pagination use the same inset. The embedded terminal receives its inset once from the panel body. Control groups use `ui-space-1` as gap. Panel headings wrap actions as whole units. Navigation labels ellipsize within narrow panes and retain full titles; URLs and file paths can wrap while prose preserves words.

The selected problem title and its category badge share one header row. The
title uses the shared section scale; a long title ellipsizes and retains its
full-title tooltip. `--ui-workspace-header-size` derives the minimum height from
the compact control, two equal workspace insets and a divider. Category badges
use the control radius and equal padding, with accessible category context.

The submission bar is fixed below the right-side tool pane, outside its scrolling contents. Its field and button share the default control height; narrow panes stack them. A fixed-height caption holds the result and reserves a compact action slot. Environment failures reveal a reconnect action in that slot. An accepted flag remains visible and readonly, styled with the success color and a check icon to the right of the completed button label. Loading, rejection and connection errors update existing slots and controls. Tool errors and progress use overlays within their viewports so adjacent controls keep their positions. TCP endpoints appear in terminal tab actions.

The right-side tools use `UITabs`, with a continuous navigation strip at the
workspace-header height. Each tab fills the strip height and uses the workspace
inset equally on all sides. The declared `[player].tools` determine visible tools and their order; the first tool is the default. Note Vault uses Web; Rotor Lock uses Files and Terminal. The selected tab
uses accent text and a bottom indicator along the shared content edge. Hover and
press affect the tab surface; keyboard focus outlines its label separately from
selection. Navigation uses the flush radius and the label uses the inline radius. Sectile owns
arrow-key navigation and linked tab/panel semantics. Inactive panels are inert,
hidden from assistive technology and visually excluded while their components
remain mounted.

Files owns distribution preview and download in an independently scrolling
panel. Web provides a bounded iframe viewport beneath a fixed toolbar, with
compact reload and new-tab actions of equal dimensions and symmetric insets.
Multiple HTTP services have a labeled selector. The viewport uses the surface
radius with inner radius reduced by its border. Source, shell and opened web
document state persist across tool switches. Browser-page visibility and problem
switches retain the existing environment lifecycle. Frames load on first use;
stopping the environment removes old frames and reconnecting uses the new address.

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

Buttons, inputs, disclosures and switch tracks share `rounded.control`. Inline code, links and disclosure focus labels use `rounded.inline`; standalone code blocks and framed web documents use `rounded.surface`. Adjoining workspace panels use `rounded.flush`. General status is text with semantic color; tool actions use compact square icon controls with accessible names and progress state. Section headings use a raised navy surface and a dividing stroke.

Nested contours derive the inner radius from the outer radius minus the complete border and padding inset, clamped to zero. `theme.css` provides `.ui-radius-outer` and `.ui-radius-inner`, with inherited `--ui-radius-outer` and `--ui-radius-inset` values. The calculation stays on the child so local overrides work. The switch applies it to its track and thumb. Material lists apply it only to the first and last exposed edges; preview content follows those corners while focus indicators remain visible.

## Components

### Difficulty badges

Challenges declare levels 1–5: Intro, Easy, Medium, Hard and Expert. The UI owns
their quartz, emerald, sapphire, amethyst and ruby colors, respectively
`#CBD5E1`, `#64D6A4`, `#70A5FF`, `#B79AFF` and `#F27C9B`. Each badge uses the
full color for text, a 10% background and a 30% border. Shared control corners
and equal four-sided padding apply. Labels and a screen-reader difficulty prefix
convey the level independently of color.

`UIDifficultyBadge.vue` composes `UIBadge.vue`; `difficulty.ts` owns the UI labels
and tones, while `theme.css` owns the palette. Badges appear beside catalog titles
and beside the category in the fixed problem header. Long titles truncate.
Catalog rows share a minimum height that accommodates badges. Legacy problems
without a declared level have no difficulty badge.

Search, category and difficulty filters combine. Ordering selects name,
easiest or hardest within each category; unrated problems sort last in either
difficulty direction. Changing a filter or order resets pagination and scroll
while preserving the current problem selection.

### Buttons

The shared button has a minimum height of (2.75rem), a thin border, font size of (1rem) and line height of (1.4). Its four-sided inset is `(height - line height) / 2 - border`. Primary, secondary, ghost, danger and row variants use the assignments in the frontmatter. Hover, pressing, selection and keyboard focus follow Elevation & Depth. Compact buttons use a fixed height of (2.5rem), label size of (0.85rem), content line height of (1.19rem) and the same inset calculation. Icon-only controls center their icon in a square. Disabled or busy buttons are disabled, muted and rendered at (0.5) opacity; busy controls expose `aria-busy`.

Source: [UIButton.vue](packages/ui/src/UIButton.vue), [theme.css](packages/ui/src/theme.css).

### Inputs / Fields

An associated label precedes a full-width field. Its height and four-sided inset follow default controls, with a fixed (1.4rem) content line box. The resting field uses the active blue stroke, a dark canvas and an accent caret. Hover and focus accent the border; keyboard focus adds the shared inset outline. Placeholder text is muted. Invalid fields use the danger border. Disabled fields use (0.5) opacity. `UITextField` wraps Sectile through the public UI package. `UISelect` places the selected label and chevron in an inset grid. A native select covers the complete field, preserving click, keyboard and accessibility behavior; focus and disabled states apply to its visible wrapper.

Source: [UITextField.vue](packages/ui/src/UITextField.vue).

### Panels

Each reading panel is a semantic section with a labelled heading and optional actions. A lower heading border separates the content; the header has a minimum height of (3.5rem). `headingLevel` places the catalog heading at level 2 and reading section headings at level 3 beneath the selected problem title. Brief, hints and walkthrough share raised heading bands. Tool contents use accessible region names beneath their shared tab strip. Panels flex to fill available space while permitting their contents to shrink. The terminal uses the terminal background.

The submission bar groups its fixed feedback slot and error-only reconnect action above the form. Reconnect retries problem presence without resetting the environment. Submission labels precede their progress, completion and action icons. TCP addresses occupy the terminal tab actions when present. The tool pane has one fixed tab strip; selected-tool actions occupy its right edge, outside the tablist. Content fills the remaining bounded area above submission.

Source: [UIPanel.vue](packages/ui/src/UIPanel.vue), [PlayPanel.vue](features/play/src/PlayPanel.vue).

### Navigation

Problem rows are compact full-width title buttons, grouped under readable Korean category headings. Search matches all words across titles, identifiers and categories; a category filter narrows results. Pages contain at most (20) rows, with the result range and previous/next controls below. `aria-pressed` identifies the current selection through accent fill and stroke. Keyboard focus remains a separate indicator. Search and pagination preserve the current problem and terminal session. The responsive sidebar behavior follows Layout.

Source: [ProblemList.vue](features/catalog/src/ProblemList.vue), [App.vue](apps/player/src/App.vue).

### Status

Inline text communicates muted, info, success or danger tone. Labels describe actual client or execution state. Alert text uses danger. Supporting status text uses (0.8rem). Terminal preparation is announced over the viewport and through the busy refresh control; failures and shell exit appear in overlays within the viewport.

Source: [UIStatus.vue](packages/ui/src/UIStatus.vue), [UIIconButton.vue](packages/ui/src/UIIconButton.vue).

### Terminal

`UITerminal` owns xterm and its fit addon inside the UI package. It maps shared tokens to the terminal theme, enables screen reader mode and resizes after fonts load or column changes. Its public resize event contains only `cols` and `rows`. Problem presence prepares the environment independently. A declared Terminal tool starts its shell on first activation. The tab strip's terminal refresh action reattaches to the existing shell, retaining files, working directory and jobs. Server snapshots restore output. After shell exit it creates a new shell. Preparation disables repeated refresh and uses an overlay to preserve the viewport geometry. Leaving starts the ten-minute inactivity timer; returning cancels it. At the ten-environment limit, an inline recovery list offers explicit cleanup of a retained environment.

Source: [UITerminal.vue](packages/ui/src/UITerminal.vue), [UIIconButton.vue](packages/ui/src/UIIconButton.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

### Markdown

`UIMarkdown` renders source through md4x's WASM parser and an allowlisted Vue AST renderer. Source headings fit below the panel title. Inline code uses the accent and raised surface; code blocks use the terminal background. Tables scroll inside their own region. Quotes use a thin active stroke, and task markers are read-only. Links use the shared accent and keyboard focus. Images display alternative text. Loading is announced; failure offers retry and escaped source.

Requester dialogue uses the MDC `message` block. Its optional sender uses accent color and weight (600); the body uses foreground color and type size (1.05rem). The accent surface, surface radius and equal inset of space-3 distinguish the message from narrative text. Ordinary quotations keep their general quote treatment. Message markup is fixed by the UI renderer, with sender text escaped and author DOM attributes omitted.

Briefings use semantic MDC blocks with one UI-owned heading per role. `objective` uses the hover surface, accent heading and (1.05rem) body to emphasize the mission. `resources` groups supplied data on the raised surface. `knowledge` uses a top divider and open explanation layout. `submission` uses a compact wrapping label-and-body row with a top divider. Section insets use equal space-3 padding, or space-2 for submission. Headings follow the panel's heading offset. Narrative context remains ordinary prose.

`UIReveal` uses native details/summary semantics with shared border, spacing and compact keyboard focus. Individually numbered hints and answer-labeled walkthroughs open inside the reading pane. Closed spoilers have no content DOM. `UICode` displays escaped selectable source, preserves indentation and blank lines, and scrolls in both dimensions with a maximum height of (32rem). Both Markdown fences and material previews use Shiki with a shared blue-black theme. Fences supply their language; material filenames supply their extension. Keywords use syntax magenta, functions use syntax cyan, strings use success, numbers use warning, types use accent and comments use muted. Grammars and the JavaScript regex engine load on demand. Vue renders tokens as text spans with fixed CSS classes under the existing CSP. Unknown languages, initialization failure and large sources remain readable as escaped text. The catalog feature fetches revealed content on demand and keeps it across close/reopen within the selected problem. Text previews stop at (1 MiB); binary and larger materials remain available through download.

The Files tool displays the selected file immediately on first entry. A single file has a compact path label; multiple files use a named selector. Source fills the available height with independent scrolling. The selected file's download icon lives at the right edge of the common tab strip. File sizes remain internal metadata for preview limits. The catalog feature owns file selection, fetching, retry and cache through a public handle. Terminal refresh, file download, web refresh and the new-tab link share the same compact square hit box, control radius and icon size. Busy icons retain their dimensions.

The Web tool groups back, forward and reload immediately to the left of its address field. All four controls share the compact height and control radius; equal gaps separate them. The field fills the remaining row width and shrinks in narrower panes while icon buttons keep their square hit boxes. The new-tab link remains at the right edge of the common tab strip.

Source: [UIMarkdown.vue](packages/ui/src/UIMarkdown.vue), [markdown.ts](packages/ui/src/markdown.ts), [UICode.vue](packages/ui/src/UICode.vue), [syntax.ts](packages/ui/src/syntax.ts), [ProblemFiles.vue](features/catalog/src/ProblemFiles.vue), [ProblemWeb.vue](features/play/src/ProblemWeb.vue).

The web address field shows the current path, query and fragment. Players enter paths directly; the platform supplies the problem origin. Unusable input restores the current path while preserving the displayed page.

## Do's and Don'ts

### Do:

- **Do** use shared color, spacing and corner tokens for controls and panels.
- **Do** use Pretendard for reading and controls, and bundled monospace for code, terminal output and the brand.
- **Do** keep input borders visible at rest and provide the shared keyboard focus outline.
- **Do** preserve the full viewport layout and bounded terminal screen when adapting widths.
- **Do** pair status colors with meaningful labels derived from actual state.
- **Do** honor reduced-motion preferences for CSS animation and transitions.
