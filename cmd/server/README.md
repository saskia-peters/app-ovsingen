# cmd/server

**Composition root (AD-1).** This package is the only place that wires module
hexagons and their adapters together and mounts the HTTP surface. Handlers,
adapters and repositories contain no business logic — they delegate to the
modules. The auth gateway and server-side policy (AD-2/AD-6) attach here in
later stories.

Run: `just dev` (or `go run ./cmd/server`). Configuration via `GEAR_*` env
vars (see `internal/platform/config`).
