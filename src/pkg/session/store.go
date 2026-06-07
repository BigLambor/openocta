// Package session provides session store loading.
// Mirrors src/config/sessions/store.ts loadSessionStore.
package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

// SessionMu protects the session store read-modify-write operations across different goroutines.
var SessionMu sync.Mutex

// SessionEntry is a minimal session store entry.
// Mirrors SessionEntry from src/config/sessions/types.ts.
// Extended with systemPromptReport, skillsSnapshot, tools for UI and next-session load.
type SessionEntry struct {
	SessionID          string      `json:"sessionId"`
	UpdatedAt          int64       `json:"updatedAt"`
	SessionFile        string      `json:"sessionFile,omitempty"`
	Label              string      `json:"label,omitempty"`
	SpawnedBy          string      `json:"spawnedBy,omitempty"`
	Channel            string      `json:"channel,omitempty"`
	ChatType           string      `json:"chatType,omitempty"`
	ThinkingLevel      string      `json:"thinkingLevel,omitempty"`
	VerboseLevel       string      `json:"verboseLevel,omitempty"`
	SystemPromptReport interface{} `json:"systemPromptReport,omitempty"`
	SkillsSnapshot     interface{} `json:"skillsSnapshot,omitempty"`
	Tools              interface{} `json:"tools,omitempty"`
}

// SessionStore is a map of key -> SessionEntry.
type SessionStore map[string]SessionEntry

// ResolveDefaultSessionStorePath returns the default sessions.json path for an agent.
func ResolveDefaultSessionStorePath(agentID string, env func(string) string) string {
	sessionsDir := ResolveAgentSessionsDir(agentID, env)
	return filepath.Join(sessionsDir, "sessions.json")
}

// LoadSessionStore reads sessions.json and returns the store.
func LoadSessionStore(storePath string) (SessionStore, error) {
	sqliteDB := db.GetDB()
	if sqliteDB != nil {
		if err := createSessionTables(sqliteDB); err != nil {
			return nil, err
		}
		if err := migrateSessionJSONToSQLite(sqliteDB, storePath); err != nil {
			fmt.Printf("warning: session JSON to SQLite migration failed: %v\n", err)
		}

		rows, err := sqliteDB.Query(`SELECT session_key, detail_json FROM sessions WHERE store_path = ?`, storePath)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		store := SessionStore{}
		for rows.Next() {
			var key, detailStr string
			if err := rows.Scan(&key, &detailStr); err != nil {
				return nil, err
			}
			var entry SessionEntry
			if err := json.Unmarshal([]byte(detailStr), &entry); err != nil {
				return nil, err
			}
			store[key] = entry
		}
		return store, nil
	}

	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionStore{}, nil
		}
		return nil, err
	}
	var store SessionStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store == nil {
		store = SessionStore{}
	}
	return store, nil
}

// LoadCombinedSessionStore loads and merges session stores for the given agents.
// If agentIDs is empty, uses ["main"].
func LoadCombinedSessionStore(env func(string) string, agentIDs []string) (storePath string, store SessionStore) {
	if len(agentIDs) == 0 {
		agentIDs = []string{"main"}
	}
	store = SessionStore{}
	for _, agentID := range agentIDs {
		p := ResolveDefaultSessionStorePath(agentID, env)
		s, err := LoadSessionStore(p)
		if err != nil {
			continue
		}
		for k, e := range s {
			canonical := k
			if agentID != "main" && !strings.HasPrefix(k, "agent:") {
				canonical = "agent:" + agentID + ":" + k
			}
			store[canonical] = e
			// Only add agent:agentID:sessionID alias when key is bare (e.g. bare sessionID).
			// Skip when key already has agent: prefix to avoid duplicate like
			// "agent:main:channel:feishu:oc_xxx" vs "agent:main:channel-feishu-oc_xxx".
			if e.SessionID != "" && !strings.HasPrefix(k, "agent:") {
				store["agent:"+agentID+":"+e.SessionID] = e
			}
		}
		if storePath == "" {
			storePath = p
		}
	}
	if len(agentIDs) > 1 {
		storePath = "(multiple)"
	}
	return storePath, store
}

// SaveSessionStore writes the session store back to disk.
func SaveSessionStore(storePath string, store SessionStore) error {
	sqliteDB := db.GetDB()
	if sqliteDB != nil {
		if err := createSessionTables(sqliteDB); err != nil {
			return err
		}

		tx, err := sqliteDB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		_, err = tx.Exec(`DELETE FROM sessions WHERE store_path = ?`, storePath)
		if err != nil {
			return err
		}

		stmt, err := tx.Prepare(`INSERT INTO sessions (store_path, session_key, detail_json) VALUES (?, ?, ?)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for k, entry := range store {
			b, err := json.Marshal(entry)
			if err != nil {
				return err
			}
			_, err = stmt.Exec(storePath, k, string(b))
			if err != nil {
				return err
			}
		}
		return tx.Commit()
	}

	if storePath == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0o644)
}

// UpdateSessionUpdatedAt updates or creates a session entry's updatedAt for the given agent/session.
// This mirrors the behavior of touching a session on new activity.
func UpdateSessionUpdatedAt(agentID, sessionID string, env func(string) string, nowMs int64) error {
	SessionMu.Lock()
	defer SessionMu.Unlock()

	if env == nil {
		env = os.Getenv
	}
	if nowMs == 0 {
		nowMs = time.Now().UnixMilli()
	}
	id := normalizeAgentID(agentID)
	storePath := ResolveDefaultSessionStorePath(id, env)
	store, err := LoadSessionStore(storePath)
	if err != nil {
		return err
	}
	if store == nil {
		store = SessionStore{}
	}

	// Try to find an existing entry for this session.
	canonicalKey := "agent:" + id + ":" + sessionID
	for k, e := range store {
		if k == sessionID || k == canonicalKey || e.SessionID == sessionID {
			e.UpdatedAt = nowMs
			if e.SessionID == "" {
				e.SessionID = sessionID
			}
			store[k] = e
			return SaveSessionStore(storePath, store)
		}
	}

	// If not found, do NOT create entry keyed by bare sessionID.
	// Creating store[sessionID] would produce a duplicate key (e.g. "channel-feishu-oc_xxx")
	// that differs from the canonical sessionKey (e.g. "agent:main:channel:feishu:oc_xxx").
	// The entry will be created by updateSessionAfterRun with the correct sessionKey.
	return nil
}

func createSessionTables(sqliteDB *sql.DB) error {
	var count int
	err := sqliteDB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name='sessions'").Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		_, err = sqliteDB.Exec(`
			CREATE TABLE sessions (
				store_path TEXT,
				session_key TEXT,
				detail_json TEXT,
				PRIMARY KEY (store_path, session_key)
			);
		`)
		return err
	}

	rows, err := sqliteDB.Query("PRAGMA table_info(sessions)")
	if err != nil {
		return err
	}
	defer rows.Close()

	hasStorePath := false
	for rows.Next() {
		var cid int
		var name, typeVal string
		var notnull, pk int
		var dfltVal interface{}
		if err := rows.Scan(&cid, &name, &typeVal, &notnull, &dfltVal, &pk); err != nil {
			return err
		}
		if name == "store_path" {
			hasStorePath = true
		}
	}

	if !hasStorePath {
		tx, err := sqliteDB.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback()

		if _, err := tx.Exec("ALTER TABLE sessions RENAME TO sessions_old"); err != nil {
			return err
		}
		if _, err := tx.Exec(`
			CREATE TABLE sessions (
				store_path TEXT,
				session_key TEXT,
				detail_json TEXT,
				PRIMARY KEY (store_path, session_key)
			);
		`); err != nil {
			return err
		}
		if _, err := tx.Exec("INSERT INTO sessions (store_path, session_key, detail_json) SELECT '', session_key, detail_json FROM sessions_old"); err != nil {
			return err
		}
		if _, err := tx.Exec("DROP TABLE sessions_old"); err != nil {
			return err
		}
		return tx.Commit()
	}

	return nil
}

func migrateSessionJSONToSQLite(db *sql.DB, storePath string) error {
	if _, err := os.Stat(storePath); err == nil {
		data, err := os.ReadFile(storePath)
		if err == nil && len(data) > 0 {
			var store SessionStore
			if json.Unmarshal(data, &store) == nil && len(store) > 0 {
				tx, err := db.Begin()
				if err != nil {
					return err
				}
				defer tx.Rollback()

				stmt, err := tx.Prepare(`INSERT OR REPLACE INTO sessions (store_path, session_key, detail_json) VALUES (?, ?, ?)`)
				if err != nil {
					return err
				}
				defer stmt.Close()

				for k, entry := range store {
					b, err := json.Marshal(entry)
					if err != nil {
						return err
					}
					_, err = stmt.Exec(storePath, k, string(b))
					if err != nil {
						return err
					}
				}

				if err := tx.Commit(); err == nil {
					_ = os.Rename(storePath, storePath+".bak")
				}
			}
		}
	}
	return nil
}
