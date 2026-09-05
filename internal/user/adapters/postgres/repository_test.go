package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/saskia-peters/gear/internal/user/core"
)

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
