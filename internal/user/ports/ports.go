// Package ports declares the inbound/outbound port interfaces of the User
// Directory & Auth hexagon (AD-1/AD-2).
package ports

import (
	"context"

	"github.com/saskia-peters/gear/internal/user/core"
)

// RegisterResult is the anti-enumeration confirmation returned on registration.
type RegisterResult = core.RegisterResult

// Service is the User Directory & Auth inbound port (AD-2).
type Service interface {
	Register(ctx context.Context, input core.RegisterInput) (*core.RegisterResult, error)
}

// Repository is the outbound persistence port for User data.
type Repository interface {
	CreateRegisteredUser(ctx context.Context, email, displayName, firstName, lastName, passwordHash string) (*core.User, error)
	GetUserByEmail(ctx context.Context, email string) (*core.User, error)
}

// PasswordHasher is the outbound password hashing port (AD-13).
type PasswordHasher interface {
	Hash(password string) (string, error)
	Verify(password, encodedHash string) (bool, error)
}
