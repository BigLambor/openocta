package rbac

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

func TestLoginGuardLocksAfterRepeatedFailures(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	t.Setenv("OPENOCTA_LOGIN_MAX_ATTEMPTS", "3")
	t.Setenv("OPENOCTA_LOGIN_LOCKOUT_MINUTES", "1")

	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	ip := "203.0.113.10"
	username := "locked-user"

	for i := 0; i < 3; i++ {
		status, err := CheckLoginAllowed(ip, username)
		if err != nil {
			t.Fatalf("CheckLoginAllowed #%d: %v", i, err)
		}
		if !status.Allowed {
			t.Fatalf("expected attempt %d to be allowed before lockout", i)
		}
		if _, err := RecordLoginFailure(ip, username); err != nil {
			t.Fatalf("RecordLoginFailure #%d: %v", i, err)
		}
	}

	status, err := CheckLoginAllowed(ip, username)
	if err != nil {
		t.Fatalf("CheckLoginAllowed after lock: %v", err)
	}
	if status.Allowed {
		t.Fatal("expected login to be locked")
	}
	if status.LockedUntil.Before(time.Now()) {
		t.Fatalf("expected future lockedUntil, got %v", status.LockedUntil)
	}
}

func TestLoginGuardClearsAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	t.Setenv("OPENOCTA_LOGIN_MAX_ATTEMPTS", "5")

	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	ip := "203.0.113.11"
	username := "success-user"
	if _, err := RecordLoginFailure(ip, username); err != nil {
		t.Fatalf("RecordLoginFailure: %v", err)
	}
	if err := RecordLoginSuccess(ip, username); err != nil {
		t.Fatalf("RecordLoginSuccess: %v", err)
	}
	status, err := CheckLoginAllowed(ip, username)
	if err != nil {
		t.Fatalf("CheckLoginAllowed: %v", err)
	}
	if !status.Allowed {
		t.Fatalf("expected lockout to be cleared, got %+v", status)
	}
}

func TestCleanupExpiredSessionsRemovesOldTokens(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })

	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}
	if _, err := SetupInitialAdmin("admin", "SecurePass1"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	tokens, err := requireTokenRepo()
	if err != nil {
		t.Fatalf("requireTokenRepo: %v", err)
	}
	if err := tokens.Create("expired-token", 1, time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("Create expired token: %v", err)
	}
	if err := tokens.Create("active-token", 1, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("Create active token: %v", err)
	}

	removed, err := CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("CleanupExpiredSessions: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 expired token removed, got %d", removed)
	}
	if _, err := ValidateToken("expired-token"); err == nil {
		t.Fatal("expected expired token to be invalid")
	}
	if _, err := ValidateToken("active-token"); err != nil {
		t.Fatalf("expected active token to remain valid: %v", err)
	}
}

func TestInvalidateAllSessionsKeepsCurrentToken(t *testing.T) {
	users, roles, tokens := newMemoryRBACStores("admin888")
	SetRepositoriesForTest(users, roles, tokens)
	t.Cleanup(ResetRepositoriesForTest)

	tokenA, err := AuthenticateUser("admin", "admin888")
	if err != nil {
		t.Fatalf("AuthenticateUser A: %v", err)
	}
	tokenB, err := AuthenticateUser("admin", "admin888")
	if err != nil {
		t.Fatalf("AuthenticateUser B: %v", err)
	}

	removed, err := InvalidateAllSessions(1, tokenB)
	if err != nil {
		t.Fatalf("InvalidateAllSessions: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected 1 removed session, got %d", removed)
	}
	if _, err := ValidateToken(tokenA); err == nil {
		t.Fatal("expected tokenA to be revoked")
	}
	if _, err := ValidateToken(tokenB); err != nil {
		t.Fatalf("expected tokenB to remain valid: %v", err)
	}
}

func TestAuthSecurityMigrationApplied(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	var name string
	err := openoctadb.GetDB().QueryRow(`SELECT name FROM schema_migrations WHERE version = 6`).Scan(&name)
	if err != nil {
		t.Fatalf("expected migration 006 applied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "openocta.db")); err != nil {
		t.Fatalf("openocta.db missing: %v", err)
	}
}
