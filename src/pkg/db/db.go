package db

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

var openoctaDB *sql.DB

//go:embed migrations/*.sql
var migrationFS embed.FS

type migrationFile struct {
	version  int64
	name     string
	path     string
	sql      string
	checksum string
}

// InitDB initializes the unified openocta.db database in WAL mode.
func InitDB(stateDir string) error {
	if openoctaDB != nil {
		if err := openoctaDB.Close(); err != nil {
			return err
		}
		openoctaDB = nil
	}

	dbPath := filepath.Join(stateDir, "openocta.db")
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	dsn := dbPath + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_txlock=immediate"
	var err error
	openoctaDB, err = sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}

	if err := openoctaDB.Ping(); err != nil {
		return err
	}

	if err := RunMigrations(openoctaDB); err != nil {
		_ = openoctaDB.Close()
		openoctaDB = nil
		return err
	}

	return nil
}

// RunMigrations applies embedded SQL migrations to openocta.db.
func RunMigrations(sqliteDB *sql.DB) error {
	if sqliteDB == nil {
		return fmt.Errorf("nil database")
	}
	if _, err := sqliteDB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			checksum TEXT NOT NULL,
			applied_at INTEGER NOT NULL
		);
	`); err != nil {
		return err
	}

	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	for _, m := range migrations {
		var appliedName, appliedChecksum string
		err := sqliteDB.QueryRow(`SELECT name, checksum FROM schema_migrations WHERE version = ?`, m.version).Scan(&appliedName, &appliedChecksum)
		if err == nil {
			if appliedChecksum != m.checksum {
				return fmt.Errorf("migration %d checksum mismatch: database has %s, embedded %s", m.version, appliedChecksum, m.checksum)
			}
			continue
		}
		if err != sql.ErrNoRows {
			return err
		}
		if err := applyMigration(sqliteDB, m); err != nil {
			return err
		}
	}
	return nil
}

func loadMigrations() ([]migrationFile, error) {
	entries, err := migrationFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}
	migrations := make([]migrationFile, 0, len(entries))
	seen := map[int64]string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		versionPart, namePart, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("invalid migration name %q: expected NNN_name.sql", entry.Name())
		}
		version, err := strconv.ParseInt(versionPart, 10, 64)
		if err != nil || version <= 0 {
			return nil, fmt.Errorf("invalid migration version in %q", entry.Name())
		}
		if previous := seen[version]; previous != "" {
			return nil, fmt.Errorf("duplicate migration version %d: %s and %s", version, previous, entry.Name())
		}
		seen[version] = entry.Name()

		path := "migrations/" + entry.Name()
		data, err := migrationFS.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sum := sha256.Sum256(data)
		migrations = append(migrations, migrationFile{
			version:  version,
			name:     strings.TrimSuffix(namePart, ".sql"),
			path:     path,
			sql:      string(data),
			checksum: hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].version < migrations[j].version
	})
	return migrations, nil
}

func applyMigration(sqliteDB *sql.DB, m migrationFile) error {
	tx, err := sqliteDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, stmt := range splitSQLStatements(m.sql) {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("apply migration %d %s: %w", m.version, m.path, err)
		}
	}

	if _, err := tx.Exec(
		`INSERT INTO schema_migrations (version, name, checksum, applied_at) VALUES (?, ?, ?, ?)`,
		m.version,
		m.name,
		m.checksum,
		time.Now().UnixMilli(),
	); err != nil {
		return err
	}
	return tx.Commit()
}

func splitSQLStatements(sqlText string) []string {
	var out []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	lineComment := false

	for i := 0; i < len(sqlText); i++ {
		ch := sqlText[i]
		next := byte(0)
		if i+1 < len(sqlText) {
			next = sqlText[i+1]
		}

		if lineComment {
			b.WriteByte(ch)
			if ch == '\n' {
				lineComment = false
			}
			continue
		}
		if !inSingle && !inDouble && ch == '-' && next == '-' {
			lineComment = true
			b.WriteByte(ch)
			continue
		}
		switch ch {
		case '\'':
			if !inDouble {
				inSingle = !inSingle
			}
		case '"':
			if !inSingle {
				inDouble = !inDouble
			}
		case ';':
			if !inSingle && !inDouble {
				out = append(out, b.String())
				b.Reset()
				continue
			}
		}
		b.WriteByte(ch)
	}
	if strings.TrimSpace(b.String()) != "" {
		out = append(out, b.String())
	}
	return out
}

// GetDB returns the underlying openocta.db database connection.
func GetDB() *sql.DB {
	return openoctaDB
}

// CloseDB closes the database connection and resets the global variable.
func CloseDB() error {
	if openoctaDB != nil {
		err := openoctaDB.Close()
		openoctaDB = nil
		return err
	}
	return nil
}
