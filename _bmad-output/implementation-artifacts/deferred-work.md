# Deferred Work Ledger

Triage output of review loops — real, non-story-blocking findings that are not caused by (or are deliberately out of scope for) the current story. Each entry records why it is real and where it should be picked up. This file is append-only; do not modify or delete existing entries.

- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: SPA (`web/`) cannot call the Go API in `just dev` — no Vite dev proxy and no CORS middleware, so browser fetches between :5173 and :8080 are cross-origin blocked.
  evidence: Story 1.1 only requires API+SPA+DB all run and `/healthz` proves DB liveness; no in-SPA consumer action exists yet (Spend: defer directly from the Story 1.1 review; the first real cross-origin call arrives with Story 1.2 dashboard mount / 1.3 registration).
  recommended: add `server.proxy` (e.g. `/api` → `http://localhost:8080`) in `web/vite.config.ts` when the first SPA→API call ships.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: `web/` has no lint/test gate — `just lint`/`just test` only cover Go; no ESLint configuration, no npm lint/test scripts, and the TypeScript layer is not part of the single-command quality gates.
  evidence: Story 1.1 ships `web/` as a minimal Vite+React+TS shell with no business logic; the frontend lint/test pipeline belongs with the first real frontend story (1.2 dashboard foundation).
  recommended: introduce ESLint + `npm run lint`/`test` and wire into `justfile` gates alongside the Go gates when frontend work begins.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: `updated_at` columns are never maintained — tables default `now()` but there is no `set_updated_at()` trigger and no app-layer obligation, so `updated_at` stays frozen at insert time.
  evidence: Story 1.1 performs no UPDATEs (seed is insert-only); the first row-updating story (1.3 self-registration creates accounts; admin approval later flips state) introduces the maintenance mechanism.
  recommended: add a `set_updated_at` trigger (or document app-layer obligation) in the first story that writes to `users`.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: No production serving path for the SPA — the Go binary never serves `web/dist`; only Vite dev serves the frontend.
  evidence: Story 1.1 AC requires `just dev` (dev-time Vite) serve the SPA; production serving is deploy-epic scope.
  recommended: build+Serve `web/dist` from the Go binary (or a static host) in the deploy/infra story; validate before release.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: The cold-start migration seed rows (4 base roles, 2 admin accounts, `admin.recovery.approve`) are not asserted by any automated test — only by the spec's manual-check bullet and live `just migrate-up/down` verification.
  evidence: A DB-backed test would pin the seed contents against AD-12/AD-13 but requires a running podman PostgreSQL at test time (would couple `go test ./...` to a live DB).
  recommended: add a `//go:build integration` seeded-rows test (run the migration, assert role/permission/admin rows) gated behind a `just test-integration` recipe when the CI/deploy story lands.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: The composition root (`cmd/server/main.go`) — /healthz mount, pool-error exit, graceful-shutdown — has zero automated coverage.
  evidence: wired routing is now pinned by `internal/platform/router` tests; remaining main.go paths (config→pool→shutdown) are verified only by manual `just dev` + curl in the spec's Verification.
  recommended: extract remaining composition-root seams into testable handlers/constructors, or cover with an integration test, in the CI/integration story.
- source_spec: `_bmad-output/implementation-artifacts/spec-1-1-project-scaffold-database-foundation.md`
  summary: No Go CI pipeline — `.github/workflows/` ships only `docs.yml`; `go build/vet/test/lint` run only when a human invokes the justfile.
  evidence: Story 1.1 AC does not require CI; pipelines belong to the deploy/ops epic (OpenTofu/Cloud Run are already planned).
  recommended: add a `ci.yml` running `just build`, `just vet`, `just test`, `just lint` (+ docs build) in the deploy/CI story.