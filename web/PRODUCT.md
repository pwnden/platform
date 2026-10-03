# pwnden

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Players solve locally installed security challenges in their browser. Authors develop and verify their own problems from local checkouts.

## Product Purpose

Provide a local browser workspace where players understand a challenge's objective and context, use its required tools, submit flags, and learn through progressive hints and complete explanations. Every solving and learning step is available from the website; downloads are optional. The problem's contract declares its tools and order; the first tool opens by default. Tool switches preserve ongoing work. Problem presence prepares and retains the environment independently of creating a terminal shell.

## Operating Context

Go serves the embedded Vue player and HTTP API. Docker is preinstalled. Players run `./pwnden setup` and `./pwnden serve`; authors run `./pwnden dev` with a sibling challenges checkout. The server's full session URL opens the player.

## Capabilities and Constraints

The challenges repository owns the versioned problem contract and player content. Briefs are written for solvers using actual UI controls. Hints open independently; complete explanations are labeled as containing answers and fetched only on request. Maintainer installation and automated verification instructions live in author documentation. Player commands run in the prepared website terminal. Problem host mounts and build inputs stay inside the allowed problem root. The pnpm workspace separates pure TypeScript domains, features, the API transport and UI. Only `packages/ui` consumes Sectile and xterm. Dependencies use exact versions.

## Brand Commitments

The name is `pwnden`. The player uses a hacker terminal visual language, subtly luminous dark blue on blue-tinted black, and a workspace that fills the browser viewport. Pretendard supplies readable Korean body text and controls; code, terminal output and the brand use the bundled monospace stack. The catalog supports keyword search, readable categories, five difficulty levels, difficulty filtering and sorting within categories, and bounded pages. Difficulty labels are Intro, Easy, Medium, Hard and Expert; badges use quartz, emerald, sapphire, amethyst and ruby colors. The selected problem header stays visible while its content scrolls. Section headings identify the brief, hints and walkthrough. Submission stays below the tool viewport; an accepted flag remains readonly and selectable, with a check icon and success color. Feedback occupies constant slots or viewport overlays, preserving surrounding layout dimensions. Copy describes the goal and required actions directly.

## Evidence on Hand

The local challenges checkout contains `rotor-lock` and `note-vault`, each with a player brief, three progressive hints and a complete explanation. The player supports list, detail, automatic source preview, optional download, hints, walkthrough, automatic environment preparation and cleanup, status, flag submission and interactive terminals. The selected tool's icon actions share the common tab strip. Demonstration content used in visual verification is synthetic and stays separate from player data.

Selected problems expose prerequisite reading and connections to earlier, later
and related exercises from declared concepts. Every problem remains directly
accessible. Teaching objectives start collapsed because labels may reveal
solving principles.
