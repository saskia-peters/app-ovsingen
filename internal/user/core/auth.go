package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// ErrInvalidCredentials is the anti-enumeration error returned on ANY login
// failure — wrong password, unknown email, or non-active account (UX-DR7).
// Handlers map it to a single 401 with identical German microcopy so no
// account-existence information leaks.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrInvalidLoginInput is returned when a login payload violates the input
// bounds (oversized email/password). It maps to a 400; it never depends on
// account state so it leaks no account existence.
var ErrInvalidLoginInput = errors.New("invalid login input")

// MsgInvalidLoginInput is the German microcopy for oversized login input.
const MsgInvalidLoginInput = "Ungültige Anmeldedaten."

// dummyPasswordHash is a fixed Argon2id hash (identical cost to real hashes)
// used as the canonical verify target for accounts that must not authenticate
// (unknown email, pending/deactivated). Because the same verify always runs
// regardless of account state, response timing is uniform and account
// existence cannot be probed (UX-DR7). The placeholder password is never valid
// for any account.
const dummyPasswordHash = "$argon2id$v=19$m=65536,t=3,p=4$ZNrl3L8243LA/xK0x1A/qA$pkvED0l3kypbfEGoMEcitbmtK5sgiAAYYtWNy7BTias"

// LoginInput captures the login payload.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate enforces the input bounds BEFORE the expensive Argon2id verify so
// an oversized payload cannot be used for a cheap CPU DoS. Only upper bounds
// are checked; empty values flow through to the uniform 401 anti-enumeration
// path and never leak account existence.
func (in *LoginInput) Validate() error {
	if utf8.RuneCountInString(in.Email) > 254 {
		return ErrInvalidLoginInput
	}
	if utf8.RuneCountInString(in.Password) > 1024 {
		return ErrInvalidLoginInput
	}
	return nil
}

// LoginUser is the safe user snapshot returned on a successful login. The
// resolved permission set is deliberately NOT included: permissions are always
// re-derived server-side per request (AD-2/AD-6) and must never be client-trusted.
type LoginUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
}

// LoginResult is the payload returned on successful login: an opaque session
// token plus the caller's user snapshot.
type LoginResult struct {
	Token string    `json:"token"`
	User  LoginUser `json:"user"`
}

// Login authenticates a user with email + password, enforces the active
// account state and issues an opaque session token (AD-2/AD-6).
//
// Timing normalization (UX-DR7): exactly one password verify always runs,
// against the account's real hash for active users and against a fixed-cost
// dummy hash for every other combination, so unknown-email, non-active and
// wrong-password cases are indistinguishable. Every failure maps to
// ErrInvalidCredentials.
//
// Ordering: the live permission set is resolved BEFORE the session is issued so
// a permission-resolution failure cannot orphan a session row (NFR-S2).
func (s *Service) Login(ctx context.Context, input LoginInput) (*LoginResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(input.Email))

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user core: failed to look up user: %w", err)
	}

	// Single canonical verify: the account's real hash when the account can
	// authenticate (active AND has a stored hash), otherwise a fixed-cost dummy
	// hash. This keeps the verify cost identical on every code path (UX-DR7)
	// and never verifies against an empty/malformed hash (which would leak
	// timing for accounts without credentials).
	canAuthenticate := user != nil && user.State == StateActive && user.PasswordHash != ""
	targetHash := dummyPasswordHash
	if canAuthenticate {
		targetHash = user.PasswordHash
	}

	ok, err := s.hasher.Verify(input.Password, targetHash)
	if err != nil || !ok {
		// Wrong password for any account state (or a non-existent account)
		// maps to the same error (UX-DR7).
		return nil, ErrInvalidCredentials
	}

	// The verify only passes against the account's real hash when
	// canAuthenticate; any other combination (unknown email, pending/
	// deactivated, or account without a stored hash) is rejected with the
	// identical error so account existence never leaks (FR-5/FR-21).
	if !canAuthenticate {
		return nil, ErrInvalidCredentials
	}

	// Resolve the permission set before creating the session: a failure here
	// must not leave an orphaned session behind.
	if _, err := s.repo.ListPermissionsByUser(ctx, user.ID); err != nil {
		return nil, fmt.Errorf("user core: failed to resolve permissions: %w", err)
	}

	token, err := s.sessions.Issue(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("user core: failed to issue session: %w", err)
	}

	return &LoginResult{
		Token: token,
		User: LoginUser{
			ID:          user.ID,
			Email:       user.Email,
			DisplayName: user.DisplayName,
			FirstName:   user.FirstName,
			LastName:    user.LastName,
		},
	}, nil
}

// Logout invalidates the given session token server-side (NFR-S2). Unknown or
// empty tokens are a no-op.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.sessions.Invalidate(ctx, rawToken)
}