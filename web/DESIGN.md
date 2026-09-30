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
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "clamp(1.8rem, 3vw, 3rem)"
    fontWeight: 400
    lineHeight: 1.65
  headline:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "1.2rem"
    fontWeight: 500
    lineHeight: 1.65
  title:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "0.9rem"
    fontWeight: 500
    lineHeight: 1.65
  body:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "14px"
    fontWeight: 400
    lineHeight: 1.65
  label:
    fontFamily: "'JetBrains Mono', 'D2Coding', monospace"
    fontSize: "0.85rem"
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

Blue-tinted black surfaces, restrained blue light and monospace text give the player a working terminal character. Dense controls and readable Korean labels support extended challenge sessions. The workspace fills the browser viewport, with reading and execution surfaces divided by thin borders.

**Key Characteristics:**

- Subtly luminous dark blue on blue-tinted black.
- One monospace stack for Latin, Korean and terminal output.
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

**Font stack:** JetBrains Mono, D2Coding, monospace. Bundled JetBrains Mono supplies Latin glyphs; D2Coding supplies Korean glyphs. Both are served locally with `font-display: swap`; license files and source revisions are recorded in [fonts/README.md](packages/ui/src/fonts/README.md).

The frontmatter's display role is the empty workspace prompt; headline is the problem title; title is the shared panel heading. Body sets the root scale, and label covers field and supporting labels. Smaller status and footnote text use the same stack. Headings and problem titles use medium weight. Numeric text uses tabular figures; ligatures are disabled.

Problem descriptions render Markdown, wrap long content, use a reading measure of (75ch) and line height of (1.85). Code preserves literal whitespace inside a scrollable block. The real terminal uses (14px) type and line height of (1.4). Text inputs use (16px) at the narrow layout breakpoint.

**The Shared Glyph Rule.** Keep Latin and Korean content in the bundled monospace stack, including controls and terminal output.

## Layout

The player occupies (100dvh), with a minimum height of (38rem), between a compact header and footer. The desktop sidebar is (17rem). The remaining area divides into reading and terminal columns: `minmax(21rem, 0.9fr)` and `minmax(0, 1.2fr)`. Sidebar and reading content scroll within their bounds.

At (76rem), the sidebar becomes (15rem) and the terminal follows the reading column, with a minimum terminal panel height of (32rem). At (48rem), the sidebar moves above the content with a maximum height of (18rem), the page uses document scrolling, and panel padding changes from `ui-space-3` to `ui-space-2`. At (30rem), the flag field and submit button stack. The document supports widths from (320px).

Shared panel bodies use `ui-space-3`; terminal controls and output use `ui-space-2`. Control groups use `ui-space-1`. Panel headings wrap their actions as needed. Long names, URLs and descriptions wrap within their columns.

**The Bounded Terminal Rule.** Keep terminal measurement inside a positioned, bounded viewport. The terminal screen has a minimum height of (18rem), with its renderer positioned within that screen.

Source: [App.vue](apps/player/src/App.vue), [UIPanel.vue](packages/ui/src/UIPanel.vue), [PlayPanel.vue](features/play/src/PlayPanel.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

## Elevation & Depth

Depth comes from tonal surfaces and dividing strokes. The shared blue glow is reserved for primary button hover, selected rows, focused fields, info status dots and terminal-style cursor marks. Its exact value is recorded in the sidecar. Keyboard focus uses a (2px) accent outline with a (3px) offset; inputs bring that outline to their edge.

Control color, border and shadow transitions take (140ms). The empty workspace cursor pulses over (1.6s), while the brand cursor remains steady. Reduced-motion preferences remove CSS transitions and animations. The xterm renderer separately enables its blinking cursor.

## Shapes

Controls have small, nearly square corners using `rounded.control`. Panels meet through straight borders. Status markers are circular dots of (0.375rem). Terminal prompts and cursors use simple text and rectangular blocks.

## Components

### Buttons

The shared button has a minimum height of (2.75rem), a thin border and line height of (1.4). Primary, secondary, ghost, danger and row variants use the assignments in the frontmatter. Enabled hover moves to the hover surface and active stroke; primary hover adds the glow. Compact buttons use a minimum height of (2.5rem), horizontal padding of (0.65rem) and label size of (0.85rem). Disabled or busy buttons are disabled, muted and rendered at (0.5) opacity; busy controls expose `aria-busy`.

Source: [UIButton.vue](packages/ui/src/UIButton.vue), [theme.css](packages/ui/src/theme.css).

### Inputs / Fields

An associated label precedes a full-width field. The resting field uses the active blue stroke, a dark canvas and an accent caret. Focus adds the outline and shared glow; placeholder text is muted. Disabled fields use (0.5) opacity. `UITextField` wraps Sectile through the public UI package.

Source: [UITextField.vue](packages/ui/src/UITextField.vue).

### Panels

Each panel is a semantic section with a labelled heading and optional actions. A lower heading border separates the content; the header has a minimum height of (3.5rem). Panels flex to fill available space while permitting their contents to shrink. The terminal panel uses the terminal background.

Source: [UIPanel.vue](packages/ui/src/UIPanel.vue).

### Navigation

Problem rows are full-width buttons with a title, slug and category/type metadata. `aria-pressed` identifies the current selection; its accent fill, stroke and glow provide the visual state. Selecting the current row preserves its terminal session. The responsive sidebar behavior follows Layout.

Source: [ProblemList.vue](features/catalog/src/ProblemList.vue), [App.vue](apps/player/src/App.vue).

### Status

Inline text and a small dot communicate muted, info, success or danger tone. Labels describe actual client, execution or terminal state. Info dots receive the shared glow; alert text uses danger. Supporting status text uses (0.8rem).

Source: [UIStatus.vue](packages/ui/src/UIStatus.vue).

### Terminal

`UITerminal` owns xterm and its fit addon inside the UI package. It maps shared tokens to the terminal theme, enables screen reader mode and resizes after fonts load. The surrounding panel supplies connection controls and real connection status. A centered prompt explains the closed state, and a footnote explains session cleanup.

Source: [UITerminal.vue](packages/ui/src/UITerminal.vue), [TerminalPanel.vue](features/terminal/src/TerminalPanel.vue).

### Markdown

`UIMarkdown` renders source through md4x's WASM parser and an allowlisted Vue AST renderer. Source headings fit below the problem description heading. Inline code uses the accent and raised surface; code blocks use the terminal background. Tables scroll inside their own region. Quotes use a thin active stroke, and task markers are read-only. Links use the shared accent and keyboard focus. Images display alternative text. Loading is announced; failure offers retry and escaped source.

Source: [UIMarkdown.vue](packages/ui/src/UIMarkdown.vue), [markdown.ts](packages/ui/src/markdown.ts).

## Do's and Don'ts

### Do:

- **Do** use shared color, spacing and corner tokens for controls and panels.
- **Do** keep the bundled Latin and Korean monospace stack throughout the workspace.
- **Do** keep input borders visible at rest and provide the shared keyboard focus outline.
- **Do** preserve the full viewport layout and bounded terminal screen when adapting widths.
- **Do** pair status colors with meaningful labels derived from actual state.
- **Do** honor reduced-motion preferences for CSS animation and transitions.
