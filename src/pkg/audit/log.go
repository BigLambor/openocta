package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	openoctadb "github.com/openocta/openocta/pkg/db"
)

// Entry is a persisted audit log row.
type Entry struct {
	ActorID    string
	Action     string
	ObjectType string
	ObjectID   string
	RequestID  string
	RunID      string
	SessionID  string
	Summary    string
	Metadata   map[string]interface{}
}

// Record writes an audit log entry to openocta.db.
func Record(entry Entry) error {
	db := openoctadb.GetDB()
	if db == nil {
		return fmt.Errorf("openocta.db 未初始化")
	}
	return RecordDB(db, entry)
}

// RecordDB writes an audit log entry using the provided database handle.
func RecordDB(db *sql.DB, entry Entry) error {
	if db == nil {
		return fmt.Errorf("nil database")
	}
	if entry.Action == "" || entry.ObjectType == "" {
		return fmt.Errorf("audit action and object_type are required")
	}
	meta := entry.Metadata
	if meta == nil {
		meta = map[string]interface{}{}
	}
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	_, err = db.Exec(`
		INSERT INTO audit_logs (
			id, tenant_id, workspace_id, actor_id, action, object_type, object_id,
			request_id, run_id, session_id, summary, metadata_json, created_at, updated_at
		) VALUES (?, 'default', 'default', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, uuid.New().String(), entry.ActorID, entry.Action, entry.ObjectType, entry.ObjectID,
		entry.RequestID, entry.RunID, entry.SessionID, entry.Summary, string(metaJSON), now, now)
	return err
}
