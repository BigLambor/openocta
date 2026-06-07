package db

import (
	"database/sql"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

var openoctaDB *sql.DB

// InitDB initializes the unified openocta.db database in WAL mode.
func InitDB(stateDir string) error {
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

	return nil
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
