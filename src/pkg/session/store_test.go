package session

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

func TestSessionJSONStore(t *testing.T) {
	// Ensure DB is not initialized to use JSON backend
	_ = db.CloseDB()

	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "sessions.json")

	// 1. Load from empty/non-existent
	store, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to load session store: %v", err)
	}
	if len(store) != 0 {
		t.Errorf("expected empty store, got %d entries", len(store))
	}

	// 2. Save session
	store["session-1"] = SessionEntry{
		SessionID: "session-1",
		UpdatedAt: time.Now().UnixMilli(),
		Label:     "Test Session",
	}

	err = SaveSessionStore(storePath, store)
	if err != nil {
		t.Fatalf("failed to save session store: %v", err)
	}

	// 3. Load again and verify
	loaded, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to reload session store: %v", err)
	}
	entry, ok := loaded["session-1"]
	if !ok {
		t.Fatalf("session-1 not found in loaded store")
	}
	if entry.Label != "Test Session" {
		t.Errorf("expected label 'Test Session', got %q", entry.Label)
	}
}

func TestSessionSQLiteStore(t *testing.T) {
	// Initialize DB to use SQLite backend
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	}()

	storePath := filepath.Join(tempDir, "sessions.json")

	// 1. Load from empty
	store, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to load session store: %v", err)
	}

	// 2. Save session
	store["session-1"] = SessionEntry{
		SessionID: "session-1",
		UpdatedAt: time.Now().UnixMilli(),
		Label:     "Test SQLite Session",
	}

	err = SaveSessionStore(storePath, store)
	if err != nil {
		t.Fatalf("failed to save session store: %v", err)
	}

	// 3. Load again and verify
	loaded, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to reload session store: %v", err)
	}
	entry, ok := loaded["session-1"]
	if !ok {
		t.Fatalf("session-1 not found in loaded store")
	}
	if entry.Label != "Test SQLite Session" {
		t.Errorf("expected label 'Test SQLite Session', got %q", entry.Label)
	}
}

func TestSessionMigration(t *testing.T) {
	_ = db.CloseDB()
	ResetSessionRepositoryForTest()

	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "sessions.json")

	store := SessionStore{
		"session-migrated": SessionEntry{
			SessionID: "session-migrated",
			UpdatedAt: time.Now().UnixMilli(),
			Label:     "Migrated Session",
		},
	}
	err := SaveSessionStore(storePath, store)
	if err != nil {
		t.Fatalf("failed to save json: %v", err)
	}

	// Switch to SQLite mode by initializing the DB
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	}()

	// Loading should trigger migration
	loaded, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to load SQLite session store: %v", err)
	}

	entry, ok := loaded["session-migrated"]
	if !ok {
		t.Fatalf("migrated session not found in SQLite")
	}
	if entry.Label != "Migrated Session" {
		t.Errorf("expected label 'Migrated Session', got %q", entry.Label)
	}

	// Verify old JSON file has been removed after import
	if _, err := os.Stat(storePath); err == nil {
		t.Errorf("old JSON file should have been removed after import, but still exists")
	}
	backups, _ := filepath.Glob(storePath + ".bak.*")
	if len(backups) == 0 {
		t.Errorf("expected timestamped JSON backup after import")
	}
}

func TestSessionConcurrentWrite(t *testing.T) {
	// Initialize DB
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	}()

	storePath := filepath.Join(tempDir, "sessions.json")

	// Pre-populate a session in the store
	store := SessionStore{
		"agent:main:session-1": SessionEntry{
			SessionID: "session-1",
			UpdatedAt: time.Now().UnixMilli(),
			Label:     "Session 1",
		},
	}
	err := SaveSessionStore(storePath, store)
	if err != nil {
		t.Fatalf("failed to pre-populate: %v", err)
	}

	var wg sync.WaitGroup
	workers := 20
	iterations := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Concurrently update session touch time
				envFunc := func(key string) string {
					if key == "OPENOCTA_STATE_DIR" {
						return tempDir
					}
					return os.Getenv(key)
				}
				err := UpdateSessionUpdatedAt("main", "session-1", envFunc, time.Now().UnixMilli())
				if err != nil {
					t.Errorf("worker %d failed to touch: %v", workerID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Reload and verify
	loaded, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatalf("failed to reload: %v", err)
	}
	entry, ok := loaded["agent:main:session-1"]
	if !ok {
		t.Fatalf("session not found after concurrent updates")
	}
	if entry.SessionID != "session-1" {
		t.Errorf("unexpected entry SessionID: %s", entry.SessionID)
	}
}

func TestSessionSQLiteIsolation(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	}()

	pathA := filepath.Join(tempDir, "sessions_a.json")
	pathB := filepath.Join(tempDir, "sessions_b.json")

	// 1. Save session in path A
	storeA := SessionStore{
		"session-a": SessionEntry{SessionID: "session-a", Label: "Session A"},
	}
	if err := SaveSessionStore(pathA, storeA); err != nil {
		t.Fatalf("SaveSessionStore A: %v", err)
	}

	// 2. Save session in path B
	storeB := SessionStore{
		"session-b": SessionEntry{SessionID: "session-b", Label: "Session B"},
	}
	if err := SaveSessionStore(pathB, storeB); err != nil {
		t.Fatalf("SaveSessionStore B: %v", err)
	}

	// 3. Load A and verify it only contains session-a (no session-b, i.e., no leak or overwrite)
	loadedA, err := LoadSessionStore(pathA)
	if err != nil {
		t.Fatalf("LoadSessionStore A: %v", err)
	}
	if _, ok := loadedA["session-a"]; !ok {
		t.Error("expected session-a in path A")
	}
	if _, ok := loadedA["session-b"]; ok {
		t.Error("did not expect session-b in path A (isolation leak)")
	}

	// 4. Load B and verify it only contains session-b
	loadedB, err := LoadSessionStore(pathB)
	if err != nil {
		t.Fatalf("LoadSessionStore B: %v", err)
	}
	if _, ok := loadedB["session-b"]; !ok {
		t.Error("expected session-b in path B")
	}
	if _, ok := loadedB["session-a"]; ok {
		t.Error("did not expect session-a in path B (isolation leak)")
	}

	// 5. Test legacy blob table migration into sessions_v1
	ResetSessionRepositoryForTest()
	_ = db.CloseDB()
	sqliteDB, err := sql.Open("sqlite", filepath.Join(tempDir, "openocta.db"))
	if err != nil {
		t.Fatalf("failed to open raw sqlite: %v", err)
	}
	_, _ = sqliteDB.Exec("DROP TABLE IF EXISTS sessions")
	_, err = sqliteDB.Exec(`
		CREATE TABLE sessions (
			store_path TEXT,
			session_key TEXT,
			detail_json TEXT,
			PRIMARY KEY (store_path, session_key)
		);
	`)
	if err != nil {
		sqliteDB.Close()
		t.Fatalf("failed to create legacy sessions table: %v", err)
	}
	_, err = sqliteDB.Exec(`INSERT INTO sessions (store_path, session_key, detail_json) VALUES (?, ?, ?)`, "", "legacy-key", `{"sessionId":"legacy-id","label":"Legacy","updatedAt":1000}`)
	if err != nil {
		sqliteDB.Close()
		t.Fatalf("failed to insert legacy data: %v", err)
	}
	sqliteDB.Close()

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("re-InitDB: %v", err)
	}
	ResetSessionRepositoryForTest()

	loadedLegacy, err := LoadSessionStore("")
	if err != nil {
		t.Fatalf("failed to load after schema migration: %v", err)
	}
	entry, ok := loadedLegacy["legacy-key"]
	if !ok {
		t.Fatalf("legacy session not found after schema migration")
	}
	if entry.Label != "Legacy" {
		t.Errorf("expected label 'Legacy', got %q", entry.Label)
	}
	var legacyLeft int
	if err := db.GetDB().QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='sessions'`).Scan(&legacyLeft); err != nil {
		t.Fatal(err)
	}
	if legacyLeft != 0 {
		t.Errorf("expected legacy sessions table dropped, still present")
	}
}
