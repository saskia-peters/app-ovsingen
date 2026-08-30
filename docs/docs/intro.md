---
sidebar_position: 1
---

# THW OV Singen App — Documentation

The **THW OV Singen App** modernizes and centralizes the operational equipment management of the THW Ortsverband Singen: a modularized web application for mobile and desktop devices.

This documentation site is generated from the BMad planning artifacts (PRD, product brief, architecture spine) and will grow with the project: API contracts and module integration points are published here per NFR-M4.

## Planning documents

- [PRD — THW OV Singen App](/docs/planning/prd) — functional requirements, epics, and non-functional guidelines
- [Product Brief](/docs/planning/product-brief) — scope, objectives, and high-level requirements
- [Architecture Spine](/docs/planning/architecture-spine) — technical invariants, ADs, database artifacts, diagrams
- [Architecture Addendum](/docs/planning/addendum) — technology stack decisions and deferred options

## Stack

| Layer | Technology |
|---|---|
| Backend | Go (chi, sqlc/pgx, golang-migrate) |
| Frontend | React + Vite + TypeScript SPA |
| Database | PostgreSQL |
| Docs | Docusaurus (this site) |
| IaC | OpenTofu (Google Cloud Run testing/staging) |