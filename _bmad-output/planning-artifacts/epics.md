---
stepsCompleted: ["validate-prerequisites"]
inputDocuments:
  - _bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/prd.md
  - _bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/addendum.md
  - _bmad-output/planning-artifacts/architecture/architecture-app-ovsingen-2026-08-30/ARCHITECTURE-SPINE.md
---

# THW OV Singen App - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for THW OV Singen App, decomposing the requirements from the PRD, UX Design if it exists, and Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

FR-1: Email-Based Authentication — any user authenticates with registered email + secure password; correct credentials return a secure session token, incorrect return HTTP 401.
FR-2: Password Policy — minimum 10 characters, no character-composition complexity rules; shorter passwords rejected with clear validation error.
FR-3: Progressive Login Lockout — after 3 consecutive failed logins the account is blocked for 30s (HTTP 429); after 4, for 60s.
FR-4: Multi-Factor Authentication (MFA) — optional TOTP via authenticator app; when enabled login requires a valid 6-digit code after password; invalid/expired codes rejected.
FR-5: Self-Registration and Admin Approval — new volunteers self-register; account created in `pending_approval` and blocked from auth; only Admin can approve to `active`.
FR-6: Group & Permission Management — admins assign users to User Groups, map permissions/Permission Groups (e.g. Helfer*in, Fuehrung, Admin) to users/groups; access controlled by resolved active permission set of the logged-in user.
FR-7: Flexible User Attributes — dynamic metadata attributes on user profiles via JSON field without DB migration.
FR-8: Tool Type Management — create/edit Tool Types defining name, default Inspection Schedule (from named schedule catalog FR-30), required Qualification, inspection mode (pass/fail vs checklist); mode persists and controls inspection UI.
FR-9: Tool Management — create/edit individual Tools belonging to a Tool Type, optional per-tool schedule overriding the type default; bulk CSV import with per-row error report without partial silent failures.
FR-10: Flexible Attributes on Tools & Tool Types — arbitrary custom metadata via JSON field without DB migration.
FR-11: Qualification-Gated Inspection — Helfer*in and Fuehrung may submit an inspection only if they hold the Qualification required by the Tool Type; otherwise a clear access error.
FR-12: Checklist Inspection — checklist-mode Tool Types present configured items; all items must be answered before submit; per-item pass/fail persisted and visible in history.
FR-13: Pass/Fail Inspection — pass/fail-mode Tool Types present a simple pass/fail toggle with optional notes field.
FR-14: Out of Service Flagging — any failed item or failed pass/fail result transitions the Tool to Out of Service immediately; failure event records inspector, timestamp, failed item(s), defect notes; OOS shows Red on dashboard.
FR-15: Out of Service Reinstatement — Fuehrung/Admin may clear OOS with mandatory non-empty reason; clock resets (next due = reinstatement + schedule); resulting color derived immediately; reinstatement logged with actor/timestamp/reason; Helfer*in cannot reinstate.
FR-16: Color-Coded Status Dashboard — all authenticated users view every Tool color-coded: Red = past due or OOS, Orange = due within next 14 calendar days, Green = current (due > 14 days); filterable by status.
FR-17: Status Report Export (PDF) — Fuehrung/Admin export PDF of currently visible/filtered tool list incl. tool name, type, status, last inspection date, last inspector.
FR-18: Inspection History — full history per Tool stored and accessible to Fuehrung/Admin; per-inspection: inspector, timestamp, overall result, mode, checklist item results; reverse-chronological display.
FR-19: Admin Module Access Isolation — admin routes return HTTP 403 for non-Admin users; admin module existence hidden (no links/menu entries/UI hints).
FR-20: User Approval Workflow — admins view `pending_approval` users, approve (→ `active`, allows auth) or reject (removes pending record).
FR-21: User & Group Administration — admins create/edit/deactivate user accounts, create/edit User Groups, assign users to groups; deactivated accounts cannot authenticate; removing a user from a group revokes inherited permissions immediately.
FR-22: Qualification Management — admins create Qualifications, assign/revoke on users; assignment immediately grants inspect right, revocation immediately removes it.
FR-23: Tool & Tool Type Configuration — admin exposes FR-8/FR-9 incl. checklist-item definition per Tool Type and CSV import; added/removed checklist items reflect on future inspections, historical records unchanged.
FR-24: DSGVO Compliance Operations — admin generates data-access report (profile fields, login history, qualifications) and executes account deletion; deletion purges personal data, retains inspection history anonymized to "Deleted User", account cannot be re-activated.
FR-25: Change Own Password (Authenticated) — any logged-in user changes own password after confirming current password; new password ≥ 10 chars (FR-2); successful change revokes all other active sessions; audited (NFR-O1); never stored/logged in plaintext.
FR-26: Self-Service Password Reset (Forgot Password) — login offers "Forgot password?" accepting email; if `active` account, sends single transactional email with single-use, 30-min, hashed reset link/token; uniform response for unknown email (no enumeration); multiple requests invalidate earlier tokens; audited.
FR-27: Admin Credential Recovery (Dual-Control) — exactly two pre-seeded admin accounts (out-of-band credentials, never in VCS); locked-out admin cannot self-reset via FR-26 alone; recovery requires second admin's approval (`admin.recovery.approve`), recorded as high-severity immutable audit event; last-admin recovery is out-of-band documented manual procedure with mandatory audit; FR-2 applies.
FR-28: SMTP Configuration (Admin Interface) — admin panel exposes email-settings surface (host, port, security none/STARTTLS/TLS, sender/display name, username); SMTP password encrypted at rest and write-only/masked; saves migrate-backed and apply without redeploy; "send test email" action; failures logged not silent; gated by `admin.settings.email`.
FR-29: Backup Destination Configuration (Admin Interface) — admin panel exposes backup-destination surface (≥1 destination required; mechanism type S3-compatible/FTP·SFTP/local, endpoint/host, bucket/path, credentials, optional schedule); credentials encrypted at rest, write-only/masked; "test connection" action; destinations used by backup job without redeploy; failures logged/surfaced; gated by `admin.settings.backup`.
FR-30: Schedule Catalog Management (Admin Interface) — admin panel exposes schedule-catalog surface (create/edit/archive named schedules: name + repeating interval unit/magnitude); rows carry reserved weekday-set + time-of-day fields nullable/unused in V1 (cron-like) so a future composite engine needs no migration; Tool Types pick default, Tools pick individual schedule by FK; archiving does not delete tool/tool-type history; gated by `schedules.manage`.

### NonFunctional Requirements

NFR-S1: Transport Security — all client-server communication TLS 1.2+; plain HTTP rejected or redirected.
NFR-S2: Authentication Hardening — session tokens cryptographically signed, expire after configurable idle period (default 8h), invalidated server-side on logout.
NFR-S3: Authorization Enforcement — permission checks enforced server-side on every request; frontend visibility controls supplementary only.
NFR-S4: Secrets Management — no credentials/API keys/secrets in source code or VCS; secrets injected via env vars or secrets manager at runtime.
NFR-S5: Dependency Security — third-party dependencies auditable and updated for known CVEs within 30 days of disclosure.
NFR-P1: Response Time — API responses (dashboard load, inspection submission, status updates) within 500ms at 95th percentile at ≤50 concurrent users.
NFR-P2: Mobile Usability — frontend fully functional on iOS Safari and Android Chrome without native app; key workflows usable on 4G.
NFR-R1: Containerization — backend, frontend, database deployable as Docker containers via a single docker-compose configuration.
NFR-R2: Database Migrations — all schema changes via versioned, incremental migration files; forward migrations supported; rollback documented.
NFR-R3: Backup & Restore — configurable automated backups of DB and media/file storage to ≥1 destination; documented and tested restore procedure from initial deployment.
NFR-R4: Availability — 99% monthly target, excluding scheduled maintenance windows.
NFR-M1: Modularity — feature modules decoupled at codebase level; new module integrated without modifying existing module business logic.
NFR-M2: Test Coverage — all FRs have corresponding automated tests (unit and/or integration); CI fails on test failure.
NFR-M3: Linting & Code Style — all code passes linter checks in CI; PRs failing lint not merged.
NFR-M4: Documentation — public API contracts and module integration points documented in the docs site; kept current with each release.
NFR-O1: Structured Logging — backend emits JSON structured logs for auth events, permission denials, inspection submissions, OOS transitions, reinstatements, DSGVO operations.
NFR-O2: Audit Trail — all DSGVO-relevant operations (account deletion, data-access report) produce immutable audit log entry with actor, timestamp, operation type.
NFR-C1: DSGVO / GDPR — data minimization; right to access (FR-24 report) and erasure (FR-24 deletion); compliance first-class.
NFR-PL1: Browser Support — two most recent major versions of Chrome, Firefox, Safari, Edge.
NFR-PL2: Form Factor — responsive web app for mobile, tablet, desktop; no native app for V1.

### Additional Requirements

- Greenfield build; no starter template specified — Epic 1 Story 1 is the cold-start scaffold (structural seed below).
- AD-1: Modular monolith, hexagonally partitioned per module — each module an isolated Go package exposing only port interfaces; adapters (HTTP, DB) wired only at the composition root (`cmd/server`); no cross-module import of adapters/handlers/repositories; no in-memory shared cross-module state.
- AD-2: User Directory & Auth is single owner of identity, users, groups, permissions, sessions, MFA/TOTP, credentials, recovery tokens, account state, qualifications; exposes one `Service` port (identity + granted qualifications + resolved permission set); revocation takes effect immediately per request; approved users seeded with `helfende`.
- AD-3: First-class typed columns for core attributes + one `attributes JSONB` column on Users, Tools, Tool Types; JSONB is the no-migration extension surface; promotion path via golang-migrate + backfill.
- AD-4: Tool status (Red/Orange/Green/OOS) always derived on read (inspection records + due-date math), never stored; OOS derived from latest failed inspection/checklist item not since reinstated; reinstatement sole exit from OOS; never-inspected tool shown Red.
- AD-5: Single shared inspection clock — `next_due = last_successful_inspection + resolved schedule interval` (per-tool `schedule_id` else tool-type default); reinstatement resets clock; thresholds Red=past due/OOS, Orange=≤14 days, Green=>14 days; single shared function all consumers call.
- AD-6: Server-side authorization is only source of truth — every action maps to exactly one permission code; server re-validates on each request (HTTP 403); admin existence hidden in SPA.
- AD-7: Qualification gating — qualification data owned by User module; Tool module obtains required + granted qualifications via auth port; assign/revoke effective immediately.
- AD-8: Cross-module user lifecycle & DSGVO flows — composition-root orchestration calls each owning module through its exports; deletion anonymizes inspection history (`"Deleted User"`); runs in one transaction; immutable audit entry.
- AD-9: OOS reinstatement Fuehrung/Admin only; mandatory reason; sole exit from OOS; clock reset.
- AD-10: Tool/Tool-Type config write path single owner (Tool module) — Admin configures via Tool module's configuration port; per-tool schedule is a first-class FK to `schedules`, never JSONB.
- AD-11: Cross-module schema/reference ownership — one golang-migrate migration set; every table has exactly one owning module; cross-module refs are FKs read via owning module's port; Admin owns exactly 3 config tables (`smtp_settings`, `backup_destinations`, `schedules`).
- AD-12: Action-matched, additive permission model — every action ↔ exactly one permission code; resolved set = additive union of permission-group memberships + direct grants; no deny permissions; roles are named permission groups; base roles `helfende`, `schirrmeister`, `fuehrende`, `admin`; admins create/edit named groups and may edit base roles; flat (no groups-in-groups in V1); user groups are organisational only (grant no access).
- AD-13: Self-service password management + dual-admin recovery — FR-25/26/27 flows in User module through `Service` port; Argon2id hashes; single-use 30-min hashed reset tokens; two seeded admin accounts; `admin.recovery.approve` gated dual-control; last-admin recovery out-of-band manual.
- AD-14: Admin-owned SMTP settings (`smtp_settings`, `admin.settings.email`) — password encrypted at rest, write-only/masked; User email adapter consumes via Admin settings port; test-email action.
- AD-15: Admin-owned backup destinations (`backup_destinations`, `admin.settings.backup`) — ≥1 required; mechanism type, endpoint, bucket/path, encrypted credentials; test-connection action; consumed by backup job via settings port.
- AD-16: Named schedule catalog (`schedules`, `schedules.manage`) — name + interval_unit/interval_magnitude in V1; reserved weekday-set + time-of-day fields nullable/unused; Tool Types/Tools reference by FK via Admin schedule port; lenient validation.
- Stack: Go 1.27, chi v5.3.2 (>=v5.2.4 to avoid GO-2026-4316 open-redirect), sqlc v1.31.1, pgx v5, golang-migrate v4.19.1, PostgreSQL 18, TypeScript 6.x, React 19, Vite 8.x, Docker/podman-compose (single Compose source), just (single command runner), OpenTofu v1.12.6 Google provider 5.x (Cloud Run, Artifact Registry, Cloud SQL; min_instance_count=0), Docusaurus docs.
- Structural seed: `cmd/server/` composition root; `internal/{user,tools,admin,platform}/`; `web/` React+Vite+TS SPA (no business logic); `migrations/`; `deploy/` compose + backup; `infra/` OpenTofu; root `justfile`; `docs/`.
- Database: single migration set, 22 tables (1-11, 19 owned by User; 12-18 by Tool; 20-22 by Admin); UUID v7 PKs; UTC RFC 3339 timestamps; uniform JSON error envelope `{"error":{"code","message","details?"}}`.
- Base permission series (21 codes): `dashboard.view`, `inspection.submit`, `inspection.history.view`, `report.export`, `tool.reinstate`, `tools.manage`, `tool_types.manage`, `users.view`, `users.approve`, `users.manage`, `user_groups.manage`, `roles.create`, `roles.edit`, `roles.assign`, `qualifications.manage`, `dsgvo.access_report`, `dsgvo.delete`, `admin.recovery.approve`, `admin.settings.email`, `admin.settings.backup`, `schedules.manage`.
- Base role matrix (additive): `helfende` = dashboard.view, inspection.submit; `schirrmeister` = + tools.manage, tool_types.manage, inspection.history.view; `fuehrende` = + inspection.history.view, report.export, tool.reinstate; `admin` = all 21 codes.
- Deferred to build epics (not blocking stories): frontend framework internals (router/state/component library) chosen in frontend epic; operational envelope/deploy topology details and CI/CD pipeline provider+workflow files in deploy epic; concrete TOTP and SMTP libraries in User epic; composite schedule engine (weekday/time) deferred to future enhancement (schema fields exist).
- User journeys UJ-1 (Joshi chainsaw inspection), UJ-2 (Nico readiness audit + PDF export), UJ-3 (Saskia approves helper + qualification), UJ-4 (Torsten catalogue maintenance + history check) from PRD §2.3 bind to Epic 2 and Epic 3 stories.

### UX Design Requirements

UX-DR: No separate UX design contract exists. UX guidance comes from the PRD (mobile-first responsive web, NFR-P2/PL2; UJ-1..UJ-4; traffic-light status dashboard) and architecture (SPA with server-side gating, admin existence hidden from non-admins). Story-level UX details (router, component library, styling) are deferred to the frontend story within Epic 1 (structural seed).

### FR Coverage Map

{{requirements_coverage_map}}

## Epic List

{{epics_list}}

<!-- Repeat for each epic in epics_list (N = 1, 2, 3...) -->

## Epic {{N}}: {{epic_title_N}}

{{epic_goal_N}}

<!-- Repeat for each story (M = 1, 2, 3...) within epic N -->

### Story {{N}}.{{M}}: {{story_title_N_M}}

As a {{user_type}},
I want {{capability}},
So that {{value_benefit}}.

**Acceptance Criteria:**

<!-- for each AC on this story -->

**Given** {{precondition}}
**When** {{action}}
**Then** {{expected_outcome}}
**And** {{additional_criteria}}

<!-- End story repeat -->