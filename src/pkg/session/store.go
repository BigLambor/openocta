// Package session provides session store loading.
// Mirrors src/config/sessions/store.ts loadSessionStore.
package session

import (
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

// LoadSessionStore reads the session store for storePath (DB primary; JSON fallback when DB unavailable).
func LoadSessionStore(storePath string) (SessionStore, error) {
	if repo := defaultSessionRepository(); repo != nil {
		if err := repo.ImportJSONIfEmpty(storePath); err != nil {
			fmt.Printf("warning: session JSON import failed: %v\n", err)
		}
		return repo.ListByStorePath(storePath)
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

// SaveSessionStore writes the session store (DB primary; JSON fallback when DB unavailable).
func SaveSessionStore(storePath string, store SessionStore) error {
	if repo := defaultSessionRepository(); repo != nil {
		return repo.ReplaceStore(storePath, store)
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

	if repo := defaultSessionRepository(); repo != nil {
		store, err := repo.ListByStorePath(storePath)
		if err != nil {
			return err
		}
		canonicalKey := "agent:" + id + ":" + sessionID
		for k, e := range store {
			if k == sessionID || k == canonicalKey || e.SessionID == sessionID {
				e.UpdatedAt = nowMs
				if e.SessionID == "" {
					e.SessionID = sessionID
				}
				return repo.UpsertEntry(storePath, k, e)
			}
		}
		return nil
	}

	store, err := LoadSessionStore(storePath)
	if err != nil {
		return err
	}
	if store == nil {
		store = SessionStore{}
	}
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
	return nil
}

// ResetSessionRepositoryForTest clears cached repository state (tests only).
func ResetSessionRepositoryForTest() {
	defaultSessionRepo = nil
	legacyMigrated = false
}

// UsesDBStore reports whether session metadata is persisted in openocta.db.
func UsesDBStore() bool {
	return db.GetDB() != nil
}
