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

-- name: CreateRegisteredUser :one
INSERT INTO users (
    email,
    display_name,
    first_name,
    last_name,
    password_hash,
    state
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    'pending_approval'
)
RETURNING id, email, display_name, first_name, last_name, password_hash, state, is_mfa_enabled, attributes, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, display_name, first_name, last_name, password_hash, state, is_mfa_enabled, attributes, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING id, user_id, token_hash, expires_at, created_at;

-- name: GetSessionByTokenHash :one
SELECT s.id, s.user_id, s.token_hash, s.expires_at, s.created_at,
       u.email, u.display_name, u.first_name, u.last_name, u.state, u.is_mfa_enabled, u.attributes
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1;

-- name: DeleteSessionByTokenHash :exec
-- Atomic logout (NFR-S2): delete by the hashed token directly so there is no
-- Get-then-Delete TOCTOU window.
DELETE FROM sessions
WHERE token_hash = $1;

-- name: ListPermissionsByUser :many
-- Resolved permission set (AD-12): additive union of permission-group
-- memberships plus direct grants. Deduplicated via DISTINCT.
SELECT DISTINCT p.code
FROM permissions p
WHERE p.id IN (
    SELECT pgp.permission_id
    FROM user_permission_groups upg
    JOIN permission_group_permissions pgp ON pgp.permission_group_id = upg.permission_group_id
    WHERE upg.user_id = $1
    UNION
    SELECT up.permission_id
    FROM user_permissions up
    WHERE up.user_id = $1
)
ORDER BY p.code;

