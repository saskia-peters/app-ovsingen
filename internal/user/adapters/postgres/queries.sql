-- User Directory & Auth module store (AD-2), generated into package postgres
-- by sqlc. These are the building blocks the auth port resolves identity and
-- permission sets from (AD-12: additive union of permission-group memberships
-- + direct grants). Story 1.1 ships the toolchain plus the first queries;
-- later stories extend this file and re-run `just sqlc-generate`.

-- name: GetPermissionByCode :one
SELECT id, code, description
FROM permissions
WHERE code = $1;

-- name: ListPermissionGroupsByUser :many
SELECT pg.id, pg.name, pg.description, pg.is_base_role
FROM permission_groups pg
JOIN user_permission_groups upg ON upg.permission_group_id = pg.id
WHERE upg.user_id = $1
ORDER BY pg.name;