package rbac

import (
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

// GetDB returns the unified openocta.db connection used for RBAC migrations and legacy tooling.
func GetDB() *sql.DB {
	return sqlDB
}

// InitDB attaches RBAC to openocta.db, migrates legacy rbac.db once, and seeds defaults when empty.
// db.InitDB must be called before InitDB.
func InitDB(stateDir string) error {
	unified := openoctadb.GetDB()
	if unified == nil {
		return fmt.Errorf("openocta.db 未初始化，请先调用 db.InitDB")
	}
	initRepositories(unified)

	if err := migrateLegacyRBACDB(stateDir, unified); err != nil {
		return err
	}
	if err := seedDefaultData(); err != nil {
		return err
	}
	StartSessionJanitor()
	return nil
}

func migrateLegacyRBACDB(stateDir string, target *sql.DB) error {
	if target == nil {
		return fmt.Errorf("nil database")
	}
	legacyPath := filepath.Join(stateDir, "rbac.db")
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var userCount int
	if userRepo != nil {
		var countErr error
		userCount, countErr = userRepo.Count()
		if countErr != nil {
			return countErr
		}
	} else if err := target.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&userCount); err != nil {
		return err
	}
	if userCount > 0 {
		return backupLegacyRBACDB(legacyPath)
	}

	legacyDB, err := sql.Open("sqlite", legacyPath+"?_pragma=busy_timeout(5000)&mode=ro")
	if err != nil {
		return err
	}
	defer legacyDB.Close()
	if err := legacyDB.Ping(); err != nil {
		return err
	}

	tx, err := target.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := copyLegacyRoles(legacyDB, tx); err != nil {
		return err
	}
	if err := copyLegacyPermissions(legacyDB, tx); err != nil {
		return err
	}
	if err := copyLegacyRolePermissions(legacyDB, tx); err != nil {
		return err
	}
	if err := copyLegacyUsers(legacyDB, tx); err != nil {
		return err
	}
	if err := copyLegacyUserTokens(legacyDB, tx); err != nil {
		return err
	}
	if err := syncSQLiteSequence(tx, "roles"); err != nil {
		return err
	}
	if err := syncSQLiteSequence(tx, "users"); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return backupLegacyRBACDB(legacyPath)
}

func copyLegacyRoles(legacy *sql.DB, tx *sql.Tx) error {
	rows, err := legacy.Query(`SELECT id, name, description FROM roles ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name, description sql.NullString
		if err := rows.Scan(&id, &name, &description); err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO roles (id, name, description) VALUES (?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				name = excluded.name,
				description = excluded.description
		`, id, name.String, description.String)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyLegacyPermissions(legacy *sql.DB, tx *sql.Tx) error {
	rows, err := legacy.Query(`SELECT code, name, type FROM permissions ORDER BY code`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var code, name, ptype string
		if err := rows.Scan(&code, &name, &ptype); err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO permissions (code, name, type) VALUES (?, ?, ?)
			ON CONFLICT(code) DO UPDATE SET
				name = excluded.name,
				type = excluded.type
		`, code, name, ptype)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyLegacyRolePermissions(legacy *sql.DB, tx *sql.Tx) error {
	rows, err := legacy.Query(`SELECT role_id, permission_code FROM role_permissions ORDER BY role_id, permission_code`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var roleID int
		var code string
		if err := rows.Scan(&roleID, &code); err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)
		`, roleID, code)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyLegacyUsers(legacy *sql.DB, tx *sql.Tx) error {
	rows, err := legacy.Query(`SELECT id, username, password_hash, salt, role_id, created_at FROM users ORDER BY id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id, roleID int
		var username, passwordHash, salt string
		var createdAt sql.NullString
		if err := rows.Scan(&id, &username, &passwordHash, &salt, &roleID, &createdAt); err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO users (id, username, password_hash, salt, role_id, created_at)
			VALUES (?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
			ON CONFLICT(id) DO UPDATE SET
				username = excluded.username,
				password_hash = excluded.password_hash,
				salt = excluded.salt,
				role_id = excluded.role_id,
				created_at = excluded.created_at
		`, id, username, passwordHash, salt, roleID, nullStringValue(createdAt))
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func copyLegacyUserTokens(legacy *sql.DB, tx *sql.Tx) error {
	rows, err := legacy.Query(`SELECT token, user_id, expires_at FROM user_tokens ORDER BY token`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var token string
		var userID int
		var expiresAt string
		if err := rows.Scan(&token, &userID, &expiresAt); err != nil {
			return err
		}
		_, err = tx.Exec(`
			INSERT INTO user_tokens (token, user_id, expires_at) VALUES (?, ?, ?)
			ON CONFLICT(token) DO UPDATE SET
				user_id = excluded.user_id,
				expires_at = excluded.expires_at
		`, token, userID, expiresAt)
		if err != nil {
			return err
		}
	}
	return rows.Err()
}

func syncSQLiteSequence(tx *sql.Tx, table string) error {
	var maxID sql.NullInt64
	if err := tx.QueryRow(fmt.Sprintf(`SELECT MAX(id) FROM %s`, table)).Scan(&maxID); err != nil {
		return err
	}
	if !maxID.Valid || maxID.Int64 <= 0 {
		return nil
	}
	_, err := tx.Exec(`UPDATE sqlite_sequence SET seq = ? WHERE name = ?`, maxID.Int64, table)
	if err != nil {
		_, err = tx.Exec(`INSERT INTO sqlite_sequence (name, seq) VALUES (?, ?)`, table, maxID.Int64)
	}
	return err
}

func nullStringValue(v sql.NullString) interface{} {
	if v.Valid {
		return v.String
	}
	return nil
}

func backupLegacyRBACDB(legacyPath string) error {
	if stringsTrim := filepath.Clean(legacyPath); stringsTrim == "" {
		return nil
	}
	if _, err := os.Stat(legacyPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%d", legacyPath, time.Now().UnixMilli())
	return os.Rename(legacyPath, backupPath)
}

func seedDefaultData() error {
	roles, err := requireRoleRepo()
	if err != nil {
		return err
	}
	if err := roles.SeedDefaults(); err != nil {
		return err
	}

	users, err := requireUserRepo()
	if err != nil {
		return err
	}
	count, err := users.Count()
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return nil
}

func generateSalt() string {
	b := make([]byte, 16)
	if _, err := cryptorand.Read(b); err != nil {
		return hex.EncodeToString([]byte("openocta-fallback-salt"))
	}
	return hex.EncodeToString(b)
}

// IsAdminPasswordDefault reports whether admin still uses a known weak default (legacy installs only).
func IsAdminPasswordDefault() bool {
	users, err := requireUserRepo()
	if err != nil {
		return false
	}
	passwordHash, salt, ok, err := users.AdminCredentials()
	if err != nil || !ok {
		return false
	}
	if IsArgon2Hash(passwordHash) {
		return false
	}
	return passwordHash == HashPasswordLegacy("admin888", salt)
}
