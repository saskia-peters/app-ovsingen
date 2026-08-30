---
name: THW OV Singen App V1
type: architecture-spine
purpose: build-substrate
altitude: feature
paradigm: Modular monolith — hexagonal (ports & adapters) per module
scope: THW OV Singen App V1 ecosystem — User Directory & Auth, Tool Maintenance, Admin module
status: final
created: '2026-08-30'
updated: '2026-08-30'
binds: [Epic 1 (User Directory & Auth), Epic 2 (Tool Maintenance), Epic 3 (Admin)]
sources: [_bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/prd.md, _bmad-output/planning-artifacts/prds/prd-app-ovsingen-2026-08-29/addendum.md]
companions: []
---

# Architecture Spine — THW OV Singen App V1

## Design Paradigm

**Modular monolith, hexagonally partitioned per module.** Each feature module (User Directory & Auth, Tool Maintenance, Admin) is a self-contained hexagon: domain/core in the middle owning the business rules, with inbound driving ports (application services/handlers) and outbound driven ports (repository interfaces) on the rim. Adapters (HTTP, DB) sit outside the hexagon and are wired together only at the composition root (`cmd/server`). Modules communicate exclusively through exported port interfaces — never through each other's adapters or storage, and never by reaching into another module's internals. This is what keeps a future module (vehicle logs, event scheduling) addable without modifying existing business logic (NFR-M1).

For V1 this is one deployable application (the monolith); the container/image split is free (see Stack + Structural Seed).

## Invariants & Rules

```mermaid
flowchart TD
    subgraph cmd[Composition Root]
        HTTP["HTTP adapter"]
    end
    HTTP -->|drives| UDA["User Directory & Auth hexagon"]
    HTTP -->|drives| TMM["Tool Maintenance hexagon"]
    HTTP -->|drives| ADM["Admin hexagon"]
    TMM -. consumes .-> AuthPort["Auth port (identity + permission set)"]
    ADM -. consumes .-> AuthPort
    UDA -->|implements| AuthPort
    subgraph db[PostgreSQL JSONB]
        UDA -->|repository adapter| DB_UD[(user/auth data)]
        TMM -->|repository adapter| DB_TM[(tool/inspection data)]
        ADM -->|repository adapter| DB_AD[(admin/config data)]
    end

    classDef module fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef auth fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef db fill:#ede7f6,stroke:#5e35b1,color:#4527a0,stroke-width:2px;
    classDef comp fill:#e3f2fd,stroke:#1565c0,color:#0d47a1,stroke-width:2px;
    class UDA,TMM,ADM module;
    class AuthPort auth;
    class DB_UD,DB_TM,DB_AD db;
    class HTTP comp;
```

### AD-1 — Modular Monolith, Hexagonal per Module

- **Binds:** all — Epic 1, Epic 2, Epic 3, NFR-M1
- **Prevents:** two modules built independently selecting incompatible shapes; modules reaching into each other's storage or business logic; unpluggable modules
- **Rule:** Each feature module is an isolated Go package exposing only port interfaces across its boundary. All inbound HTTP and outbound DB/event adapters live outside the hexagon and are wired in the composition root only. No module imports another module's adapter, handler, or repository implementation; no in-memory shared state across module domains.

### AD-2 — User Directory & Auth Owns Identity and Permissions

- **Binds:** Epic 1 (FR-1..FR-6, FR-20, FR-21, FR-22, FR-25, FR-26, FR-27), Epic 2 (FR-11), Epic 3 (FR-19, FR-20, FR-24), NFR-S2, NFR-S3
- **Prevents:** auth/permission logic being duplicated or owned by Tool Maintenance or Admin; divergent permission checks across modules; admin isolation leaking
- **Rule:** The User Directory & Auth module is the single owner of users, groups, permissions, sessions, MFA/TOTP, credentials and password-recovery tokens, account state (pending_approval/active/deactivated), and qualifications. It exposes a single port — `Service` — from which consumers resolve the current caller's identity, the caller's **granted Qualifications** (for AD-7), and the caller's *resolved permission set* (the additive union of all permission-group and direct grants, AD-12). Tool Maintenance and Admin consume only this port; they never store auth data, never author their own SQL over user/qualification tables, and never implement their own permission checks. FR-19 (admin route isolation, HTTP 403, existence hidden) is enforced server-side in the composition-root gateway against this port. **Permission and qualification revocation takes effect immediately on the very next check** — the auth port resolves against the live user state per request, never against a session-cached permission snapshot (FR-21/FR-22). New approved users are seeded with the `helfende` base role (AD-12, FR-5/FR-20). Password self-service, forgot-password recovery, and dual-control admin recovery all live in this module (FR-25/26/27, AD-13).

### AD-3 — First-Class Columns + JSONB Extension Surface

- **Binds:** Epic 1 (FR-7), Epic 2 (FR-10), NFR-R2
- **Prevents:** DB migrations required for every new THW attribute; type-unsafe dumping of core data into JSON
- **Rule:** Users, Tools, and Tool Types expose first-class typed columns for all known/core attributes, plus exactly one `attributes JSONB` column per entity holding the extensible custom-attribute surface (FR-7/FR-10). New attributes that are not yet first-class land in JSONB with no schema migration. Migration path: when an attribute becomes core/queryable, promote it to a real column via golang-migrate and backfill, retaining the JSONB column for continued flexibility.

### AD-4 — Tool Status Is Derived, Never Stored

- **Binds:** Epic 2 (FR-14, FR-15, FR-16, FR-17, FR-18), Dashboard, status report
- **Prevents:** UI/API/dashboard drift from a separately-stored/short-lived status value; two independent consumers disagreeing on a tool's color
- **Rule:** A tool's status (Red/Orange/Green/Out-of-Service) is always computed on read from inspection records and due-date math — never stored as a column. Out-of-Service is itself derived: a tool is Out-of-Service (displayed Red) iff its latest inspection — or any checklist item of its latest checklist inspection — failed *and* it has not since been reinstated. **Reinstatement is the sole exit from Out-of-Service**: a later passing inspection is not enough; the Fuehrung/Admin reinstatement (AD-9) must record an event that clears it. A tool with no successful inspection yet and no next_due (never inspected) is displayed Red (not due / not ready). Computed with equal force against every read path (Dashboard, report export, tool detail).

### AD-5 — One Shared Inspection Clock (Due Date Per Tool)

- **Binds:** Epic 2 (FR-8, FR-9, FR-15, FR-16, FR-17, FR-18)
- **Prevents:** Dashboard and inspection module computing different next-due dates for the same tool; inconsistent treatment of schedule overrides
- **Rule:** `next_due` is computed per tool as `last_successful_inspection_date + schedule_interval`, where `schedule_interval` is the Tool's custom interval if set, else the Tool Type default (FR-9). On Out-of-Service reinstatement the clock resets: `next_due = reinstatement_date + schedule_interval` (FR-15). Color thresholds (FR-16): Red = past due (or Out-of-Service); Orange = due within the next 14 calendar days; Green = due more than 14 days out. This exact rule is a single shared function all consumers call.

### AD-6 — Server-Side Authorization Is the Only Source of Truth

- **Binds:** Epic 3 (FR-19), NFR-S3, NFR-O1
- **Prevents:** front-end-only hiding of admin routes; direct-URL bypass of admin endpoints
- **Rule:** Every request-bearing capability (API route, admin operation, frontend action) is bound to exactly one permission code, and the server re-validates that the caller's resolved permission set (AD-12) contains it before the action runs — **a permission must match the action to carry it out**. If not present, the server returns HTTP 403 (FR-19) and the admin module's existence is not surfaced to non-Admins anywhere in the SPA. Frontend visibility is supplementary only and never substitutes for the server-side gate.

### AD-7 — Qualification Gating

- **Binds:** Epic 2 (FR-11, FR-12, FR-13, FR-14), Epic 3 (FR-22)
- **Prevents:** an unqualified user inspecting a tool despite UI affordances; divergence between "can initiate" and qualification records
- **Rule:** Qualification is data owned by the User Directory & Auth module (AD-2). A single, shared qualification vocabulary identifiers the required Qualification of a Tool Type and the granted Qualifications on a user; the Tool Maintenance module obtains both from the auth port (never by joining `users.qualifications` itself) and permits initiate/submit only when the caller holds the required one (FR-11). Tool Types declare their required Qualification via a `TOOL_TYPES→QUALIFICATIONS` cross-module reference owned by the Tool module but pointing at User-module qualification IDs (see AD-11). Assigning/revoking a Qualification in Admin takes effect immediately on the very next check (FR-22, AD-2).

### AD-8 — Cross-Module User-Lifecycle & DSGVO Data Flows

- **Binds:** Epic 1 (FR-5), Epic 3 (FR-24), NFR-O2, NFR-S3
- **Prevents:** the User or Admin module mutating another module's tables directly to satisfy user deletion; the DSGVO erasure/anonymization being split across two owners with clashing shapes; a deletion wedging mid-transaction
- **Rule:** User lifecycle events that reach other modules' data (account deletion → anonymized inspection history, FR-24; data-access report assembly) are executed by a single **composition-root orchestration** that calls each owning module through its exported port — never by one module writing another's SQL. When a user is deleted, the Tool Maintenance module's inspection history keeps its rows but the inspector reference is rewritten to the literal `"Deleted User"` (FR-24) via the Tool module's own user-lifecycle port; personal fields are purged from the User module and remain so. The whole DSGVO deletion runs as one transaction and produces an immutable audit entry (NFR-O2). Both Admin (operation trigger) and User module (lifecycle authority) cooperate through these ports; neither authors the other's tables.

### AD-9 — Out-of-Service Reinstatement (Fuehrung/Admin, Clock Reset)

- **Binds:** Epic 2 (FR-14, FR-15), Epic 3, NFR-O1
- **Prevents:** the Out-of-Service state being mutable by the wrong role; a reinstatement that fails to reset the inspection clock; diverging reinstate rules between Dashboard and tool detail
- **Rule:** Only **Fuehrung** and **Admin** (via the resolved permission set, AD-2/AD-6) can clear a Tool's Out-of-Service status (FR-15). Reinstatement requires a mandatory non-empty reason/notes field and records an event (actor, timestamp, reason). It is the sole exit from Out-of-Service (AD-4), and it **resets the inspection clock**: `next_due = reinstatement_date + schedule_interval` (AD-5). Helfer*in cannot reinstate. Every read path derives the post-reinstatement color from the reset `next_due` immediately.

### AD-10 — Tool/Tool-Type Configuration Write Path (Single Owner)

- **Binds:** Epic 2 (FR-8, FR-9), Epic 3 (FR-23), AD-5
- **Prevents:** Admin and Tool Maintenance both authoring the same tool/tool_type rows; a CSV admin import writing an `attributes`/`schedule` key that the shared due-date function never reads (adversarial two-sources-of-truth)
- **Rule:** Tools, Tool Types, and checklist items remain owned by the **Tool Maintenance** module (their persistence lives with AD-1's Tool hexagon). Admin configures them — including FR-23 checklist definition and FR-9 CSV bulk import — exclusively through the Tool module's exported **configuration port**, never by Admin-owned copies or ad-hoc SQL over the same tables. A Tool's custom schedule is the first-class `schedule_override` column (never a JSONB `attributes` key); schedule data is read only via AD-5's shared function.

### AD-11 — Cross-Module Schema & Reference Ownership

- **Binds:** all — Epic 1, Epic 2, Epic 3, NFR-R2
- **Prevents:** two modules declaring the same table or name-colliding with a shared identifier; ambiguous cross-module foreign keys; two modules writing the same JSONB key with different shapes
- **Rule:** One golang-migrate migration set is the single schema authority (NFR-R2). Each table has exactly one owning module (users/auth → User; tools/tool_types/checklist/inspections → Tool; the whole set evolves only via migrations). Cross-module references (e.g. an inspection's `inspector_id` → user, a Tool Type's required Qualification → user-module qualification) are declared as foreign keys to the owning module's identifiers, with the referencing module reading those identifiers only through the owning module's port. JSONB `attributes` keys are unique per semantic meaning across the whole app (no two modules assign a different shape to the same key).

### AD-12 — Action-Matched, Additive Permission Model

- **Binds:** Epic 1 (FR-6, FR-21, FR-22), Epic 3 (FR-19, FR-21, FR-23), NFR-S3, AD-2, AD-6
- **Prevents:** roles granted without a matching capability; permission checks diverging from the shipped vocabulary; negative/deny permissions; users locked out by group nesting; a user accumulating rights that no longer bind to any real action
- **Rule:** The permission model is **flat and additive**:
  - **Every action in the app is tied to exactly one permission code** (the base series below). A user may carry out an action if and only if their *resolved permission set* contains that action's permission — nothing else grants it (NFR-S3). There are **no deny/negative permissions**; a permission present is granted, absent is denied.
  - **A user's resolved permission set is the additive union (set-union, no precedence) of all permissions from all their permission-group memberships and their direct grants.** Users may hold more than one group/role at once (FR-6). Removing a group/direct grant subtracts exactly its permissions, immediately.
  - **Roles are just named permission groups.** Four **base roles** ship pre-seeded: `helfende`, `fuehrende`, `admin`, `schirrmeister` (see the role matrix). **Admins may create additional permission groups, give them a name, choose their permissions, and assign them to users** — and may also edit the permission sets of the base roles themselves (all are the same kind of entity). These groups are **flat: no groups inside groups in V1** — a permission group contains only permissions, never other groups.
  - **V1 flatness:** because groups cannot nest, resolution is a single level of union over a user's groups plus direct grants; there is no transitive inclusion to compute.
  - **User groups are organisational only** (team membership) and grant **no** permission by themselves — only the permission-group/direct-grant path affects access (isolation from `user_groups` in the resolution, AD-2).
  - On approval (FR-5/FR-20) a new user is seeded with the **`helfende`** base role; an admin can add/remove roles thereafter.

### AD-13 — Self-Service Password Management + Dual-Admin Credential Recovery

- **Binds:** Epic 1 (FR-25, FR-26, FR-27), NFR-S2, NFR-S4, NFR-O2, AD-2, AD-12
- **Prevents:** passwords unrecoverable for day-to-day users; single-admin accounts that a single compromise could take over; admin lockout with no safe recovery path; reset tokens that are replayable or long-lived
- **Rule:** The User module owns all credential and recovery flows (FR-25/26/27, AD-2) and exposes them purely through its `Service` port:
  - **Change own password (FR-25).** Any authenticated user changes their own password by confirming the current password and satisfying the FR-2 policy (≥ 10 chars). Password hashes use Argon2id; plaintext is never stored or logged. A successful change revokes the user's other server-side sessions and is audited (NFR-O1). No elevated permission is required — access is intrinsic to one's own account (no `*` permission needed; gated only by authentication, consistent with AD-12's "one permission per action", where ownership of self is the capability).
  - **Forgot-password reset (FR-26).** The login flow accepts an email; if it matches an `active` account, the module emits a **single transactional email** (SMTP, env/secrets-configured, NFR-S4) carrying a single-use, 30-minute, hashed-at-rest reset token (newer request invalidates older ones). Response is uniform for known/unknown email (no enumeration, NFR-S1-friendly). Reset enforces FR-2 and revokes sessions; audited. This email is the **only** email in V1 and does not reopen the automated-notification non-goal (PRD §5).
  - **Dual-admin recovery (FR-27).** Exactly **two `admin` accounts are seeded** at deployment; credentials are generated and distributed out-of-band (NFR-S4), never in VCS. An admin in a locked-out/forgotten state **cannot** self-reset via the FR-26 flow alone; recovery requires the **other admin's** authorization, is gated by an admin-only permission (see base series `admin.recovery.approve`), and is recorded as a high-severity immutable audit event (NFR-O2). Recovery of the **last remaining** admin is deliberately **not** reachable by any self-service path; it is a documented, out-of-band manual bootstrap requiring both admins (or a designated recovery sponsor). Recovery always enforces FR-2.
- **Seeding note (extends AD-12/FR-5):** the two base admins are seeded with the `admin` role and the `admin.recovery.approve` permission; all four base roles plus the two admin accounts are part of the cold-start migration seed (NFR-R2).

## Consistency Conventions

| Concern | Convention |
| --- | --- |
| IDs | UUID (v7) primary keys for all entities; opaque, never exposed via sequential integers |
| Timestamps | UTC, RFC 3339; all "now" from one clock source (server `time.Now`), never client clocks |
| Dates | Inspection dates stored as `date`; due-date math done in UTC day-boundaries |
| Error shape | Uniform JSON error envelope `{ "error": { "code", "message", "details?" } }`; matching HTTP status codes per PRD (401, 403, 429, etc.) |
| Permission checking | Occur solely through the auth port's resolved permission set (AD-2, AD-6); one shared permission-code vocabulary across all modules |
| Table ownership | Each table owned by exactly one module; cross-module references via FK + owning-module port (AD-11) |
| JSONB keys | Unique semantic meaning across the whole app; no two modules assign different shapes to one key (AD-3, AD-11) |
| Mutations | All cross-entity writes in a DB transaction; inspection submission and status effects atomic; DSGVO deletion cross-module + atomic (AD-8) |
| Logging | Structured JSON (NFR-O1); auth events, permission denials, inspections, Out-of-Service transitions, reinstatements, DSGVO ops logged |
| Flexible attrs | Accessed via JSONB `attributes` column only through typed helpers; no ad-hoc raw SQL across modules |
| Modules | Standard library, chi, sqlc/pgx — never a different router/ORM per module |

## Stack

Seed — verified current at authoring (2026-08-30); code owns these once they exist.

| Name | Version |
| --- | --- |
| Go | 1.27 (current stable on 2026-08-30; note 1.24 is EOL) |
| go-chi/chi | v5.3.2 (requires Go ≥1.23) |
| sqlc (pgx codegen) | v1.31.1 |
| jackc/pgx | v5 |
| golang-migrate | v4.19.1 |
| PostgreSQL | 18 (current stable) |
| TypeScript | 6.x (7.x is native/current; adopt 6.x safely for tooling maturity) |
| React | 19 |
| Vite | 8.x (current stable) |
| Docker / docker-compose | latest stable — used for deployed targets (NFR-R1) |
| Podman / podman-compose | local development container runtime (drop-in alternative to Docker) |
| just | github.com/casey/just (1.x) — project command runner; single `justfile` for dev & deploy recipes |
| OpenTofu | v1.12.6 (`tofu`) — IaC for provisioning cloud resources, incl. GCP |
| Google Provider (OpenTofu) | google 5.x — provisions Cloud Run, Artifact Registry, Cloud SQL tiers |
| Google Cloud Run | managed serverless containers; `min_instance_count = 0` scales to zero when idle |

Security note: GO-2026-4316 (open redirect via `RedirectSlashes`) affects chi **v5.2.2–5.2.3**, fixed in v5.2.4; pinning chi v5.3.2+ avoids it. Run dependency/security scans in CI (NFR-S5).

**Local development** runs the multi-service stack with **Podman / podman-compose** (compatible with the same Compose definition used by Docker in deployment). The Compose file is the single source for both runtimes.

## Structural Seed

The build owns detail; this is the cold-start scaffold and the operational envelope.

```text
cmd/server/            # composition root: wire hexagons + adapters; mount HTTP, gateways, middleware
internal/
  user/               # User Directory & Auth hexagon (core, ports, adapters)
  tools/              # Tool Maintenance hexagon
  admin/              # Admin hexagon (configuration + DSGVO operations)
  platform/           # cross-cutting: logger, config, migrate, middleware (NOT business)
web/                  # React + Vite + TS SPA (frontend UI; no business logic)
migrations/           # golang-migrate versioned SQL (NFR-R2)
deploy/               # compose definition (docker-compose / podman-compose) + backup config
infra/                # OpenTofu modules for cloud resources (GCP: Cloud Run, Artifact Registry, Cloud SQL)
justfile              # dev & deploy command recipes (casey/just); single entry point for the project
docs/                 # Docusaurus documentation site (NFR-M4)
```

```mermaid
flowchart LR
    subgraph F[Frontend container]
        SPA["React/Vite SPA"]
    end
    subgraph A[Go API container]
        GW["Auth gateway (AD-2/AD-6)"]
        UD["User hexagon"]
        TM["Tool hexagon"]
        AD["Admin hexagon"]
    end
    subgraph D[Postgres container]
        DB[(PostgreSQL 18)]
    end
    SPA -->|HTTPS REST JSON| GW
    GW --> UD
    GW --> TM
    GW --> AD
    UD --> DB
    TM --> DB
    AD --> DB
    BKP["backup job/container (pg-based)"] --> DB
    BKP --> DEST[(configurable destination)]

    classDef fe fill:#e3f2fd,stroke:#1565c0,color:#0d47a1,stroke-width:2px;
    classDef mod fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef db fill:#ede7f6,stroke:#5e35b1,color:#4527a0,stroke-width:2px;
    classDef bkp fill:#fce4ec,stroke:#c2185b,color:#880e4f,stroke-width:2px;
    class SPA fe;
    class GW,UD,TM,AD mod;
    class DB,DEST db;
    class BKP bkp;
```

## Database Artifacts

The one golang-migrate schema (NFR-R2, AD-11) holds every table the app persists. **User Groups** (who people are / how volunteers are organised) and **Permission Groups** (what people are allowed to do) are **two separate tables** — a volunteer's team membership is independent of the access they hold (FR-6, FR-21). Roles and permission groups are the **same kind of entity**: the four base roles (`helfende`, `fuehrende`, `admin`, `schirrmeister`) are just pre-seeded permission groups; admins can create more named ones (AD-12). In addition to the base roles, the deployment seed creates **two `admin` accounts** (dual-admin bootstrap, FR-27 / AD-13) whose credentials are generated and distributed out-of-band. Permission groups are **flat — no groups inside groups in V1**. Each table below belongs to exactly one module and may only be reached through that module's port (AD-1, AD-11).

| # | Table | Stores | Owned by | Notes / Non-technical meaning |
| --- | --- | --- | --- | --- |
| 1 | `users` | One row per person (name, email, account state, MFA flag) | User | The people. `state` = `pending_approval` / `active` / `deactivated` (FR-5). `attributes JSONB` carries flexible metadata (FR-7). |
| 2 | `user_groups` | Organisational groups (e.g. "Gruppe Ost", "Fachgruppe Wassergefahren") | User | The **teams** volunteers are part of — purely organisational, grants **no** access by itself (AD-12). |
| 3 | `user_group_members` | The link: which user is in which user group (many-to-many) | User | "Who belongs to which team." |
| 4 | `permission_groups` | Named bundles of permissions — the **roles**. Pre-seeded: `helfende`, `fuehrende`, `admin`, `schirrmeister`; admins may add named groups (AD-12) | User | The **roles** that grant access. Users can belong to several at once; rights are additive (FR-6). Flat (no nesting). |
| 5 | `permissions` | The granular permission codes (the base series below — one per app action) | User | Individual capabilities; a user may perform an action iff its permission is in their resolved set (AD-12). |
| 6 | `permission_group_permissions` | Which permissions a permission group (role) contains (many-to-many) | User | "The `admin` role includes these capabilities." Editable to redefine roles (incl. base roles). |
| 7 | `user_permission_groups` | Which permission groups (roles) a user belongs to (many-to-many) | User | "This person is `admin` **and** `schirrmeister`." Effective rights = union of all these + direct grants (AD-12). |
| 8 | `user_permissions` | Direct one-off permissions granted to a single user (many-to-many, additive) | User | Ad-hoc individual grants that bypass any group (FR-6). No negative/deny entries (AD-12). |
| 9 | `qualifications` | The qualification vocabulary (e.g. "Chainsaw", "Generator") | User | The recognised safety certificates. |
| 10 | `user_qualifications` | Which certificates a person holds (many-to-many) | User | "This person holds the Chainsaw certificate." Feeds AD-7 gating. |
| 11 | `audit_log` | Immutable record of auth, permission, inspection, reinstate, qualification & DSGVO events | User | Compliance trail (NFR-O1, NFR-O2). |
| 12 | `tool_types` | Equipment categories (chainsaw, generator…) + default schedule, mode, required qualification | Tool | The equipment catalogue; defines default inspection schedule + pass/fail or checklist mode (FR-8). Label + `attributes JSONB`. |
| 13 | `tool_type_qualifications` | Which qualification is required to inspect each tool type (cross-module ref) | Tool | Gating link — a tool type's required certificate (FR-8, AD-7). |
| 14 | `checklist_items` | The checklist questions/tasks a tool type defines | Tool | Definition of checklist-based inspection (FR-8, FR-12, FR-23). `schedule_override`-style ordering is code-level. |
| 15 | `tools` | Individual physical items, each belonging to a tool type | Tool | The actual equipment on site. `schedule_override date` (custom interval, FR-9) + `attributes JSONB` (FR-10). |
| 16 | `inspections` | Every inspection run (header): tool, inspector, date, result | Tool | The inspection log (FR-18). Never deleted; anonymised on user deletion (FR-24). |
| 17 | `inspection_items` | Per-checkline results for checklist inspections | Tool | Checklist details, if the tool type is checklist mode (FR-12). |
| 18 | `reinstatements` | Out-of-Service clearing events (actor, reason, date) | Tool | Who cleared a tool Out-of-Service and why (AD-9). Sole exit from OOS (AD-4). |
| 19 | `password_reset_tokens` | Single-use, expiring credential-recovery tokens (hashed value, user, expiry, agent/approver) | User | Powers FR-26 forgot-password reset and FR-27 dual-admin recovery. One active token per user; new request invalidates older ones (AD-13). |

> **Attribution:** tables 1–11 and **19** belong to the **User** hexagon, 12–18 to the **Tool** hexagon. The Admin module owns no tables — it configures through the Tool module's config port and the User module's lifecycle/credential-recovery port (AD-8, AD-10, AD-13).

### Base permission series (visible from day one)

Every **action** in the app maps to exactly one permission code (AD-12). This is the complete V1 base series — one row per action. Admins grant/revoke these; the codes never change shape, only which role has them. A user performs an action **iff** its code is in their resolved (additive-union) permission set.

| Permission code | The action it guards | Module |
| --- | --- | --- |
| `dashboard.view` | View the status dashboard (FR-16) | Tool |
| `inspection.submit` | Initiate & submit an inspection (FR-11/12/13; **also** needs the qualification of the tool type, AD-7) | Tool |
| `inspection.history.view` | View a tool's inspection history (FR-18) | Tool |
| `report.export` | Export the PDF status report (FR-17) | Tool |
| `tool.reinstate` | Clear a tool's Out-of-Service status (FR-15) | Tool |
| `tools.manage` | Create/edit tools, CSV bulk import (FR-9) | Tool |
| `tool_types.manage` | Create/edit tool types & checklist items (FR-8, FR-23) | Tool |
| `users.view` | View the user directory | User |
| `users.approve` | Approve/reject pending registrations (FR-20) | User |
| `users.manage` | Create/edit/deactivate user accounts (FR-21) | User |
| `user_groups.manage` | Create/edit user groups & assign members (FR-21) | User |
| `roles.create` | Create a new named permission group (AD-12) | User |
| `roles.edit` | Change the permissions inside a permission group, incl. the 4 base roles (AD-12) | User |
| `roles.assign` | Assign permission groups/roles to users (FR-6) | User |
| `qualifications.manage` | Create qualifications, assign/revoke on users (FR-22) | User |
| `dsgvo.access_report` | Generate a user's data access report (FR-24) | User |
| `dsgvo.delete` | Execute account deletion / right-to-be-forgotten (FR-24) | User |
| `admin.recovery.approve` | Authorize credential recovery for a locked-out/forgotten admin (FR-27) | User |

### Base role matrix

The **additive** permission set each of the four base roles ships with. A tick means the role's members carry that permission; holding several roles adds the ticks together (AD-12). **Bold** marks the permission(s) a role uniquely stands out with.

| Permission | `helfende` | `schirrmeister` | `fuehrende` | `admin` |
| --- | :---: | :---: | :---: | :---: |
| `dashboard.view` | ✔ | ✔ | ✔ | ✔ |
| `inspection.submit` | ✔ | ✔ | ✔ | ✔ |
| `tools.manage` | | ✔ | | ✔ |
| `tool_types.manage` | | ✔ | | ✔ |
| `inspection.history.view` | | ✔ | ✔ | ✔ |
| `report.export` | | | ✔ | ✔ |
| `tool.reinstate` | | | ✔ | ✔ |
| `users.view` | | | | ✔ |
| `users.approve` | | | | ✔ |
| `users.manage` | | | | ✔ |
| `user_groups.manage` | | | | ✔ |
| `roles.create` | | | | ✔ |
| `roles.edit` | | | | ✔ |
| `roles.assign` | | | | ✔ |
| `qualifications.manage` | | | | ✔ |
| `dsgvo.access_report` | | | | ✔ |
| `dsgvo.delete` | | | | ✔ |
| `admin.recovery.approve` | | | | ✔ |

> **Reading the matrix:** `helfende` = "volunteer who inspects". `schirrmeister` = **equipment caretaker** — everything a volunteer can do, **plus** managing tools and tool types (and their inspection history). `fuehrende` = **leadership** — inspection plus history, PDF export, and reinstate (but not tool/tool-type administration, which belongs to the Schirrmeister). `admin` = **everything**. All four are editable by an admin, and new custom groups can be added (AD-12). Inspection still always needs the tool type's required qualification (AD-7), independent of these roles.

**Entity relations** (derivations are AD-4/AD-5; attribute detail is owned by code):

```mermaid
erDiagram
    USERS ||--o{ USER_GROUP_MEMBERS : has
    USER_GROUPS ||--o{ USER_GROUP_MEMBERS : contains
    USERS ||--o{ USER_PERMISSION_GROUPS : assigned
    PERMISSION_GROUPS ||--o{ USER_PERMISSION_GROUPS : assigned
    PERMISSION_GROUPS ||--o{ PERMISSION_GROUP_PERMISSIONS : includes
    PERMISSIONS ||--o{ PERMISSION_GROUP_PERMISSIONS : included_in
    USERS ||--o{ USER_PERMISSIONS : granted
    PERMISSIONS ||--o{ USER_PERMISSIONS : granted_to
    USERS ||--o{ USER_QUALIFICATIONS : holds
    QUALIFICATIONS ||--o{ USER_QUALIFICATIONS : held_by
    USERS ||--o{ PASSWORD_RESET_TOKENS : issues
    TOOL_TYPES ||--o{ TOOL_TYPE_QUALIFICATIONS : requires
    QUALIFICATIONS ||--o{ TOOL_TYPE_QUALIFICATIONS : required_by
    TOOL_TYPES ||--o{ CHECKLIST_ITEMS : defines
    TOOL_TYPES ||--o{ TOOLS : classifies
    TOOLS ||--o{ INSPECTIONS : receives
    INSPECTIONS ||--o{ INSPECTION_ITEMS : has
    USERS ||--o{ INSPECTIONS : performs

    classDef u fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px
    classDef ujunc fill:#fff8e1,stroke:#f9a825,color:#8d6e00,stroke-width:1px
    classDef t fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px
    classDef tjunc fill:#f1f8e9,stroke:#7cb342,color:#33691e,stroke-width:1px
    classDef cross fill:#f3e5f5,stroke:#8e24aa,color:#4a148c,stroke-width:2px
    class USERS u
    class USER_GROUPS u
    class PERMISSION_GROUPS u
    class PERMISSIONS u
    class QUALIFICATIONS u
    class PASSWORD_RESET_TOKENS u
    class USER_GROUP_MEMBERS ujunc
    class USER_PERMISSION_GROUPS ujunc
    class PERMISSION_GROUP_PERMISSIONS ujunc
    class USER_PERMISSIONS ujunc
    class USER_QUALIFICATIONS ujunc
    class TOOL_TYPES t
    class TOOLS t
    class INSPECTIONS t
    class INSPECTION_ITEMS t
    class CHECKLIST_ITEMS t
    class TOOL_TYPE_QUALIFICATIONS cross
```

**Who can do what** — how a logged-in user's effective access is resolved (AD-2, AD-6, AD-12). Every request re-resolves this live against the tables above, so revoking a role/direct grant/qualification takes effect immediately. Permissions are **additive** (set union, no precedence) and a permission must **match the action** for it to run:

```mermaid
flowchart TD
    U["Logged-in User"]
    U --> G["User Groups<br/>(teams — no access effect)"]
    U -->|"belongs to several 'roles' at once"| PG["Permission Groups / Roles<br/>(preseeded: helfende, schirrmeister,<br/>fuehrende, admin + custom ones)"]
    U -->|"one-off"| DP["Direct permissions"]
    PG --> P["Permissions<br/>(additive union, no deny)"]
    DP --> P
    P --> E["Resolved permission set"]
    E --> Q{"Does the set contain<br/>this action's permission?<br/>(AD-12)"}
    Q -- No --> X["Action blocked — HTTP 403<br/>(AD-6)"]
    Q -- Yes --> A["Action allowed<br/>e.g. inspection still needs qualification (AD-7)"]
    G -. organisational only, no access effect .-> X

    classDef user fill:#e3f2fd,stroke:#1565c0,color:#0d47a1,stroke-width:2px;
    classDef teams fill:#eeeeee,stroke:#757575,color:#424242,stroke-width:1px;
    classDef roles fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef perms fill:#fff8e1,stroke:#f9a825,color:#8d6e00,stroke-width:1px;
    classDef resolve fill:#ede7f6,stroke:#5e35b1,color:#4527a0,stroke-width:2px;
    classDef decision fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef allow fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef deny fill:#ffebee,stroke:#c62828,color:#b71c1c,stroke-width:2px;
    class U user;
    class G teams;
    class PG,DP roles;
    class P perms;
    class E resolve;
    class Q decision;
    class A allow;
    class X deny;
```

## The user-facing flows

These diagrams are the day-to-day behaviour a volunteer, a leader and an admin experience. They are what the invariants above guarantee, shown end-to-end.

**Tool status lifecycle** — how a piece of equipment moves between the colour states on the dashboard. Status is always **derived from stored facts** (inspections, due dates, reinstatements), never saved as a value, so the list can never disagree with the detail view (AD-4/AD-5):

```mermaid
stateDiagram-v2
    direction LR
    state "Green (current)" as G
    state "Orange (due ≤14d)" as O
    state "Red (past due)" as R
    state "Out of Service" as OOS
    [*] --> G : never inspected yet
    G --> O : time passes
    O --> R : due date passed
    R --> G : inspection passes
    R --> OOS : inspection fails
    OOS --> G : reinstated by Fuehrung/Admin<br/>(clock resets)
    OOS --> R : not reinstated (past due)

    classDef g fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef o fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef r fill:#ffebee,stroke:#c62828,color:#b71c1c,stroke-width:2px;
    classDef oos fill:#424242,stroke:#212121,color:#ffffff,stroke-width:2px;
    class G g;
    class O o;
    class R r;
    class OOS oos;
```

**Inspection flow** — what happens when a volunteer carries out an inspection in the warehouse. Only a qualified person can proceed, and the outcome automatically updates the tool's status and next due date (AD-5, AD-7):

```mermaid
flowchart TD
    A["Volunteer opens a Tool"] --> B{"Qualified?<br/>(auth port check — AD-7)"}
    B -- No --> X["Blocked — cannot inspect<br/>(HTTP 403)"]
    B -- Yes --> C{"Tool Type mode?<br/>(FR-8)"}
    C -- "Pass/Fail" --> D["Submit pass or fail"]
    C -- "Checklist" --> E["Answer each checklist item"]
    E --> F{"All items pass?"}
    F -- No --> OOS["Tool → Out of Service<br/>(failed inspection)"]
    F -- Yes --> D
    D --> G{"Pass?"}
    G -- No --> OOS
    G -- Yes --> H["Tool → Green<br/>next due = today + schedule (AD-5)"]
    OOS --> R["Fuehrung/Admin reinstates<br/>with reason — clock resets (AD-9)"]

    classDef action fill:#e3f2fd,stroke:#1565c0,color:#0d47a1,stroke-width:2px;
    classDef decision fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef pass fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef fail fill:#ffebee,stroke:#c62828,color:#b71c1c,stroke-width:2px;
    classDef deny fill:#fdecea,stroke:#b71c1c,color:#b71c1c,stroke-width:2px;
    classDef reinstate fill:#ede7f6,stroke:#5e35b1,color:#4527a0,stroke-width:2px;
    class A action;
    class B,C,F,G decision;
    class D,E,H pass;
    class OOS fail;
    class X deny;
    class R reinstate;
```

**User approval lifecycle** — how a new volunteer gets an account and access (FR-5, FR-20; AD-2):

```mermaid
stateDiagram-v2
    direction LR
    state "Pending approval" as P
    state "Active" as A
    state "Deactivated" as D
    [*] --> P : self-registration (FR-5)
    P --> A : Admin approves (FR-20)
    P --> D : Admin rejects/deletes
    A --> D : Admin deactivates
    D --> A : Admin reactivates
    A --> [*] : DSGVO erasure (FR-24)
    note right of A
      Approved users start with the 'helfende' role (AD-12).
      Effective access = additive union of their
      permission groups/roles + direct grants,
      resolved live on every request (AD-2/AD-6/AD-12).
      Admins grant further roles or create new named groups (FR-6).
    end note

    classDef pending fill:#fff3e0,stroke:#ef6c00,color:#e65100,stroke-width:2px;
    classDef active fill:#e8f5e9,stroke:#2e7d32,color:#1b5e20,stroke-width:2px;
    classDef deact fill:#eeeeee,stroke:#757575,color:#424242,stroke-width:2px;
    class P pending;
    class A active;
    class D deact;
```

## Capability → Architecture Map

| Capability / Area | Lives in | Governed by |
| --- | --- | --- |
| Epic 1 — User Directory & Auth (FR-1..FR-7, FR-25..FR-27) | `internal/user` hexagon (core + auth port) | AD-2, AD-3, AD-13 |
| Epic 2 — Tool Maintenance (FR-8..FR-18) | `internal/tools` hexagon | AD-4, AD-5, AD-7, AD-9, AD-10 |
| Epic 3 — Admin & Config (FR-19..FR-24) | `internal/admin` hexagon | AD-2, AD-6, AD-8, AD-10 |
| Cross-module auth/permissions | `internal/user` auth port consumed by all | AD-2, AD-6, AD-11, AD-12 |
| Permission model (roles, additive access, action match) | `internal/user` core (vocabulary + resolution) | AD-12, AD-2, AD-6 |
| Password self-service & admin credential recovery (FR-25/26/27) | `internal/user` core (credentials + reset tokens) | AD-13, AD-2 |
| Tool status & dashboard | `internal/tools` (derived) | AD-4, AD-5 |
| Tool/Tool-Type config write path | Admin triggers → Tool module config port | AD-10 |
| DSGVO ops (FR-24) / user lifecycle | Admin trigger → composition-root orchestration → owning-module ports | AD-8, AD-11 |

## Deferred

- **Operational envelope details** — the PRD's NFRs are binding targets, not invented here: NFR-S1 (TLS 1.2+, HTTP rejection/redirect), NFR-S4 (secrets via env/secret-manager, never in VCS), NFR-R4 (99% availability), and dev/staging/prod environment split are all **required**, but the concrete infra/provider, environment topology, and per-environment owners are deferred to the deploy epic/site; the spine fixes the constraints, not the host. **Firm choices already made:** all deploy/dev commands go through a single `justfile` (just); cloud resources are provisioned by **OpenTofu** IaC; the **testing/staging platform is Google Cloud Run** with containers scaled to zero when idle (`min_instance_count = 0`). Production ownership for the Ortsverband remains **self-hosted single host** (compose) per earlier agreement, with automated backup to ≥1 configurable destination (NFR-R3) and a documented restore procedure.
- **Frontend framework internals** (router, state, component library, styling) — pick within React+Vite+TS in the frontend epic; the invariants above don't depend on them.
- **Go routing/middleware implementation details** — chi is pinned (Stack) but per-route/per-module middleware composition is left to the epic; note the auth gateway and server-side policy (AD-2/AD-6) are wired at the composition root, not duplicated per epic.
- **Migration-tool rollback mechanics** — golang-migrate forward migrations are the contract (NFR-R2); detailed rollback procedure is documented per migration at build time.
- **Backup destination specifics** — automated pg backup to ≥1 configurable destination (env-driven) and a documented restore procedure are required (NFR-R3); the concrete destination (filesystem/object store) is deferred to the deploy epic/site.
- **CI/CD pipeline configuration** — required gates (lint NFR-M3, tests NFR-M2, dependency audit) are set; the provider and workflow files are deferred to the deploy epic.
- **TOTP/MFA library choice and email-send provider** — required by FR-4/FR-5 and FR-26 (forgot-password email); concrete libraries deferred to the User Directory epic. Capability shape fixed here: transactional SMTP sender for the single FR-26 email (AD-13, addendum "Password Management Decisions"), never automated notification emails (PRD §5).
- **Interactive diagram viewer** — the spine's mermaid diagrams should be click-to-enlarge with pan/zoom inside the enlarged view. This is deferred until the Docusaurus docs site exists (NFR-M4): implement there as a custom `@docusaurus/theme-mermaid` component (enlarge overlay + pan). No change needed now; the diagrams are authored as plain mermaid and stay renderer-agnostic.
