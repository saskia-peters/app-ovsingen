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
