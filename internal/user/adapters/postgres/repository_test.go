package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saskia-peters/gear/internal/user/core"
)

func TestPostgresLoginAttemptsRepository(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://gear:gear@localhost:5432/gear?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping db integration test: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db integration test (db ping failed): %v", err)
	}

	queries := New(pool)
	repo := NewRepository(queries)

	// login_attempts is keyed by email with no users FK, so an unknown email
	// can be tracked too (anti-enumeration).
	testEmail := "lockout.test." + time.Now().Format("20060102150405.000000") + "@gear.local"

	// 1. A fresh email has no attempts record.
	att, err := repo.GetLoginAttempts(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetLoginAttempts(fresh) failed: %v", err)
	}
	if att != nil {
		t.Fatalf("expected nil attempts for fresh email, got %+v", att)
	}

	// 2. Three atomic increments cross the 30s threshold (FR-3).
	for i := 0; i < 3; i++ {
		if err := repo.IncrementLoginAttempts(ctx, testEmail); err != nil {
			t.Fatalf("IncrementLoginAttempts (%d) failed: %v", i, err)
		}
	}
	att, err = repo.GetLoginAttempts(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetLoginAttempts failed: %v", err)
	}
	if att.Email != testEmail {
		t.Errorf("attempt email = %q, want %q", att.Email, testEmail)
	}
	if att.FailedCount != 3 {
		t.Errorf("attempt failed count = %d, want 3", att.FailedCount)
	}
	want := time.Now().UTC().Add(30 * time.Second)
	if att.LockoutUntil.IsZero() || att.LockoutUntil.Before(want.Add(-2*time.Second)) || att.LockoutUntil.After(want.Add(2*time.Second)) {
		t.Errorf("attempt lockout_until = %v, want ~now+30s", att.LockoutUntil)
	}

	// 3. A 4th increment escalates to the 60s window (LOCKOUT_4_PLUS).
	if err := repo.IncrementLoginAttempts(ctx, testEmail); err != nil {
		t.Fatalf("IncrementLoginAttempts (4th) failed: %v", err)
	}
	att, err = repo.GetLoginAttempts(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetLoginAttempts failed: %v", err)
	}
	if att.FailedCount != 4 {
		t.Errorf("attempt failed count = %d, want 4", att.FailedCount)
	}
	want = time.Now().UTC().Add(60 * time.Second)
	if att.LockoutUntil.IsZero() || att.LockoutUntil.Before(want.Add(-2*time.Second)) || att.LockoutUntil.After(want.Add(2*time.Second)) {
		t.Errorf("attempt lockout_until = %v, want ~now+60s", att.LockoutUntil)
	}

	// 4. The counter is capped (LockoutMaxFailedCount = 10).
	for i := 0; i < 10; i++ {
		if err := repo.IncrementLoginAttempts(ctx, testEmail); err != nil {
			t.Fatalf("IncrementLoginAttempts (cap) failed: %v", err)
		}
	}
	att, err = repo.GetLoginAttempts(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetLoginAttempts failed: %v", err)
	}
	if att.FailedCount != 10 {
		t.Errorf("attempt failed count = %d, want capped at 10", att.FailedCount)
	}

	// 5. ClearLoginAttempts resets the counter and window for a fresh cycle.
	if err := repo.ClearLoginAttempts(ctx, testEmail); err != nil {
		t.Fatalf("ClearLoginAttempts failed: %v", err)
	}
	att, err = repo.GetLoginAttempts(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetLoginAttempts after clear failed: %v", err)
	}
	if att == nil {
		t.Fatal("expected attempts row to remain after clear (reset to 0)")
	}
	if att.FailedCount != 0 {
		t.Errorf("attempt failed count = %d, want 0", att.FailedCount)
	}
	if !att.LockoutUntil.IsZero() {
		t.Errorf("attempt lockout_until = %v, want zero after clear", att.LockoutUntil)
	}
}

func TestPostgresRepository(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://gear:gear@localhost:5432/gear?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping db integration test: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db integration test (db ping failed): %v", err)
	}

	queries := New(pool)
	repo := NewRepository(queries)

	testEmail := "test.user." + time.Now().Format("20060102150405.000000") + "@gear.local"

	// 1. Check user does not exist initially
	existing, err := repo.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if existing != nil {
		t.Fatalf("expected nil user, got %+v", existing)
	}

	// 2. Create registered user
	created, err := repo.CreateRegisteredUser(ctx, testEmail, "Test User", "Test", "User", "$argon2id$v=19$dummyhash")
	if err != nil {
		t.Fatalf("CreateRegisteredUser failed: %v", err)
	}
	if created.Email != testEmail {
		t.Errorf("created.Email = %q, want %q", created.Email, testEmail)
	}
	if created.State != core.StatePendingApproval {
		t.Errorf("created.State = %q, want %q", created.State, core.StatePendingApproval)
	}
	if created.FirstName != "Test" || created.LastName != "User" {
		t.Errorf("created names = (%q, %q), want (Test, User)", created.FirstName, created.LastName)
	}

	// 3. Query newly created user
	fetched, err := repo.GetUserByEmail(ctx, testEmail)
	if err != nil {
		t.Fatalf("GetUserByEmail failed: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected user to be found, got nil")
	}
	if fetched.Email != testEmail {
		t.Errorf("fetched.Email = %q, want %q", fetched.Email, testEmail)
	}
	if fetched.State != core.StatePendingApproval {
		t.Errorf("fetched.State = %q, want %q", fetched.State, core.StatePendingApproval)
	}
}

func TestPostgresSessionAndPermissionRepository(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://gear:gear@localhost:5432/gear?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skipf("skipping db integration test: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Skipf("skipping db integration test (db ping failed): %v", err)
	}

	queries := New(pool)
	repo := NewRepository(queries)

	testEmail := "auth.test." + time.Now().Format("20060102150405.000000") + "@gear.local"

	created, err := repo.CreateRegisteredUser(ctx, testEmail, "Auth Test", "Auth", "Test", "$argon2id$v=19$m=65536,t=3,p=4$c2FsdHNhbHRzYWx0$8U3f5yO8JUpfGT5WmljHhL8n2nWlVEhL2fj7EXpS9gM")
	if err != nil {
		t.Fatalf("CreateRegisteredUser failed: %v", err)
	}
	if created.State != core.StatePendingApproval {
		t.Fatalf("created.State = %q, want pending_approval", created.State)
	}

	// 1. CreateSession + GetSessionByTokenHash round-trip.
	expiry := time.Now().UTC().Add(time.Hour)
	sess, err := repo.CreateSession(ctx, created.ID, "hash-of-raw-token", expiry)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if sess.UserID != created.ID {
		t.Errorf("session user id = %q, want %q", sess.UserID, created.ID)
	}

	fetched, err := repo.GetSessionByTokenHash(ctx, "hash-of-raw-token")
	if err != nil {
		t.Fatalf("GetSessionByTokenHash failed: %v", err)
	}
	if fetched.ID != sess.ID {
		t.Errorf("fetched session id = %q, want %q", fetched.ID, sess.ID)
	}
	if fetched.User == nil || fetched.User.ID != created.ID {
		t.Errorf("fetched session user = %+v, want attached user %q", fetched.User, created.ID)
	}

	// Unknown hash maps to core.ErrSessionNotFound.
	if _, err := repo.GetSessionByTokenHash(ctx, "no-such-hash"); err != core.ErrSessionNotFound {
		t.Errorf("unknown token error = %v, want core.ErrSessionNotFound", err)
	}

	// 2. ListPermissionsByUser for a fresh user resolves to the empty set.
	perms, err := repo.ListPermissionsByUser(ctx, created.ID)
	if err != nil {
		t.Fatalf("ListPermissionsByUser failed: %v", err)
	}
	if len(perms) != 0 {
		t.Errorf("permissions = %v, want empty set", perms)
	}

	// 3. The seeded admin resolves the admin.recovery.approve permission
	// (AD-12 additive union) and the admin group.
	admin, err := repo.GetUserByEmail(ctx, "admin.1@gear.local")
	if err != nil {
		t.Fatalf("GetUserByEmail(admin) failed: %v", err)
	}
	if admin == nil {
		t.Skip("seeded admin not present — skipping permission resolution assertion")
	}
	adminPerms, err := repo.ListPermissionsByUser(ctx, admin.ID)
	if err != nil {
		t.Fatalf("ListPermissionsByUser(admin) failed: %v", err)
	}
	found := false
	for _, p := range adminPerms {
		if p == "admin.recovery.approve" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("admin permissions %v missing admin.recovery.approve", adminPerms)
	}

	// 4. DeleteSessionByTokenHash invalidates the session server-side,
	// atomically by hashed token.
	if err := repo.DeleteSessionByTokenHash(ctx, "hash-of-raw-token"); err != nil {
		t.Fatalf("DeleteSessionByTokenHash failed: %v", err)
	}
	if _, err := repo.GetSessionByTokenHash(ctx, "hash-of-raw-token"); err != core.ErrSessionNotFound {
		t.Errorf("deleted token error = %v, want core.ErrSessionNotFound", err)
	}
}
