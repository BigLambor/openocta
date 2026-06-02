package rbac

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

var db *sql.DB

// InitDB initializes the RBAC database and pre-seeds it if empty.
func InitDB(stateDir string) error {
	dbPath := filepath.Join(stateDir, "rbac.db")
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	var err error
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}

	if err := db.Ping(); err != nil {
		return err
	}

	_, _ = db.Exec("PRAGMA journal_mode=WAL;")

	if err := createTables(); err != nil {
		return err
	}

	if err := seedDefaultData(); err != nil {
		return err
	}

	return nil
}

func createTables() error {
	// 1. users table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			salt TEXT NOT NULL,
			role_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		return err
	}

	// 2. roles table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS roles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			description TEXT
		);
	`)
	if err != nil {
		return err
	}

	// 3. permissions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS permissions (
			code TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL
		);
	`)
	if err != nil {
		return err
	}

	// 4. role_permissions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS role_permissions (
			role_id INTEGER NOT NULL,
			permission_code TEXT NOT NULL,
			PRIMARY KEY (role_id, permission_code)
		);
	`)
	if err != nil {
		return err
	}

	// 5. user_tokens table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_tokens (
			token TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			expires_at DATETIME NOT NULL
		);
	`)
	return err
}

func seedDefaultData() error {
	// Seed roles
	roles := []struct {
		id   int
		name string
		desc string
	}{
		{1, "admin", "超级管理员 - 拥有全量管理与各运维域执行权限"},
		{2, "hadoop_operator", "Hadoop生态运维员 - 具备Hadoop域的巡检与交互权限"},
		{3, "fi_operator", "FI商业版运维员 - 具备FI域的巡检与交互权限"},
		{4, "gbase_operator", "GBase数据库运维员 - 具备GBase域的巡检与交互权限"},
		{5, "viewer", "只读访客 - 仅有大盘和系统巡检查看权限，无法交互与手动巡检"},
	}

	for _, r := range roles {
		_, _ = db.Exec(`INSERT OR IGNORE INTO roles (id, name, description) VALUES (?, ?, ?)`, r.id, r.name, r.desc)
	}

	// Seed permissions
	permissions := []struct {
		code string
		name string
		ptype string
	}{
		{"menu:overview", "主导航: 运维大屏", "menu"},
		{"menu:hadoop", "主导航: Hadoop生态", "menu"},
		{"menu:fi", "主导航: FI商业生态", "menu"},
		{"menu:gbase", "主导航: GBase数据库", "menu"},
		{"menu:governance", "主导航: 开发治理平台", "menu"},
		{"menu:dataapps", "主导航: 数据App运维", "menu"},
		{"menu:config", "主导航: 系统设置", "menu"},
		{"ops:inspect", "操作: 执行深度巡检", "ops"},
		{"ops:diagnose", "操作: 发起智能诊断", "ops"},
		{"ops:ack", "操作: 确认处理告警组", "ops"},
		{"ops:wework_conf", "操作: 企业微信通道配置", "ops"},
	}

	for _, p := range permissions {
		_, _ = db.Exec(`INSERT OR IGNORE INTO permissions (code, name, type) VALUES (?, ?, ?)`, p.code, p.name, p.ptype)
	}

	// Bind permissions to Admin (role_id = 1 gets everything)
	for _, p := range permissions {
		_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)`, 1, p.code)
	}
	// Ensure newly added ops permissions are granted on existing databases
	for _, code := range []string{"ops:ack"} {
		_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (1, ?)`, code)
	}

	// Bind permissions to GBase Operator (role_id = 4 gets overview, gbase, diagnose, inspect)
	gbasePerms := []string{"menu:overview", "menu:gbase", "ops:diagnose", "ops:inspect", "ops:ack"}
	for _, p := range gbasePerms {
		_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)`, 4, p)
	}

	// Bind permissions to Viewer (role_id = 5 gets overview, hadoop, fi, gbase, governance, dataapps only)
	viewerPerms := []string{"menu:overview", "menu:hadoop", "menu:fi", "menu:gbase", "menu:governance", "menu:dataapps"}
	for _, p := range viewerPerms {
		_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)`, 5, p)
	}

	// Seed default Admin User (admin / admin888) if no users exist
	var count int
	err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		salt := generateSalt()
		passwordHash := HashPassword("admin888", salt)
		_, err = db.Exec(`
			INSERT INTO users (username, password_hash, salt, role_id)
			VALUES (?, ?, ?, ?)
		`, "admin", passwordHash, salt, 1) // role_id = 1 (admin)
		if err != nil {
			return err
		}

		// Also seed a test GBase Operator (gbase_op / op123456)
		saltOp := generateSalt()
		opHash := HashPassword("op123456", saltOp)
		_, _ = db.Exec(`
			INSERT INTO users (username, password_hash, salt, role_id)
			VALUES (?, ?, ?, ?)
		`, "gbase_op", opHash, saltOp, 4) // role_id = 4 (gbase_operator)
	}

	return nil
}

func generateSalt() string {
	b := make([]byte, 16)
	rand.Seed(time.Now().UnixNano())
	rand.Read(b)
	return hex.EncodeToString(b)
}

// HashPassword hashes plain password with SHA256 using salt.
func HashPassword(password, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

// IsAdminPasswordDefault checks if the default admin user is still using the default password "admin888".
func IsAdminPasswordDefault() bool {
	if db == nil {
		return false
	}
	var passwordHash, salt string
	err := db.QueryRow(`SELECT password_hash, salt FROM users WHERE username = ?`, "admin").Scan(&passwordHash, &salt)
	if err != nil {
		return false
	}
	return passwordHash == HashPassword("admin888", salt)
}
