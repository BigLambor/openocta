package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

func TestSessionRepositoryNormalizedColumns(t *testing.T) {
	dir := t.TempDir()
	if err := db.InitDB(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	})

	storePath := filepath.Join(dir, "agents", "main", "sessions", "sessions.json")
	entry := SessionEntry{
		SessionID:   "sess-normalized-1",
		UpdatedAt:   time.Now().UnixMilli(),
		Label:       "Ops Chat",
		Channel:     "feishu",
		SpawnedBy:   "webhook",
		ChatType:    "group",
		SessionFile: "sess-normalized-1.jsonl",
	}
	store := SessionStore{
		"agent:main:channel:feishu:oc_1": entry,
	}
	if err := SaveSessionStore(storePath, store); err != nil {
		t.Fatal(err)
	}

	var agentID, sessionKey, title, channel, origin string
	err := db.GetDB().QueryRow(`
		SELECT agent_id, session_key, title, channel, origin
		FROM sessions_v1
		WHERE store_path = ? AND session_id = ?
	`, storePath, "sess-normalized-1").Scan(&agentID, &sessionKey, &title, &channel, &origin)
	if err != nil {
		t.Fatalf("query sessions_v1: %v", err)
	}
	if agentID != "main" || sessionKey != "agent:main:channel:feishu:oc_1" {
		t.Fatalf("unexpected identity columns: agent=%s key=%s", agentID, sessionKey)
	}
	if title != "Ops Chat" || channel != "feishu" || origin != "webhook" {
		t.Fatalf("unexpected metadata: title=%s channel=%s origin=%s", title, channel, origin)
	}

	ResetSessionRepositoryForTest()
	loaded, err := LoadSessionStore(storePath)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := loaded["agent:main:channel:feishu:oc_1"]
	if !ok {
		t.Fatal("session not reloaded")
	}
	if got.Label != "Ops Chat" || got.ChatType != "group" || got.SessionFile != "sess-normalized-1.jsonl" {
		t.Fatalf("unexpected reloaded entry: %+v", got)
	}
	if _, err := os.Stat(storePath); err == nil {
		t.Log("sessions.json may still exist when written before DB path; DB remains source of truth")
	}
}

func TestUsesDBStore(t *testing.T) {
	_ = db.CloseDB()
	if UsesDBStore() {
		t.Fatal("expected false without DB")
	}
	dir := t.TempDir()
	if err := db.InitDB(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ResetSessionRepositoryForTest()
		_ = db.CloseDB()
	})
	if !UsesDBStore() {
		t.Fatal("expected true with DB")
	}
}
