package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

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

	var attrs map[string]any
	if len(row.Attributes) > 0 {
		_ = json.Unmarshal(row.Attributes, &attrs)
	}

	return &core.User{
		ID:           uuidToString(row.ID.Bytes),
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		PasswordHash: row.PasswordHash,
		State:        core.UserState(row.State),
		IsMFAEnabled: row.IsMfaEnabled,
		Attributes:   attrs,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
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

	var attrs map[string]any
	if len(row.Attributes) > 0 {
		_ = json.Unmarshal(row.Attributes, &attrs)
	}

	return &core.User{
		ID:           uuidToString(row.ID.Bytes),
		Email:        row.Email,
		DisplayName:  row.DisplayName,
		FirstName:    row.FirstName,
		LastName:     row.LastName,
		PasswordHash: row.PasswordHash,
		State:        core.UserState(row.State),
		IsMFAEnabled: row.IsMfaEnabled,
		Attributes:   attrs,
		CreatedAt:    row.CreatedAt.Time,
		UpdatedAt:    row.UpdatedAt.Time,
	}, nil
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
