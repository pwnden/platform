# pwnden

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Players solve locally installed security challenges in their browser. Authors develop and verify their own problems from local checkouts.

## Product Purpose

Provide a local browser workspace where players understand a challenge's objective and context, inspect materials, use target services and a prepared terminal, submit flags, and learn through progressive hints and complete explanations. Every solving and learning step is available from the website; downloads are optional.

## Operating Context

Go serves the embedded Vue player and HTTP API. Docker is preinstalled. Players run `./pwnden setup` and `./pwnden serve`; authors run `./pwnden dev` with a sibling challenges checkout. The server's full session URL opens the player.

## Capabilities and Constraints

The challenges repository owns the versioned problem contract and player content. Briefs are written for solvers using actual UI controls. Hints open independently; complete explanations are labeled as containing answers and fetched only on request. Maintainer installation and automated verification instructions live in author documentation. Player commands run in the prepared website terminal. Problem host mounts and build inputs stay inside the allowed problem root. The pnpm workspace separates pure TypeScript domains, features, the API transport and UI. Only `packages/ui` consumes Sectile and xterm. Dependencies use exact versions.

## Brand Commitments

The name is `pwnden`. The player uses a hacker terminal visual language, monospace typography, subtly luminous dark blue on blue-tinted black, and a workspace that fills the browser viewport. Korean UI labels and real problem content remain available.

## Evidence on Hand

The local challenges checkout contains `rotor-lock` and `note-vault`, each with a player brief, three progressive hints and a complete explanation. The player supports list, detail, source preview, optional download, hints, walkthrough, run, stop, status, flag submission and interactive terminals. Demonstration content used in visual verification is synthetic and stays separate from player data.
