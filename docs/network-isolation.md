# Problem network isolation

The platform owns a common execution policy for vulnerable services, patches,
command toolboxes, interactive terminals and browser ingress. Authors declare
their services, networks and endpoints through the problem contract.

## Runtime policy

Docker Engine 28 or newer is required. The platform resolves Compose and checks
repository containment and resource ownership before starting resources. It
preserves the complete resolved exercise configuration while applying:

- Project-scoped bridge networks with `internal: true` and both gateway modes
  set to `isolated`. Those two options are the accepted network driver options.
- Service host port mappings removed from the effective configuration.
- All capabilities dropped and `no-new-privileges` enabled for every service,
  toolbox and connector. File toolboxes use network mode `none`.
- Custom security profiles, host namespaces, GPU/device access, extra capabilities
  and privileged lifecycle hooks rejected before startup. The engine's default
  seccomp and security profiles remain active.
- Services, command toolboxes and terminals limited to 2 CPUs, 2 GiB RAM,
  no additional swap and 256 processes. These are per-container ceilings,
  rather than reserved resources. Root filesystems are read-only; `/tmp` is
  a writable 128 MiB tmpfs with execution permitted for compiled exercises.
- Command toolboxes and terminals run as UID/GID `10001:10001` in both modes.
  A separate 64 MiB `/home/pwnden` tmpfs provides their writable home.
  Services retain their authored image/user, so authors should provide a
  non-root service user and declare temporary data mounts for writes.
- Writable problems use a shared 256 MiB tmpfs copy of their directory, capped
  at 32768 inodes. Originals remain unchanged. A networkless keeper retains the
  copy between commands and PTYs; cleanup removes it. Its ceilings are 0.25 CPU,
  512 MiB RAM and 32 processes. Patched environments have separate copies.
- Service binds are read-only. Named volumes use bounded 256 MiB tmpfs with no
  image copy; other tmpfs mounts are capped at 256 MiB and shared memory at 64 MiB.
  Implicit image volumes and multiple service replicas are rejected.
- The `local` logging driver rotates 10 MiB files with a maximum of 3 files.
  Each captured command output stream is limited to 8 MiB; exceeding it fails
  execution and triggers cleanup. Interactive terminal output is streamed.

The effective configuration is supplied to Compose on stdin. Generated flags
stay out of temporary configuration files. Live Docker networks are inspected
after startup and before service observation, endpoint access or toolbox
attachment. Live service limits and policy ownership are checked too. An older
environment must be stopped and restarted to apply the current policy.

Services and tools communicate within their own problem networks. Internet
destinations, host services and other problem networks are blocked. Repository
mount containment remains a separate file access boundary. Images are downloaded
and built during preparation; runtime isolation applies to the started containers.

Docker's ordinary `internal` bridge can still reach its host gateway. The
[`isolated` gateway mode](https://docs.docker.com/engine/network/port-publishing/#gateway-modes)
removes the bridge address and requires an internal network. This mode was added
in [Engine 28](https://docs.docker.com/engine/release-notes/28/).

## Local web access

The Go server listens on a separate IPv4 loopback origin for each declared HTTP
endpoint. Browser requests are proxied through a non-TTY Docker exec stream to
one declared service IP and port. Paths, queries, cookies, form bodies, response
bytes and exercise headers retain the common browser proxy behavior. The player
and problem sites have separate origins. The browser controller and target site
share the problem ingress origin, supporting the existing navigation controls.

The first connection creates one platform-owned connector per problem. It uses
the pinned multi-architecture image
`busybox:1.37.0-musl@sha256:5cec3fc171c87218698e85a52af7087de727372aae264a787b8112901a5b0092`,
runs as `65534:65534` with a read-only root filesystem, and has only the isolated
problem network. It has no host mounts, Docker socket or published ports. The
platform checks an existing connector's ownership and immutable configuration
before reuse. The connector is limited to 0.5 CPU, 128 MiB RAM, no additional
swap and 64 processes, with the same log rotation. Problems provide no ingress
code or additional external network.

The destination comes from the contract and inspected service ownership;
requests supply paths rather than arbitrary destination URLs. Each origin is
bound to a run identity, so restart cannot retarget an old origin. Idle workspace
cleanup, stop and normal server shutdown remove the connector together with its
project. Server shutdown closes owned listeners and streams. Image caches remain.

## Verification boundary

The [actual Docker check](verification.md#actual-network-isolation) covers
Internet IPv4/IPv6 and DNS destinations, host and separate-network positive
controls, same-problem HTTP, command and PTY behavior, web solving, patched
verification and cleanup. Linux/WSL is the current verified execution target;
actual Windows and macOS host checks remain scheduled for a later stage.

On Linux/WSL, Python and Go controllers running as the same OS user share a
per-daemon admission lock in `/tmp`. Container creation reserves actual resource
ceilings before start: by default 8 CPUs, 8 GiB RAM, 1024 processes and 12
containers in total, including connectors and workspace keepers. Docker-reported
CPU/RAM capacity clips those limits. Created and stopped managed containers count
until removal; unrelated applications do not count.

`PWNDEN_CONTAINER_CPUS` selects a per-service, command and terminal CPU ceiling
of 1 or 2 (default 2). This can tighten the policy on a smaller machine; restart
existing problems after changing it. Connectors and workspace keepers retain
their smaller limits.

Positive integer settings `PWNDEN_RUNTIME_CPUS`, `PWNDEN_RUNTIME_MEMORY_MIB`, `PWNDEN_RUNTIME_PIDS` and
`PWNDEN_RUNTIME_CONTAINERS` let the operator set the budget. A rejected creation
leaves current environments running. Controllers on other hosts or OS users do
not share this admission lock. Builds and image caches are outside the runtime
budget. Docker and its kernel remain trusted; this policy does not establish
resistance to kernel or daemon exploits.
