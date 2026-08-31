---
stepsCompleted: ["validate-prerequisites", "design-epics"]
inputDocuments:
  - _bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/prd.md
  - _bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/addendum.md
  - _bmad-output/planning-artifacts/architecture/architecture-app-ovsingen-2026-08-30/ARCHITECTURE-SPINE.md
  - _bmad-output/planning-artifacts/ux-designs/ux-app-ovsingen-2026-08-30/DESIGN.md
  - _bmad-output/planning-artifacts/ux-designs/ux-app-ovsingen-2026-08-30/EXPERIENCE.md
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

UX-DR1: Design token foundation — implement the full color system from DESIGN.md tokens (brand: THW-blau `#003399`, navy `#001a66`, navy-deep `#00124d`, orange `#f5821f`, plus dark-mode variants; status traffic-light green `#2e7d32`, orange `#f5821f`, red `#c62828`, OOS `#1c1c1c` and dark variants; surface-base/raised/overlay; ink-primary/secondary/disabled/on-brand/on-orange; border-hairline — each with `-dark` pair) as consumable CSS custom properties/tokens consumed by the SPA.
UX-DR2: Typography ramp — five roles (display 28/700, title 20/600, body 16/400, meta 13/400, label 14/500, Inter/system-ui, per DESIGN.md.typography) implemented as reusable tokens; all support dynamic scaling and 200% zoom without layout breakage; German compound nouns must not truncate or clip.
UX-DR3: Spacing / rounded / elevation scales — 4-base spacing scale (4..48px, gutter 16px, margin-mobile 16px), rounded scale (sm 4 / md 8 / lg 12 / xl 16 / full 9999), and tonal elevation (no shadows by default; soft shadow only on modal/confirmation overlay `0 4px 12px`) applied consistently; inspection checklist always single-column.
UX-DR4: Light + dark mode token pairs ship in V1 — every DESIGN.md color/component token has a `-dark` counterpart; theme toggles per DESIGN.md; user can select and system persists it.
UX-DR5: Reusable UI component library — implement as reusable, testable components (per DESIGN.md.Components): (1) Button-primary (THW-blau fill, white text), (2) Button-danger (status-red, white text, irreducible/high-stakes actions only), (3) Status chip (green/orange/red/OOS pills, distinct token per color), (4) Pass/Fail chip (large ≥48px tappable, green OK/BESTANDEN vs red FEHLER/NICHT BESTANDEN), (5) Input field (underline style, floating label, blue focus / red error underline), (6) Filter chip (pill, active THW-blau fill, one-tap, multi-active), (7) Summary count (display number in colored block, tap activates filter), (8) Card (surface-raised, no shadow) and (9) Toggle (BESTANDEN/NICHT BESTANDEN binary).
UX-DR6: Comprehensive states — implement cold-open (cached + background refresh), empty (`Keine Werkzeuge vorhanden`; `Keine ausstehenden Anträge`), loading (skeleton, no spinner-over-content), submitting (button disabled + action verb, no double-submit), lockout (HTTP 429 `Zu viele Fehlversuche — 30/60 Sekunden warten`), offline/network (inspection data queues locally, retries on reconnect), inline form validation errors (red underline + inline text, never toast-only), 403 (redirect to Dashboard + `Zugriff verweigert`), and post-submit inline confirmation with auto-return to refreshed Dashboard (~2s).
UX-DR7: Interaction primitives — fewest-taps inspection (single-screen checklist, `Alle bestanden` shortcut, one submit), one-submit-one-confirmation (no chained modals), auto-return to Dashboard post-inspection, PDF export current-filtered-view, anti-enumeration on all auth flows (registration, password reset, account recovery — uniform response, never reveal email existence), irreversible DSGVO deletion (typed name + mandatory Begründung + `Endgültig löschen`), dual-admin recovery (second-admin approval with Begründung + checkbox), qualification-gated inspection start (disabled `Prüfung starten` with explanation when qualification absent), one-tap status filter chips.
UX-DR8: German microcopy / Voice & Tone standard — implement German precision microcopy: name-the-entity confirmations/errors (`Kettensäge SG-01 → GRÜN — Nächste Prüfung: +12 Monate`; `NICHT BESTANDEN — Anlasser defekt`; `⛔ Wird als Außer Betrieb gesperrt`; `→ Sofort kein Login`; `Login erst möglich nach Admin-Freigabe`; `Endgültig löschen — Unumkehrbar · Audit-Pflicht`); never vague (`Gespeichert` alone, `E-Mail nicht gefunden`, generic `Fehler aufgetreten`). UI language German; docs English.
UX-DR9: Accessibility floor (regulated) — WCAG AA contrast on all text/chips/controls (incl. traffic-light colors on light and dark), interactive touch targets ≥48px (pass/fail chips may be larger), focus traversal follows reading order, no focus traps except modals (DSGVO/lockout, return focus on close), screen-reader announcements for status changes/form errors/post-submission, no icon-only buttons, keyboard-operable CTAs/filter chips/toggles/fields, Reduce Motion skips auto-return animation.
UX-DR10: Responsive & platform — mobile-first responsive web (NFR-P2/PL2): breakpoints sm 640 / md 768 / lg 1024; mobile single-column, dashboard 2×2 summary grid, admin nav collapses to hamburger/bottom; tablet two-column dashboard + persistent admin sidebar; desktop full sidebar; inspection checklist always single-column at all widths (safety-critical input never split); one codebase, touch-first + desktop; UI language German. 10 wireframes in `ux-app-ovsingen-2026-08-30/wireframes/` are the composition references.

### FR Coverage Map

FR-1: Epic 1 - Email-based authentication (login session token / HTTP 401)
FR-2: Epic 1 - Password policy (min 10 chars, clear validation)
FR-3: Epic 1 - Progressive login lockout (3→30s / 4→60s HTTP 429)
FR-4: Epic 1 - Optional TOTP MFA (6-digit code after password)
FR-5: Epic 1 (self-registration → pending_approval) / Epic 2 (admin approval) - Self-Registration and Admin Approval
FR-6: Epic 2 - Group & permission management (permission groups, resolved active permission set)
FR-7: Epic 1 - Flexible user attributes (JSON metadata on profiles)
FR-8: Epic 4 - Tool type management (name, default schedule, qualification, inspection mode)
FR-9: Epic 4 - Tool management (per-tool schedule override, bulk CSV import with per-row error report)
FR-10: Epic 4 - Flexible attributes on tools & tool types (JSON metadata)
FR-11: Epic 5 - Qualification-gated inspection (access error if no required qualification)
FR-12: Epic 5 - Checklist inspection (all items answered, pass/fail per item persisted)
FR-13: Epic 5 - Pass/Fail inspection (toggle + optional notes)
FR-14: Epic 5 - Out of Service flagging (failed item/result → OOS immediately, failure record)
FR-15: Epic 5 - OOS reinstatement (Fuehrung/Admin only, mandatory reason, clock reset)
FR-16: Epic 6 - Color-coded status dashboard (Red/Orange/Green, filterable)
FR-17: Epic 6 - Status report export PDF (Fuehrung/Admin, current filtered view)
FR-18: Epic 6 - Inspection history (reverse-chronological per tool)
FR-19: Epic 2 - Admin module access isolation (HTTP 403, hidden existence)
FR-20: Epic 2 - User approval workflow (approve → active, reject → remove pending)
FR-21: Epic 2 - User & group administration (create/edit/deactivate, revoke inherited permissions)
FR-22: Epic 2 - Qualification management (create/assign/revoke, immediate effect)
FR-23: Epic 4 - Tool & tool type configuration (checklist items, CSV import, historical records unchanged)
FR-24: Epic 3 - DSGVO compliance operations (data-access report, account deletion, anonymized history)
FR-25: Epic 1 - Change own password (revokes other sessions, audited)
FR-26: Epic 1 - Self-service password reset (single-use 30-min link, anti-enumeration)
FR-27: Epic 1 - Admin credential recovery (dual-control, high-severity audit, last-admin out-of-band)
FR-28: Epic 3 - SMTP configuration (admin interface, encrypted/masked secrets, test email)
FR-29: Epic 3 - Backup destination configuration (admin interface, encrypted credentials, test connection)
FR-30: Epic 4 - Schedule catalog management (named schedules, FK references, archiving preserves history)

## Epic List

### Epic 1: Account & Authentication
Users can register, authenticate, and manage their own identity securely. A volunteer self-registers for a `pending_approval` account, logs in with email + password (with optional TOTP MFA and progressive lockout), manages their profile, changes their own password (revoking other sessions), and recovers a forgotten password via a secure email link. Seeded admin accounts are protected by dual-admin credential recovery.
**FRs covered:** FR-1, FR-2, FR-3, FR-4, FR-5 (self-registration), FR-7, FR-25, FR-26, FR-27
**Implementation notes:** Greenfield structural seed (Epic 1 Story 1) scaffolds `cmd/server`, `internal/{user,tools,admin,platform}`, `web/` React+Vite+TS SPA, migrations, deploy, infra, justfile. User module owns identity/auth/permissions/qualifications (AD-2, AD-3, AD-13). Password reset (FR-26) depends on working email delivery; full admin SMTP panel (FR-28) ships in Epic 3 — Epic 1 uses a baseline sender. UX: UX-DR4/5/6/7/8/9 (auth surfaces, MFA, lockout, anti-enumeration microcopy, accessibility).

### Epic 2: Permissions, Roles & Administration Foundation
Administrators manage who belongs and what they are allowed to do. Admins approve pending self-registered volunteers, create/edit/deactivate users, manage user groups and roles, assign qualifications, and maintain the additive action-matched permission model. The admin module is isolated and returned HTTP 403 (with hidden existence) for non-admins.
**FRs covered:** FR-5 (admin approval), FR-6, FR-19, FR-20, FR-21, FR-22
**Implementation notes:** User module owns identity/permissions/qualifications (AD-2); additive permission model with base roles helfende/schirrmeister/fuehrende/admin and 21 permission codes (AD-12); server-side authorization is the only source of truth, admin existence hidden in SPA (AD-6). Every later epic's permission gating depends on this. UX: UX-DR5/6/7/8/9 (admin approve/reject surface, user/role/qualification management, 403 handling, German microcopy).

### Epic 3: System Configuration & Compliance
Administrators configure operational infrastructure and satisfy legal compliance. Admins manage SMTP email settings (test email, encrypted/masked secrets) and backup destinations (test connection, encrypted credentials); handle DSGVO operations including data-access reports and irreversible account deletion with anonymized inspection history.
**FRs covered:** FR-24, FR-28, FR-29
**Implementation notes:** Admin module owns exactly 3 config tables — `smtp_settings`, `backup_destinations`, `schedules` (AD-11, AD-14, AD-15); secrets encrypted at rest and write-only/masked (NFR-S4, FR-28/FR-29); DSGVO deletion runs in one transaction with immutable audit entries (AD-8, NFR-O2, FR-24). Placed before catalogue/inspection so operational email/backup and compliance are in place early. UX: UX-DR5/6/7/8/9 (DSGVO heavy two-step delete, settings forms, test actions).

### Epic 4: Equipment Catalogue & Scheduling
The Schirrmeister administers the equipment universe. Admins/Schirrmeister define tool types (name, default schedule, required qualification, checklist vs pass/fail mode), individual tools (optional per-tool schedule override), flexible attributes, the named schedule catalog, and bulk CSV import with per-row error reporting.
**FRs covered:** FR-8, FR-9, FR-10, FR-23, FR-30
**Implementation notes:** Tool module owns tool config write path (AD-10); schedule catalog owned by Admin (AD-16); per-tool schedule is a first-class FK (FR-9/AD-10); CSV import (FR-9/FR-23) with per-row error table. Gated by Epic 2 permissions (tools.manage, tool_types.manage, schedules.manage). UX: UX-DR5/6/7/8/9/10 (catalogue tabs, type editor, CSV result screen, responsive).

### Epic 5: Inspection Execution & Serviceability
Volunteers perform safety-critical inspections and manage availability. Qualified Helfer*in/Fuehrung run checklist-mode or pass/fail-mode inspections; a failed item flips the tool to Out of Service immediately with a full failure record; Fuehrung/Admin reinstate a tool with a mandatory reason, resetting the inspection clock.
**FRs covered:** FR-11, FR-12, FR-13, FR-14, FR-15
**Implementation notes:** Tool module owns inspection + OOS derivation (AD-4, AD-5, AD-7, AD-9); status always derived on read, never stored; qualification-gated start (FR-11/AD-7); OOS reinstate Fuehrung/Admin only with clock reset (FR-15/AD-9). UX: UX-DR5/6/7/8/9/10 (single-screen checklist, pass/fail toggle, defect capture, OOS, post-submit auto-return).

### Epic 6: Dashboard, Reporting & History
Users see the current operational truth, and leadership reviews and exports it. All authenticated users view a color-coded status dashboard (Red/Orange/Green, derived) filterable by status; Fuehrung/Admin export the current filtered view as PDF and inspect full per-tool inspection history.
**FRs covered:** FR-16, FR-17, FR-18
**Implementation notes:** Status derived on read via single shared clock (AD-4, AD-5); PDF export reflects active filters (FR-17); history reverse-chronological with per-initiation details (FR-18); gated by dashboard.view, report.export, inspection.history.view. UX: UX-DR5/6/7/8/9/10 (dashboard summary counts + filter chips, PDF export current view, history).

<!-- Epics and stories are appended below in order. -->

## Epic 1: Account & Authentication

Users can register, authenticate, and manage their own identity securely. A volunteer self-registers for a `pending_approval` account, logs in with email + password (with optional TOTP MFA and progressive lockout), manages their profile, changes their own password (revoking other sessions), and recovers a forgotten password via a secure email link. Seeded admin accounts are protected by dual-admin credential recovery.

### Story 1.1: Project Scaffold & Database Foundation

As a developer on the THW OV Singen App team,
I want a cold-start repository scaffold with a wired PostgreSQL data layer and a single-command runnable stack,
So that every subsequent story builds on a real, runnable application with a valid database.

**Acceptance Criteria:**

**Given** an empty repository on the `app-ovsingen` project,
**When** the scaffold is created per the architecture Structural Seed,
**Then** the repository contains `cmd/server/`, `internal/{user,tools,admin,platform}/`, `web/`, `migrations/`, `deploy/`, `infra/`, `docs/`, and a root `justfile`, each with a documented purpose
**And** `cmd/server` is the composition root wiring the module hexagons and adapters (no business logic in adapters/handlers/repositories, per AD-1).

**Given** the chosen stack (Go 1.27, chi v5.3.2, sqlc v1.31.1, pgx v5, golang-migrate v4.19.1),
**When** the Go module is initialized,
**Then** `go build ./...`, `go vet ./...`, and `go test ./...` pass with no errors
**And** chi is pinned to v5.3.2 or higher (avoiding GO-2026-4316 open-redirect, per Stack).

**Given** the database requirements (PostgreSQL 18, single shared migration set per AD-3/AD-11, NFR-R2),
**When** the local stack is started via the Compose definition (Docker or Podman),
**Then** `just db-up` provisions a PostgreSQL 18 container and `golang-migrate` applies all migrations in `migrations/`
**And** a connection pool (pgx) is established at `cmd/server` and exposed to the module repositories through their adapter ports
**And** the cold-start migration seeds the four base roles (`helfende`, `schirrmeister`, `fuehrende`, `admin`), the two pre-seeded admin accounts (credentials delivered out-of-band, never in VCS, per FR-27/AD-13), and the `admin.recovery.approve` permission (AD-12/AD-13).

**Given** the JSON error contract (Architecture error shape),
**When** any handler returns an error,
**Then** the response body is the uniform envelope `{ "error": { "code", "message", "details?" } }` with the matching HTTP status (e.g. 401, 403, 429).

**Given** the app is started with the root `justfile`,
**When** a single command (e.g. `just dev`) runs the full stack (API + SPA + DB),
**Then** the API responds to a health check, confirming the database connection is live
**And** no database command is duplicated outside the `justfile` recipes.

### Story 1.2: Self-Registration with Admin Pending Approval

As a new volunteer,
I want to register an account with my personal details,
So that I can be considered for access to the equipment-inspection app once an admin approves me.

**Acceptance Criteria:**

**Given** I am an unauthenticated visitor on the registration page (German UI),
**When** I submit Vorname, Nachname, E-Mail, Passwort and Passwort-Wiederholung,
**Then** a `pending_approval` account is created and I cannot log in until an admin approves it (FR-5)
**And** the password is validated to ≥10 characters (FR-2) with a clear inline German validation error if shorter
**And** the account is created in a single transaction; the password is stored as an Argon2id hash (AD-13, never plaintext).

**Given** I submit a registration email that already exists or is otherwise invalid,
**When** the form is submitted,
**Then** the response is a uniform anti-enumeration confirmation ("Wenn deine E-Mail bereits registriert ist, erhältst du eine Bestätigung") that does not reveal whether the email exists (FR-5/UX-DR7)
**And** no error reveals account-existence to any caller.

**Given** the registration completes,
**When** I am shown the confirmation,
**Then** I see "Dein Konto ist in Bearbeitung. Login erst möglich nach Admin-Freigabe" (UX-DR8) describing the `pending_approval` dependency.

**Given** I attempt to log in with a `pending_approval` account,
**When** I submit credentials,
**Then** authentication is rejected even with correct credentials (FR-5 pending state blocks auth)
**And** the rejection does not leak whether the account exists.

**Given** the user module schema,
**When** a profile is created,
**Then** core profile fields map to first-class typed columns and the `attributes JSONB` column is available for extensible custom attributes (FR-7/AD-3).

### Story 1.3: Email & Password Authentication

As a registered volunteer (status `active`),
I want to log in with my email and password,
So that I can access the app under a secure authenticated session.

**Acceptance Criteria:**

**Given** I have an `active` account,
**When** I submit my registered email and the correct password,
**Then** I receive a secure, cryptographically signed session token (NFR-S2) and am granted access
**And** the session token expires after the configured idle period (default 8h) and is invalidated server-side on logout (NFR-S2).

**Given** I submit an incorrect password or a non-existent email,
**When** authentication is attempted,
**Then** HTTP 401 is returned with German microcopy and the response does not reveal whether the email or password was wrong (UX-DR7 anti-enumeration).

**Given** my account status is `pending_approval` or `deactivated`,
**When** I attempt to log in with correct credentials,
**Then** authentication is rejected and no session is issued (FR-5/FR-21), without leaking account existence.

**Given** the auth gateway per AD-2/AD-6,
**When** any protected request arrives,
**Then** the server validates the session token and resolves my active permission set (all server-side, never client-trusted)
**And** permission checks return HTTP 403 when the resolved set lacks the required code.

**Given** a logged-in user requests logout,
**When** logout completes,
**Then** the session token is invalidated server-side, and future requests with that token are rejected.

### Story 1.4: Progressive Login Lockout

As the authentication service,
I want to progressively block repeated failed logins,
So that brute-force attempts are slowed without locking out legitimate users permanently.

**Acceptance Criteria:**

**Given** a login attempt fails 3 times in a row for the same account,
**When** a further login attempt is made,
**Then** the account is blocked for 30 seconds and the response is HTTP 429 (FR-3).

**Given** a login attempt fails 4 or more times in a row for the same account,
**When** a further login attempt is made,
**Then** the account is blocked for 60 seconds and the response is HTTP 429 (FR-3).

**Given** an account is in lockout,
**When** the lockout period expires,
**Then** login attempts are accepted again and the failure counter is available for a fresh cycle
**And** the counter does not permanently lock the account (no indefinite lockout from repeated failures alone).

**Given** a blocked user is shown the lockout screen (German UI),
**When** they attempt to retry,
**Then** the UI shows "Zu viele Fehlversuche — 30/60 Sekunden warten" with no retry button until the timer expires (UX-DR6/UX-DR8) and the screen is accessible (UX-DR9).

**Given** any 429 lockout is triggered,
**When** the event occurs,
**Then** it is emitted to structured logging for auth events (NFR-O1).

**Given** the lockout is per account,
**When** multiple accounts are targeted,
**Then** lockout does not cascade across unrelated accounts (each account tracked independently).

### Story 1.5: Optional TOTP Multi-Factor Authentication

As a volunteer,
I want to optionally enable TOTP two-factor authentication on my account,
So that my account is protected by a second factor beyond my password.

**Acceptance Criteria:**

**Given** I am an authenticated user on the MFA settings surface (German UI),
**When** I enable TOTP,
**Then** I am shown a shared secret and a QR code to scan into an authenticator app, and only after I confirm a valid 6-digit code is MFA enabled (FR-4)
**And** the secret is never transmitted in plaintext after provisioning and is protected at rest (NFR-S4).

**Given** MFA is enabled on my account,
**When** I log in with a valid email and password,
**Then** I am required to provide a current 6-digit TOTP code before a session is issued, and the UI shows an "MFA aktiv" indicator (FR-4/UX-DR6).

**Given** I provide an invalid or expired TOTP code,
**When** the code is validated,
**Then** login is rejected with German microcopy and no session is issued, and the rejection does not reveal why (UX-DR7 anti-enumeration).

**Given** MFA is enabled,
**When** I choose to disable it,
**Then** I must re-authenticate with a valid current TOTP code before MFA is removed.

**Given** the MFA enrollment/validation flow,
**When** any TOTP operation occurs,
**Then** relevant events are emitted to structured auth logging (NFR-O1) and the flow meets the accessibility floor (UX-DR9).

### Story 1.6: Change Own Password

As an authenticated user,
I want to change my own password,
So that I can keep my credentials current and revoke any other active sessions.

**Acceptance Criteria:**

**Given** I am logged in,
**When** I open the "Passwort ändern" surface,
**Then** I can enter my Aktuelles Passwort, a Neues Passwort, and a Wiederholung (German UI) (FR-25).

**Given** I submit the change,
**When** the current password is incorrect,
**Then** the change is rejected with a clear German error and my password remains unchanged.

**Given** I submit a new password,
**When** it is shorter than 10 characters,
**Then** it is rejected with a clear inline validation error (FR-2/UX-DR8) and no change occurs.

**Given** I successfully change my password,
**When** the change is committed,
**Then** the password is stored as an Argon2id hash, never plaintext or logged (FR-25/NFR-O1/AD-13)
**And** all other active sessions for my account are revoked instantly (FR-25), forcing re-authentication on those devices
**And** the change is audited with actor, timestamp, and operation type (NFR-O1/NFR-O2).

**Given** the change completes,
**When** I am shown a confirmation,
**Then** the UI shows "→ Andere Sitzungen beendet" (UX-DR8) and I remain logged in on the current session.

### Story 1.7: Self-Service Password Reset (Forgot Password)

As a logged-out user,
I want to reset my forgotten password,
So that I can regain access to my account without an admin.

**Acceptance Criteria:**

**Given** I am on the login page and click "Passwort vergessen" and enter my email,
**When** SMTP email delivery is configured,
**Then** I am shown the uniform confirmation "Wenn deine E-Mail registriert ist, erhältst du einen Link" (FR-26/UX-DR7 anti-enumeration), regardless of whether the account exists.

**Given** my account is `active` and SMTP email delivery is configured,
**When** I submit the reset request,
**Then** the system sends a single transactional email containing a single-use, hashed, 30-minute reset link (FR-26/AD-13), and no automated notification email is sent to other addresses (FR-26).

**Given** SMTP email delivery is **not** configured (as in early deployment, before Epic 3),
**When** I submit the reset request,
**Then** I am informed that "the admins have been notified" and will provide a one-time password (German microcopy), and no email is sent
**And** the account is flagged so the next login requires a mandatory password change.

**Given** an admin has created a one-time password for my account (created via the admin management surface in Epic 2, transmitted out-of-band in a secure way),
**When** I log in with that one-time password,
**Then** I am forced to change it immediately before accessing the app (the one-time password cannot be reused after the change).

**Given** the email reset link,
**When** I open it,
**Then** I can set a new password (min 10 chars, FR-2) plus a repeat; on success the old password is replaced with an Argon2id hash (AD-13)
**And** earlier reset tokens are invalidated if I requested multiple resets (FR-26)
**And** an expired (>30 min) or already-used (single-use) token is rejected, requiring a new link.

**Given** my account is `deactivated` or `pending_approval`,
**When** I request a reset,
**Then** the system returns the uniform confirmation without sending an actionable reset or leaking account state (anti-enumeration).

**Given** any reset request or completion,
**When** the event occurs,
**Then** it is emitted to structured auth logging and audited (NFR-O1).

> **Note:** the admin-side "create one-time password" surface is a story in Epic 2 (User & Group Administration, FR-21).

### Story 1.8: Flexible User Attributes

As a system owner,
I want to store flexible attributes on a user profile without schema migration,
So that the organization can track custom, non-core metadata about volunteers as needs evolve.

**Acceptance Criteria:**

**Given** the user profile schema,
**When** a user profile is created or updated,
**Then** all known/core attributes map to first-class typed columns and extensible custom attributes are stored in the single `attributes JSONB` column (FR-7/AD-3).

**Given** I set a custom attribute on a user profile,
**When** the profile is saved,
**Then** the attribute is persisted and retrievable with no database migration (FR-7)
**And** JSONB attribute keys are unique per semantic meaning app-wide (no two modules assign a different shape to the same key, AD-3).

**Given** the profile API/adapter,
**When** attributes are read or written,
**Then** they are served/consumed through the User module's port with valid JSON serialization and validation (AD-1/AD-3).

**Given** a custom attribute later becomes core/queryable,
**When** it is promoted,
**Then** it is migrated to a real typed column via golang-migrate and backfilled, retaining the JSONB column for continued flexibility (AD-3/NFR-R2).

### Story 1.9: Dual-Admin Credential Recovery

As the system,
I want to protect the two pre-seeded admin accounts with dual-control credential recovery,
So that no single compromised admin can take over the system.

**Acceptance Criteria:**

**Given** the system starts with exactly two pre-seeded admin accounts (credentials delivered out-of-band, never in VCS, FR-27/AD-13),
**When** an admin is locked out or forgets credentials,
**Then** they cannot self-reset recovery via the password-reset flow alone (FR-26 does not apply to admin self-recovery).

**Given** admin A is locked out,
**When** admin A requests recovery,
**Then** a recovery request is created that only admin B can act on, gated by the `admin.recovery.approve` permission (FR-27/AD-12/AD-13).

**Given** admin B reviews the recovery request,
**When** admin B approves,
**Then** they must provide a mandatory Begründung and a confirmation checkbox, and on approval admin A gains access to set a new password (min 10 chars, FR-2)
**And** the approval is recorded as a high-severity, immutable audit event with actor, timestamp, and operation type (FR-27/NFR-O2).

**Given** the last remaining admin is locked out,
**When** recovery is requested,
**Then** self-recovery is disabled and the documented out-of-band manual procedure is applied, with a mandatory audit trail (FR-27/AD-13).

**Given** any admin-recovery attempt (successful or attempted),
**When** it occurs,
**Then** it is emitted to structured auth logging (NFR-O1) and audited as high-severity (NFR-O2).

### Story 1.10: Auth UX Foundation

As a user,
I want auth screens that are consistent, accessible, and clearly worded in German,
So that I can register, log in, and manage my account confidently across devices and in light or dark mode.

**Acceptance Criteria:**

**Given** the app design tokens (from DESIGN.md),
**When** the auth surfaces are built,
**Then** they consume the DESIGN.md color / typography / rounded / spacing tokens with light + `-dark` pairs (UX-DR4) and follow the component specs (buttons, input fields, status language) (UX-DR5).

**Given** I use the auth surfaces on touch or desktop across breakpoints,
**When** any auth screen is rendered,
**Then** it is responsive (mobile-first, single-column auth, container ≤ desktop max width) (UX-DR10), with touch targets ≥48px and 200% zoom without breakage (UX-DR9).

**Given** I interact with auth forms (login, registration, password reset/change, 2FA),
**When** a validation error or state changes (submitting, lockout, success),
**Then** feedback is shown as inline German text (never toast-only), with red/blue underline states per DESIGN.md input-field spec (UX-DR5/UX-DR6/UX-DR8/UX-DR9).

**Given** any auth error or confirmation,
**When** it is rendered,
**Then** it uses the anti-enumeration microcopy from UX-DR7/UX-DR8 (e.g. "Wenn deine E-Mail registriert ist, erhältst du einen Link") and never leaks account existence.

**Given** the auth screens,
**When** they are built,
**Then** they meet the accessibility floor: keyboard-operable, focus traversal in reading order, screen-reader announcements for status changes and errors, no icon-only controls (UX-DR9)
**And** the 2-Faktor, Sperre (lockout), and password surfaces match the DESIGN.md/EXPERIENCE.md IA and component behavior.