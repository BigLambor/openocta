package rbac

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	openoctadb "github.com/openocta/openocta/pkg/db"
	_ "modernc.org/sqlite"
)

func initUnifiedRBAC(t *testing.T, dir string) {
	t.Helper()
	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("rbac.InitDB: %v", err)
	}
}

func createLegacyRBACDB(t *testing.T, dir string, seed func(*sql.DB)) {
	t.Helper()
	legacyPath := filepath.Join(dir, "rbac.db")
	legacyDB, err := sql.Open("sqlite", legacyPath)
	if err != nil {
		t.Fatalf("open legacy rbac.db: %v", err)
	}
	defer legacyDB.Close()

	_, err = legacyDB.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			role_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
		CREATE TABLE roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT
		);
		CREATE TABLE permissions (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL
		);
		CREATE TABLE role_permissions (
			role_id INTEGER NOT NULL,
			permission_code TEXT NOT NULL,
			PRIMARY KEY (role_id, permission_code)
		);
		CREATE TABLE user_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create legacy tables: %v", err)
	}
	if seed != nil {
		seed(legacyDB)
	}
}

func TestInitDBSeedsRolesWithoutDefaultUsers(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	initUnifiedRBAC(t, dir)

	if _, err := os.Stat(filepath.Join(dir, "rbac.db")); !os.IsNotExist(err) {
		t.Fatalf("expected no standalone rbac.db on fresh install, stat err=%v", err)
	}

	needs, err := NeedsSetup()
	if err != nil {
		t.Fatalf("NeedsSetup: %v", err)
	}
	if !needs {
		t.Fatal("expected fresh install to require setup")
	}

	if _, err := AuthenticateUser("admin", "admin888"); err == nil {
		t.Fatal("expected default admin login to be unavailable before setup")
	}

	roles, err := ListRoles()
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("expected default roles to be seeded")
	}
}

func TestSetupInitialAdminCreatesArgon2User(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	initUnifiedRBAC(t, dir)

	if _, err := SetupInitialAdmin("admin", "MySecurePass1"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	rec, err := requireUserRepo()
	if err != nil {
		t.Fatalf("requireUserRepo: %v", err)
	}
	user, err := rec.FindByUsername("admin")
	if err != nil {
		t.Fatalf("FindByUsername: %v", err)
	}
	if !IsArgon2Hash(user.PasswordHash) {
		t.Fatalf("expected argon2 hash after setup, got %q", user.PasswordHash)
	}

	token, err := AuthenticateUser("admin", "MySecurePass1")
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
}

func TestInitDBMigratesLegacyRBACDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })

	customSalt := "legacy-salt"
	customHash := HashPassword("custom-pass", customSalt)
	createLegacyRBACDB(t, dir, func(legacy *sql.DB) {
		_, err := legacy.Exec(`INSERT INTO roles (id, name, description) VALUES (1, 'admin', 'legacy admin')`)
		if err != nil {
			t.Fatalf("seed role: %v", err)
		}
		_, err = legacy.Exec(`INSERT INTO permissions (code, name, type) VALUES ('menu:overview', 'Overview', 'menu')`)
		if err != nil {
			t.Fatalf("seed permission: %v", err)
		}
		_, err = legacy.Exec(`INSERT INTO role_permissions (role_id, permission_code) VALUES (1, 'menu:overview')`)
		if err != nil {
			t.Fatalf("seed role permission: %v", err)
		}
		_, err = legacy.Exec(`
			INSERT INTO users (id, username, password_hash, salt, role_id)
			VALUES (1, 'legacy_admin', ?, ?, 1)
		`, customHash, customSalt)
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}
		expires := time.Now().Add(2 * time.Hour).Format(time.RFC3339)
		_, err = legacy.Exec(`INSERT INTO user_tokens (token, user_id, expires_at) VALUES ('legacy-token', 1, ?)`, expires)
		if err != nil {
			t.Fatalf("seed token: %v", err)
		}
	})

	initUnifiedRBAC(t, dir)

	if _, err := os.Stat(filepath.Join(dir, "rbac.db")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy rbac.db to be moved after migration, stat err=%v", err)
	}

	token, err := AuthenticateUser("legacy_admin", "custom-pass")
	if err != nil {
		t.Fatalf("AuthenticateUser legacy_admin: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	session, err := ValidateToken("legacy-token")
	if err != nil {
		t.Fatalf("ValidateToken legacy-token: %v", err)
	}
	if session.Username != "legacy_admin" {
		t.Fatalf("unexpected migrated session user: %s", session.Username)
	}

	var adminCount int
	if err := openoctadb.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, "admin").Scan(&adminCount); err != nil {
		t.Fatalf("count default admin: %v", err)
	}
	if adminCount != 0 {
		t.Fatalf("expected migrated install not to re-seed default admin, got count=%d", adminCount)
	}
}

func TestInitDBMigratesCustomRoles(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })

	createLegacyRBACDB(t, dir, func(legacy *sql.DB) {
		_, err := legacy.Exec(`INSERT INTO roles (id, name, description) VALUES (9, 'custom_auditor', 'Custom role')`)
		if err != nil {
			t.Fatalf("seed custom role: %v", err)
		}
		_, err = legacy.Exec(`INSERT INTO permissions (code, name, type) VALUES ('menu:config', 'Config', 'menu')`)
		if err != nil {
			t.Fatalf("seed permission: %v", err)
		}
		_, err = legacy.Exec(`INSERT INTO role_permissions (role_id, permission_code) VALUES (9, 'menu:config')`)
		if err != nil {
			t.Fatalf("seed role permission: %v", err)
		}
		salt := "auditor-salt"
		hash := HashPassword("auditor-pass", salt)
		_, err = legacy.Exec(`
			INSERT INTO users (id, username, password_hash, salt, role_id)
			VALUES (2, 'auditor', ?, ?, 9)
		`, hash, salt)
		if err != nil {
			t.Fatalf("seed auditor user: %v", err)
		}
	})

	initUnifiedRBAC(t, dir)

	roles, err := ListRoles()
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	found := false
	for _, role := range roles {
		if role.Name == "custom_auditor" {
			found = true
			if role.ID != 9 {
				t.Fatalf("expected custom role id=9, got %d", role.ID)
			}
		}
	}
	if !found {
		t.Fatalf("expected custom role to be migrated, got roles=%+v", roles)
	}

	perms, err := GetRolePermissions(9)
	if err != nil {
		t.Fatalf("GetRolePermissions: %v", err)
	}
	if len(perms) != 1 || perms[0] != "menu:config" {
		t.Fatalf("unexpected custom role permissions: %+v", perms)
	}

	if _, err := AuthenticateUser("auditor", "auditor-pass"); err != nil {
		t.Fatalf("AuthenticateUser auditor: %v", err)
	}
}

func TestInitDBMigrationIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })

	createLegacyRBACDB(t, dir, func(legacy *sql.DB) {
		_, err := legacy.Exec(`INSERT INTO roles (id, name, description) VALUES (1, 'admin', 'legacy admin')`)
		if err != nil {
			t.Fatalf("seed role: %v", err)
		}
		salt := "legacy-salt"
		hash := HashPassword("custom-pass", salt)
		_, err = legacy.Exec(`
			INSERT INTO users (id, username, password_hash, salt, role_id)
			VALUES (1, 'legacy_admin', ?, ?, 1)
		`, hash, salt)
		if err != nil {
			t.Fatalf("seed user: %v", err)
		}
	})

	initUnifiedRBAC(t, dir)
	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("second db.InitDB: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("second rbac.InitDB: %v", err)
	}

	var userCount int
	if err := openoctadb.GetDB().QueryRow(`SELECT COUNT(*) FROM users WHERE username = ?`, "legacy_admin").Scan(&userCount); err != nil {
		t.Fatalf("count migrated user: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected one migrated user after re-init, got %d", userCount)
	}
}

func TestInitDBRequiresUnifiedDatabase(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	if err := InitDB(dir); err == nil {
		t.Fatal("expected InitDB to fail when openocta.db is not initialized")
	}
}
