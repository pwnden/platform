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

The effective configuration is supplied to Compose on stdin. Generated flags
stay out of temporary configuration files. Live Docker networks are inspected
after startup and before service observation, endpoint access or toolbox
attachment. An older, unisolated environment must be stopped and restarted.

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
before reuse. Problems provide no ingress code or additional external network.

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

This policy protects ordinary container network and mount access. Docker and
its kernel remain trusted; it is not a guarantee against kernel or daemon exploits.
