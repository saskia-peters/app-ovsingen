package core

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
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
	// TotpCode is the optional 6-digit TOTP code (FR-4). It is omitted for the
	// first step of a two-step login when MFA is enabled.
	TotpCode string `json:"totp_code,omitempty"`
}

// Validate enforces the input bounds BEFORE the expensive Argon2id verify so
// an oversized payload cannot be used for a cheap CPU DoS. Only upper bounds
// are checked; empty values flow through to the uniform 401 anti-enumeration
// path and never leak account existence. A non-empty TotpCode must be exactly
// 6 digits (review finding 1.6-7) — enforced server-side, consistent with the
// client.
func (in *LoginInput) Validate() error {
	if utf8.RuneCountInString(in.Email) > 254 {
		return ErrInvalidLoginInput
	}
	if utf8.RuneCountInString(in.Password) > 1024 {
		return ErrInvalidLoginInput
	}
	if in.TotpCode != "" && !isValidTotpCodeFormat(in.TotpCode) {
		return ErrInvalidLoginInput
	}
	return nil
}

// LoginUser is the safe user snapshot returned on a successful login. The
// resolved permission set is deliberately NOT included: permissions are always
// re-derived server-side per request (AD-2/AD-6) and must never be client-trusted.
type LoginUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	DisplayName  string `json:"display_name"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	IsMFAEnabled bool   `json:"is_mfa_enabled"`
}

// LoginResult is the payload returned on successful login: an opaque session
// token plus the caller's user snapshot. When MFA is enabled and the client
// has not yet submitted a TOTP code, MFARequired is true and NO token is
// issued (two-step login, FR-4).
type LoginResult struct {
	Token       string    `json:"token,omitempty"`
	MFARequired bool      `json:"mfa_required,omitempty"`
	User        LoginUser `json:"user,omitempty"`
}

// Login authenticates a user with email + password, enforces the active
// account state and issues an opaque session token (AD-2/AD-6). When MFA is
// enabled (FR-4) the login is two-step: a valid password returns an MFA
// challenge (MFARequired, no token); only a subsequent login with a valid
// current TOTP code issues the session.
//
// Progressive lockout (FR-3): failures are tracked per normalized email —
// including unknown emails — so a blocked email is rejected with ErrLockedOut
// BEFORE the password verify, regardless of whether the presented password is
// correct or the account even exists. Because every probed email accumulates
// failures and can hit 429, a 429 is not discriminating (anti-enumeration).
// Outside lockout, all failures remain identical 401s (UX-DR7).
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
	now := time.Now().UTC()

	user, err := s.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user core: failed to look up user: %w", err)
	}

	// Lockout gate runs before the Argon2id verify (FR-3). Keyed on the
	// normalized email, so an email with enough accumulated failures is blocked
	// no matter what password (or whether the account) is presented.
	attempts, err := s.repo.GetLoginAttempts(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("user core: failed to read login attempts: %w", err)
	}
	if retryAfter, locked := Check(attempts, now); locked {
		return nil, NewLockoutError(retryAfter, email)
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
	if err != nil || !ok || !canAuthenticate {
		// Wrong password for any account state (or a non-existent account)
		// maps to the same error (UX-DR7). Every failure — including unknown
		// emails — is recorded against the normalized email so the counter and
		// lockout apply identically to every probed email (anti-enumeration).
		if err := s.repo.IncrementLoginAttempts(ctx, email); err != nil {
			return nil, fmt.Errorf("user core: failed to record login failure: %w", err)
		}
		return nil, ErrInvalidCredentials
	}

	// Successful login: clear the failure counter for a fresh cycle (FR-3).
	if err := s.repo.ClearLoginAttempts(ctx, email); err != nil {
		return nil, fmt.Errorf("user core: failed to clear login attempts: %w", err)
	}

	// Two-step MFA (FR-4): when MFA is enabled, a valid password alone must NOT
	// issue a session. The client first receives an MFA challenge (no token);
	// it then re-submits the credentials plus a current 6-digit TOTP code which
	// is validated against the stored (decrypted) secret before a session is
	// issued. A wrong/expired code is rejected with the identical 401 so the
	// challenge failure reveals nothing (UX-DR7); TOTP failures do not touch
	// the progressive password lockout (FR-3) since the password was valid.
	if user.IsMFAEnabled {
		if user.TotpSecretEncrypted == "" {
			// Defense-in-depth: an account flagged MFA without a stored secret
			// must never authenticate — fail closed.
			return nil, ErrInvalidCredentials
		}
		if input.TotpCode == "" {
			return &LoginResult{MFARequired: true}, nil
		}
		secret, err := s.decryptSecret(user.TotpSecretEncrypted)
		if err != nil {
			return nil, fmt.Errorf("user core: failed to decrypt TOTP secret: %w", err)
		}
		if !validTotpCode(secret, input.TotpCode) {
			return nil, ErrInvalidCredentials
		}
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
			ID:           user.ID,
			Email:        user.Email,
			DisplayName:  user.DisplayName,
			FirstName:    user.FirstName,
			LastName:     user.LastName,
			IsMFAEnabled: user.IsMFAEnabled,
		},
	}, nil
}

// Logout invalidates the given session token server-side (NFR-S2). Unknown or
// empty tokens are a no-op.
func (s *Service) Logout(ctx context.Context, rawToken string) error {
	return s.sessions.Invalidate(ctx, rawToken)
}