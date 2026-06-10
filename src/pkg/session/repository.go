package session

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

const sessionsV1Table = "sessions_v1"

type sessionRepository struct {
	db *sql.DB
}

var (
	defaultSessionRepo *sessionRepository
	legacyMigrated     bool
)

func defaultSessionRepository() *sessionRepository {
	sqliteDB := db.GetDB()
	if sqliteDB == nil {
		return nil
	}
	if defaultSessionRepo == nil {
		defaultSessionRepo = &sessionRepository{}
	}
	defaultSessionRepo.db = sqliteDB
	return defaultSessionRepo
}

func (r *sessionRepository) ensureReady() error {
	if r == nil || r.db == nil {
		return fmt.Errorf("session repository 未初始化")
	}
	if legacyMigrated {
		return nil
	}
	if err := r.migrateLegacyTable(); err != nil {
		return err
	}
	legacyMigrated = true
	return nil
}

func (r *sessionRepository) ListByStorePath(storePath string) (SessionStore, error) {
	if err := r.ensureReady(); err != nil {
		return nil, err
	}
	rows, err := r.db.Query(`
		SELECT session_key, session_id, title, origin, channel, owner_id, detail_json, created_at, updated_at
		FROM sessions_v1
		WHERE store_path = ?
		ORDER BY updated_at DESC
	`, storePath)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	store := SessionStore{}
	for rows.Next() {
		var (
			sessionKey, sessionID, title, origin, channel, ownerID, detailJSON string
			createdAt, updatedAt                                                int64
		)
		if err := rows.Scan(&sessionKey, &sessionID, &title, &origin, &channel, &ownerID, &detailJSON, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		entry := mergeSessionEntry(sessionID, title, origin, channel, ownerID, detailJSON, createdAt, updatedAt)
		store[sessionKey] = entry
	}
	return store, rows.Err()
}

func (r *sessionRepository) ReplaceStore(storePath string, store SessionStore) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM sessions_v1 WHERE store_path = ?`, storePath); err != nil {
		return err
	}
	for sessionKey, entry := range store {
		if err := upsertSessionRowTx(tx, storePath, sessionKey, entry); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *sessionRepository) UpsertEntry(storePath, sessionKey string, entry SessionEntry) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := upsertSessionRowTx(tx, storePath, sessionKey, entry); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *sessionRepository) ImportJSONIfEmpty(storePath string) error {
	if err := r.ensureReady(); err != nil {
		return err
	}
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM sessions_v1 WHERE store_path = ?`, storePath).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	storePath = strings.TrimSpace(storePath)
	if storePath == "" {
		return nil
	}
	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if len(data) == 0 {
		return nil
	}
	var store SessionStore
	if err := json.Unmarshal(data, &store); err != nil || len(store) == 0 {
		return nil
	}
	if err := r.ReplaceStore(storePath, store); err != nil {
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%d", storePath, time.Now().UnixMilli())
	if err := os.WriteFile(backupPath, data, 0o644); err != nil {
		return err
	}
	return os.Remove(storePath)
}

func (r *sessionRepository) migrateLegacyTable() error {
	if !r.legacyTableExists() {
		return nil
	}
	rows, err := r.db.Query(`SELECT store_path, session_key, detail_json FROM sessions`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	count := 0
	for rows.Next() {
		var storePath, sessionKey, detailJSON string
		if err := rows.Scan(&storePath, &sessionKey, &detailJSON); err != nil {
			return err
		}
		var entry SessionEntry
		if err := json.Unmarshal([]byte(detailJSON), &entry); err != nil {
			continue
		}
		if err := upsertSessionRowTx(tx, storePath, sessionKey, entry); err != nil {
			return err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if count == 0 {
		if _, err := tx.Exec(`DROP TABLE IF EXISTS sessions`); err != nil {
			return err
		}
		return tx.Commit()
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	_, err = r.db.Exec(`DROP TABLE IF EXISTS sessions`)
	return err
}

func (r *sessionRepository) legacyTableExists() bool {
	var name string
	err := r.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='sessions'
	`).Scan(&name)
	if err != nil {
		return false
	}
	rows, err := r.db.Query(`PRAGMA table_info(sessions)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	hasDetailJSON := false
	hasID := false
	for rows.Next() {
		var cid, notnull, pk int
		var colName, colType string
		var dflt interface{}
		if err := rows.Scan(&cid, &colName, &colType, &notnull, &dflt, &pk); err != nil {
			return false
		}
		switch colName {
		case "detail_json":
			hasDetailJSON = true
		case "id":
			hasID = true
		}
	}
	return hasDetailJSON && !hasID
}

func upsertSessionRowTx(tx *sql.Tx, storePath, sessionKey string, entry SessionEntry) error {
	agentID := agentIDFromSessionKey(sessionKey)
	now := time.Now().UnixMilli()
	createdAt := entry.UpdatedAt
	if createdAt <= 0 {
		createdAt = now
	}
	updatedAt := entry.UpdatedAt
	if updatedAt <= 0 {
		updatedAt = now
	}
	if strings.TrimSpace(entry.SessionID) == "" {
		entry.SessionID = sessionIDFromKey(sessionKey)
	}
	title := strings.TrimSpace(entry.Label)
	origin := strings.TrimSpace(entry.SpawnedBy)
	channel := strings.TrimSpace(entry.Channel)
	detailJSON, err := marshalSessionDetailExtra(entry)
	if err != nil {
		return err
	}
	rowID := sessionRowID(storePath, sessionKey)
	_, err = tx.Exec(`
		INSERT INTO sessions_v1 (
			id, tenant_id, workspace_id, agent_id, session_key, session_id, title, origin, channel,
			owner_id, store_path, detail_json, created_at, updated_at
		) VALUES (?, 'default', 'default', ?, ?, ?, ?, ?, ?, '', ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			agent_id = excluded.agent_id,
			session_key = excluded.session_key,
			session_id = excluded.session_id,
			title = excluded.title,
			origin = excluded.origin,
			channel = excluded.channel,
			detail_json = excluded.detail_json,
			updated_at = excluded.updated_at
	`, rowID, agentID, sessionKey, entry.SessionID, title, origin, channel, storePath, detailJSON, createdAt, updatedAt)
	return err
}

type sessionDetailExtra struct {
	SessionFile        string      `json:"sessionFile,omitempty"`
	ChatType           string      `json:"chatType,omitempty"`
	ThinkingLevel      string      `json:"thinkingLevel,omitempty"`
	VerboseLevel       string      `json:"verboseLevel,omitempty"`
	SystemPromptReport interface{} `json:"systemPromptReport,omitempty"`
	SkillsSnapshot     interface{} `json:"skillsSnapshot,omitempty"`
	Tools              interface{} `json:"tools,omitempty"`
}

func marshalSessionDetailExtra(entry SessionEntry) (string, error) {
	extra := sessionDetailExtra{
		SessionFile:        entry.SessionFile,
		ChatType:           entry.ChatType,
		ThinkingLevel:      entry.ThinkingLevel,
		VerboseLevel:       entry.VerboseLevel,
		SystemPromptReport: entry.SystemPromptReport,
		SkillsSnapshot:     entry.SkillsSnapshot,
		Tools:              entry.Tools,
	}
	data, err := json.Marshal(extra)
	if err != nil {
		return "{}", err
	}
	return string(data), nil
}

func mergeSessionEntry(sessionID, title, origin, channel, ownerID, detailJSON string, createdAt, updatedAt int64) SessionEntry {
	entry := SessionEntry{
		SessionID: sessionID,
		Label:     title,
		SpawnedBy: origin,
		Channel:   channel,
		UpdatedAt: updatedAt,
	}
	if entry.UpdatedAt <= 0 {
		entry.UpdatedAt = createdAt
	}
	_ = ownerID
	var extra sessionDetailExtra
	if detailJSON != "" && json.Unmarshal([]byte(detailJSON), &extra) == nil {
		entry.SessionFile = extra.SessionFile
		entry.ChatType = extra.ChatType
		entry.ThinkingLevel = extra.ThinkingLevel
		entry.VerboseLevel = extra.VerboseLevel
		entry.SystemPromptReport = extra.SystemPromptReport
		entry.SkillsSnapshot = extra.SkillsSnapshot
		entry.Tools = extra.Tools
	}
	return entry
}

func sessionRowID(storePath, sessionKey string) string {
	sum := sha256.Sum256([]byte(storePath + "\x00" + sessionKey))
	return hex.EncodeToString(sum[:16])
}

func agentIDFromSessionKey(sessionKey string) string {
	parts := strings.Split(strings.TrimSpace(sessionKey), ":")
	if len(parts) >= 2 && strings.EqualFold(parts[0], "agent") {
		return normalizeAgentID(parts[1])
	}
	return DefaultAgentID
}

func sessionIDFromKey(sessionKey string) string {
	parts := strings.Split(strings.TrimSpace(sessionKey), ":")
	if len(parts) >= 3 && strings.EqualFold(parts[0], "agent") {
		return strings.Join(parts[2:], ":")
	}
	return sessionKey
}
