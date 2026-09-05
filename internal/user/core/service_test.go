package core

import (
	"context"
	"errors"
	"testing"
)

type mockRepo struct {
	users       map[string]*User
	createCalls int
	getCalls    int
	createErr   error
}

func newMockRepo() *mockRepo {
	return &mockRepo{users: make(map[string]*User)}
}

func (m *mockRepo) CreateRegisteredUser(_ context.Context, email, displayName, firstName, lastName, passwordHash string) (*User, error) {
	m.createCalls++
	if m.createErr != nil {
		return nil, m.createErr
	}
	u := &User{
		Email:        email,
		DisplayName:  displayName,
		FirstName:    firstName,
		LastName:     lastName,
		PasswordHash: passwordHash,
		State:        StatePendingApproval,
	}
	m.users[email] = u
	return u, nil
}

func (m *mockRepo) GetUserByEmail(_ context.Context, email string) (*User, error) {
	m.getCalls++
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

type mockHasher struct {
	hashCalls int
}

func (m *mockHasher) Hash(password string) (string, error) {
	m.hashCalls++
	return "hashed:" + password, nil
}

func (m *mockHasher) Verify(password, encodedHash string) (bool, error) {
	return encodedHash == "hashed:"+password, nil
}

func TestServiceRegisterHappyPath(t *testing.T) {
	repo := newMockRepo()
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	input := RegisterInput{
		FirstName:       "Erika",
		LastName:        "Musterfrau",
		Email:           "erika@example.com",
		Password:        "geheimespasswort123",
		PasswordConfirm: "geheimespasswort123",
	}

	res, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if res.Message != UniformSuccessMessage {
		t.Errorf("got message %q, want %q", res.Message, UniformSuccessMessage)
	}
	if res.Status != "pending_approval" {
		t.Errorf("got status %q, want %q", res.Status, "pending_approval")
	}

	if repo.createCalls != 1 {
		t.Errorf("expected 1 CreateRegisteredUser call, got %d", repo.createCalls)
	}
	if hasher.hashCalls != 1 {
		t.Errorf("expected 1 Hash call, got %d", hasher.hashCalls)
	}

	createdUser := repo.users["erika@example.com"]
	if createdUser == nil {
		t.Fatal("user was not saved in repo")
	}
	if createdUser.DisplayName != "Erika Musterfrau" {
		t.Errorf("displayName = %q, want %q", createdUser.DisplayName, "Erika Musterfrau")
	}
	if createdUser.State != StatePendingApproval {
		t.Errorf("state = %q, want %q", createdUser.State, StatePendingApproval)
	}
}

func TestServiceRegisterDuplicateEmailAntiEnumeration(t *testing.T) {
	repo := newMockRepo()
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	repo.users["existing@example.com"] = &User{
		Email: "existing@example.com",
		State: StateActive,
	}

	input := RegisterInput{
		FirstName:       "Max",
		LastName:        "Mustermann",
		Email:           "existing@example.com",
		Password:        "geheimespasswort123",
		PasswordConfirm: "geheimespasswort123",
	}

	res, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("Register failed for existing email: %v", err)
	}

	if res.Message != UniformSuccessMessage {
		t.Errorf("got message %q, want %q", res.Message, UniformSuccessMessage)
	}
	if res.Status != "pending_approval" {
		t.Errorf("got status %q, want %q", res.Status, "pending_approval")
	}

	// Should not have called create on repo, but should have hashed password for constant time
	if repo.createCalls != 0 {
		t.Errorf("expected 0 CreateRegisteredUser calls, got %d", repo.createCalls)
	}
	if hasher.hashCalls != 1 {
		t.Errorf("expected 1 dummy Hash call for timing protection, got %d", hasher.hashCalls)
	}
}

func TestServiceRegisterValidationErrors(t *testing.T) {
	repo := newMockRepo()
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	input := RegisterInput{
		FirstName:       "Max",
		LastName:        "Mustermann",
		Email:           "max@example.com",
		Password:        "short",
		PasswordConfirm: "short",
	}

	_, err := svc.Register(context.Background(), input)
	if !errors.Is(err, ErrShortPassword) {
		t.Errorf("expected ErrShortPassword, got: %v", err)
	}
	if repo.createCalls != 0 {
		t.Errorf("expected 0 create calls on validation failure, got %d", repo.createCalls)
	}
}

func TestServiceRegisterDuplicateKeyRaceCondition(t *testing.T) {
	repo := newMockRepo()
	repo.createErr = errors.New("ERROR: duplicate key value violates unique constraint \"users_email_key\" (SQLSTATE 23505)")
	hasher := &mockHasher{}
	svc := NewService(repo, hasher)

	input := RegisterInput{
		FirstName:       "Hans",
		LastName:        "Müller",
		Email:           "hans@example.com",
		Password:        "geheimespasswort123",
		PasswordConfirm: "geheimespasswort123",
	}

	res, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("expected duplicate key error to be swallowed into uniform confirmation, got: %v", err)
	}

	if res.Message != UniformSuccessMessage {
		t.Errorf("got message %q, want %q", res.Message, UniformSuccessMessage)
	}
	if res.Status != "pending_approval" {
		t.Errorf("got status %q, want %q", res.Status, "pending_approval")
	}
}
