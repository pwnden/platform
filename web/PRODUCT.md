# pwnden

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Players solve locally installed security challenges in their browser. Authors develop and verify their own problems from local checkouts.

## Product Purpose

Provide a local workspace for reading challenges, downloading their files, running isolated services, using a real terminal and submitting flags.

## Operating Context

Go serves the embedded Vue player and HTTP API. Docker is preinstalled. Players run `./pwnden setup` and `./pwnden serve`; authors run `./pwnden dev` with a sibling challenges checkout. The server's full session URL opens the player.

## Capabilities and Constraints

The challenges repository owns the versioned problem contract. Problem host mounts and build inputs stay inside the allowed problem root. The pnpm workspace separates pure TypeScript domains, features, the API transport and UI. Only `packages/ui` consumes Sectile and xterm. Dependencies use exact versions.

## Brand Commitments

The name is `pwnden`. The player uses a hacker terminal visual language, monospace typography, subtly luminous dark blue on blue-tinted black, and a workspace that fills the browser viewport. Korean UI labels and real problem content remain available.

## Evidence on Hand

The local challenges checkout contains `rotor-lock` and `note-vault`. The player already supports list, detail, download, run, stop, status, flag submission and interactive terminals. Demonstration content used in visual verification is synthetic and stays separate from player data.
