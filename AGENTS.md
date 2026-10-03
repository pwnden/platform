# Naming

Project-owned abbreviations use whole lowercase or whole uppercase in names,
paths and prose. Examples: `api`, `APIClient`, `createAPI`, `HTTPError`,
`UIButton`, `url`, `URL`, `DTO`, `CLI`. Preserve upstream names when calling
third-party APIs; give project-owned aliases the project spelling.

# Delivery validation

For changes to player file exposure, terminal behavior, installation, packaging
or HTTP delivery, run `python3 -B tools/verify_delivery.py --challenges ../challenges`
before publishing or marking the work complete. This is the same native package,
HTTP and PTY gate used by CI. `tools/verify.py` checks catalog execution and is
not sufficient for those delivery changes. Player `exec` and terminal smoke
checks use declared exercise files; trusted `verify` supplies submission fixtures.

# Challenge integration

Discover problems and derive player tools from the challenges contract. Keep
the `[player].tools` order and use its first tool as the default. Problem presence
prepares and retains the environment independently of creating a terminal shell.
Keep shared player capabilities in the platform so adding a problem requires its
declarations and exercise resources. Keep workspace usage guidance in common
platform documentation and UI; author content describes the exercise. Preserve
problem HTTP behavior and the separate origins of the player and target sites
when extending embedded web tools.

Common web navigation belongs to the platform browsing proxy, controller and UI.
Derive it from declared HTTP endpoints using the existing problem declarations.
Keep current-page reload, browser history and address editing together in the
shared web tool's address bar. Compose terminal refresh, file download and the
web new-tab action in the common tab strip with shared controls.
The web address field displays page paths and resolves input within the current
problem origin. Unusable input restores the current page path without navigation.

Submission stays below the tool viewport. Accepted values remain readonly and
selectable, with success styling. State feedback uses fixed slots or viewport
overlays so surrounding layout retains its dimensions.
