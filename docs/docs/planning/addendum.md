---
sidebar_position: 4
---
# Addendum: THW OV Singen App

*This document preserves technical decisions, options considered, and implementation context that informs downstream work (architecture, solution design) but does not belong in the PRD itself.*

---

## Technology Stack Decisions

These choices were made by the product owner and recorded here for downstream architecture and development work.

| Layer | Technology | Notes |
|---|---|---|
| **Backend** | Go | Statically typed, high-performance, strong stdlib for HTTP servers |
| **Frontend** | TypeScript | Type safety for the web client; framework to be determined in architecture phase |
| **Linting** | prek | Enforced in CI pipeline (see NFR-M3) |
| **Code Storage** | GitHub | Source of truth for all code; CI/CD pipeline anchored here |
| **Documentation** | Docusaurus | Project documentation site; API contracts and module integration points published here (see NFR-M4) |

> **Note for architecture phase:** Framework choices within Go (e.g. routing library, ORM, migration tool) and TypeScript (e.g. React vs. Vue, component library) are deferred to the architecture document. The PRD's NFRs (modularity, test coverage, linting, structured logging, DB migrations) constrain those choices but do not prescribe them.

---

## Password Management Decisions

Added to support FR-25, FR-26, and FR-27 (self-service password management and dual-admin recovery).

| Concern | Decision | Notes |
|---|---|---|
| Password storage | Argon2id (memory-hard) via a maintained Go library | Never store passwords in plaintext or reversible form; hashing is out of band from the app's log data. Exact library pinned in the User epic. |
| Email sending | **SMTP-compatible transactional sender** for V1; a SaaS provider is a possible later swap. **Delivery parameters are admin-configurable in the UI (FR-28)** via an email-settings table owned by the User module, exposed through the User module's config port, and edited from the Admin panel | Host, port, security (none/STARTTLS/TLS), sender & display name, username, and password are editable at runtime with no redeploy. The **password is stored encrypted at rest** (app-level key from env/secret-manager, NFR-S4) and is write-only/masked in the UI. Includes a "send test email" action. Deferred: exact library/provider to the User epic. Used **only** for the FR-26 transactional reset email; no automated/notification emails in V1 (PRD §5). |
| Reset tokens | Cryptographically random, high-entropy, hashed-at-rest, single-use, 30-minute expiry | One active token per user (new request invalidates older ones). Token hash stored, not the plaintext. |
| Admin bootstrap | **Two pre-seeded `admin` accounts**, credentials generated and distributed out-of-band | Neither stored in VCS (NFR-S4). Dual-control recovery, FR-27. |
| Dual-control recovery | Second admin authorizes recovery of a locked-out/forgotten admin; recovery of the last admin is a documented manual out-of-band procedure with a mandatory audit entry | Prevents single-account takeover; FR-27. |
