package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/saskia-peters/gear/internal/user/core"
)

// Repository wraps the sqlc-generated Queries to implement core.Repository.
type Repository struct {
	queries *Queries
}

// NewRepository creates a new postgres user repository.
func NewRepository(queries *Queries) *Repository {
	return &Repository{queries: queries}
}

// CreateRegisteredUser inserts a new user record in pending_approval state.
func (r *Repository) CreateRegisteredUser(ctx context.Context, email, displayName, firstName, lastName, passwordHash string) (*core.User, error) {
	row, err := r.queries.CreateRegisteredUser(ctx, CreateRegisteredUserParams{
		Email:        email,
		DisplayName:  displayName,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return nil, err
	}

	return userFromRow(row.ID, row.Email, row.DisplayName, row.FirstName, row.LastName,
		row.PasswordHash, row.State, row.IsMfaEnabled, row.Attributes, row.CreatedAt, row.UpdatedAt), nil
}

// GetUserByEmail queries a user by their email address. If not found, returns nil, nil.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*core.User, error) {
	row, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return userFromRow(row.ID, row.Email, row.DisplayName, row.FirstName, row.LastName,
		row.PasswordHash, row.State, row.IsMfaEnabled, row.Attributes, row.CreatedAt, row.UpdatedAt), nil
}

// ListPermissionsByUser resolves the user's live permission set (AD-12):
// the additive union of permission-group memberships and direct grants.
func (r *Repository) ListPermissionsByUser(ctx context.Context, userID string) ([]string, error) {
	uid, err := uuidFromString(userID)
	if err != nil {
		return nil, err
	}
	return r.queries.ListPermissionsByUser(ctx, uid)
}

// CreateSession persists a new server-side session row and returns its domain
// representation (NFR-S2).
func (r *Repository) CreateSession(ctx context.Context, userID, tokenHash string, expiresAt time.Time) (*core.Session, error) {
	uid, err := uuidFromString(userID)
	if err != nil {
		return nil, err
	}
	row, err := r.queries.CreateSession(ctx, CreateSessionParams{
		UserID:    uid,
		TokenHash: tokenHash,
		ExpiresAt: pgtype.Timestamptz{Time: expiresAt, Valid: true},
	})
	if err != nil {
		return nil, err
	}
	return &core.Session{
		ID:        uuidToString(row.ID.Bytes),
		UserID:    uuidToString(row.UserID.Bytes),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
	}, nil
}

// GetSessionByTokenHash looks up a session by its hashed token, returning the
// associated user. Not found maps to core.ErrSessionNotFound.
func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenHash string) (*core.Session, error) {
	row, err := r.queries.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, core.ErrSessionNotFound
		}
		return nil, err
	}

	var attrs map[string]any
	if len(row.Attributes) > 0 {
		_ = json.Unmarshal(row.Attributes, &attrs)
	}

	return &core.Session{
		ID:        uuidToString(row.ID.Bytes),
		UserID:    uuidToString(row.UserID.Bytes),
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt.Time,
		CreatedAt: row.CreatedAt.Time,
		User: &core.User{
			ID:           uuidToString(row.UserID.Bytes),
			Email:        row.Email,
			DisplayName:  row.DisplayName,
			FirstName:    row.FirstName,
			LastName:     row.LastName,
			State:        core.UserState(row.State),
			IsMFAEnabled: row.IsMfaEnabled,
			Attributes:   attrs,
		},
	}, nil
}

// DeleteSessionByTokenHash removes a session row server-side by its hashed
// token (NFR-S2). Atomic: no Get-then-Delete window. Unknown hashes are a no-op.
func (r *Repository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	return r.queries.DeleteSessionByTokenHash(ctx, tokenHash)
}

func uuidToString(b [16]byte) string {
	return fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		b[0], b[1], b[2], b[3],
		b[4], b[5],
		b[6], b[7],
		b[8], b[9],
		b[10], b[11], b[12], b[13], b[14], b[15],
	)
}

// userFromRow maps an sqlc user row to the core.User domain entity.
func userFromRow(id pgtype.UUID, email, displayName, firstName, lastName, passwordHash, state string, isMfa bool, attributes []byte, createdAt, updatedAt pgtype.Timestamptz) *core.User {
	var attrs map[string]any
	if len(attributes) > 0 {
		_ = json.Unmarshal(attributes, &attrs)
	}
	return &core.User{
		ID:           uuidToString(id.Bytes),
		Email:        email,
		DisplayName:  displayName,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,
		State:        core.UserState(state),
		IsMFAEnabled: isMfa,
		Attributes:   attrs,
		CreatedAt:    createdAt.Time,
		UpdatedAt:    updatedAt.Time,
	}
}

// uuidFromString parses a canonical UUID string into a pgtype.UUID.
func uuidFromString(s string) (pgtype.UUID, error) {
	var b [16]byte
	_, err := fmt.Sscanf(s, "%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
		&b[0], &b[1], &b[2], &b[3],
		&b[4], &b[5],
		&b[6], &b[7],
		&b[8], &b[9],
		&b[10], &b[11], &b[12], &b[13], &b[14], &b[15],
	)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: b, Valid: true}, nil
}
