# Durable player state

The platform owns `progress.db`, a SQLite database at
`filepath.Join(os.UserConfigDir(), "pwnden", "progress.db")`. On Linux/WSL this
normally resolves to `$XDG_CONFIG_HOME/pwnden/progress.db` or
`~/.config/pwnden/progress.db`; Windows and macOS use Go's OS configuration path.
The database parent is created with mode 0700 and the database with mode 0600
where Unix permissions apply. The platform opens it; problem services and
toolboxes receive no mount of it.

## Responsibilities

- `internal/progress` defines the completion model and storage port.
- `internal/storage` implements SQLite with `modernc.org/sqlite v1.60.1`.
- `internal/application` validates submissions and calls the port.
- `internal/cli` opens, injects and closes storage for `serve`, `dev` and `submit`.
- The authenticated HTTP adapter and Vue player expose the saved result.

The driver retains `CGO_ENABLED=0` packaging. A separate database server or host
SQLite installation is unnecessary. Future state adds explicit models, ports
and transactional migrations in this storage layer when its capability is used.

## Completion identity and lifetime

Only accepted submissions are stored: namespace, stable problem slug, accepted
input and first completion time. Repeated accepted submissions update the input
and retain the first time. Incorrect submissions leave the record unchanged.
The saved input restores the readonly answer field; it is local player data and
is stored as text. Catalog lists expose only completion times.

The official namespace persists across catalog revisions and installation
directories. Development namespaces derive from the canonical checkout path,
keeping local catalogs separate from official progress and from each other.
Problem slugs identify the same exercise across revisions.

Problem changes, page reloads, server restarts, container resets and catalog
updates retain the database. Container lifecycle state and browser session
credentials keep their existing ownership and lifetimes.

## Migrations and failures

Schema version 1 uses SQLite `user_version`. Migration runs transactionally and
retains existing records. A newer unsupported schema is rejected. WAL and a
five-second busy timeout coordinate access; each store uses one connection.
A correct answer is reported as accepted after its write succeeds. A failed
read or write reports `storage_failed` rather than presenting unsaved success.

Tests cover reopen, first-time preservation, namespace isolation, invalid state,
newer-schema rejection with records retained, catalog path changes, rejected
answers and storage failure. The UI restores accepted text without submitting
it again.
