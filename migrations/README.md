# migrations

The **single** golang-migrate migration set — the one schema authority for the
whole application (NFR-R2, AD-11). Each table is owned by exactly one module;
each story ships its own incremental up/down pair and must not modify earlier
migrations.

| File | Story | Content |
|------|-------|---------|
| `000001_cold_start.up/down.sql` | 1.1 | User-owned identity tables the seeder needs, `admin.recovery.approve` permission, the four base roles and the two seeded admin accounts (AD-12/AD-13). |

Naming: `NNNNNN_snake_case.up.sql` / `NNNNNN_snake_case.down.sql`.

Apply from the root `justfile`: `just migrate-up` / `just migrate-down`.
sqlc generates the per-module stores from these forward migrations
(`just sqlc-generate`).