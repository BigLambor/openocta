package rbac

import (
	"testing"
	"time"
)

func TestAuthenticateUserUpgradesLegacyHash(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("legacy-pass")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	token, err := AuthenticateUser("admin", "legacy-pass")
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	if token == "" {
		t.Fatal("expected token")
	}
	rec, err := users.FindByUsername("admin")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if !IsArgon2Hash(rec.PasswordHash) {
		t.Fatalf("expected password hash upgraded to argon2, got %q", rec.PasswordHash)
	}
}

func TestCreateUserUsesArgon2Hash(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	if err := CreateUser("viewer1", "pass12345", 5); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	rec, err := users.FindByUsername("viewer1")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if !IsArgon2Hash(rec.PasswordHash) {
		t.Fatalf("expected argon2 hash for new user, got %q", rec.PasswordHash)
	}
}

func TestAuthenticateUserWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	token, err := AuthenticateUser("admin", "admin888")
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	session, err := ValidateToken(token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if session.Username != "admin" || session.RoleName != "admin" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if len(session.Permissions) == 0 {
		t.Fatal("expected permissions from memory role repository")
	}
}

func TestAuthenticateUserRejectsBadPasswordWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	if _, err := AuthenticateUser("admin", "wrong"); err == nil {
		t.Fatal("expected authentication failure")
	}
}

func TestValidateTokenExpiresWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	token, err := AuthenticateUser("admin", "admin888")
	if err != nil {
		t.Fatalf("AuthenticateUser: %v", err)
	}
	memTokens := tokens.(*memoryTokenRepository)
	memTokens.mu.Lock()
	rec := memTokens.tokens[token]
	rec.ExpiresAt = time.Now().Add(-time.Minute)
	memTokens.tokens[token] = rec
	memTokens.mu.Unlock()

	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected expired token error")
	}
}

func TestCreateUserAndListWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	if err := CreateUser("viewer1", "pass123", 5); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	list, err := ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected admin + viewer1, got %d users", len(list))
	}
}

func TestEnsureRolePermissionWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	if err := EnsureRolePermission(2, "menu:hadoop"); err != nil {
		t.Fatalf("EnsureRolePermission: %v", err)
	}
	perms, err := GetRolePermissions(2)
	if err != nil {
		t.Fatalf("GetRolePermissions: %v", err)
	}
	if len(perms) != 1 || perms[0] != "menu:hadoop" {
		t.Fatalf("unexpected permissions: %+v", perms)
	}
}

func TestDeleteUserRevokesTokensWithMemoryRepositories(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	if err := CreateUser("temp_user", "pass123", 5); err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	list, err := ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	var tempID int
	for _, u := range list {
		if u.Username == "temp_user" {
			tempID = u.ID
		}
	}
	token, err := AuthenticateUser("temp_user", "pass123")
	if err != nil {
		t.Fatalf("AuthenticateUser temp_user: %v", err)
	}
	if err := DeleteUser(tempID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := ValidateToken(token); err == nil {
		t.Fatal("expected token to be revoked after user delete")
	}
}
