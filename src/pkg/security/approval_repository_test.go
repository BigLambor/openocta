package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

func TestApprovalQueueDBPersistence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", dir)
	t.Setenv("OPENOCTA_APPROVAL_JSON_STORE", "0")
	t.Cleanup(func() {
		ClearApprovalQueues()
		sharedApprovalRepo = nil
		_ = openoctadb.CloseDB()
	})

	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	jsonPath := filepath.Join(dir, "agents", "approvals", "approvals.json")
	q1, err := NewApprovalQueue(jsonPath)
	if err != nil {
		t.Fatalf("NewApprovalQueue: %v", err)
	}
	rec, err := q1.Request("session-db-1", "Bash(ls -la)", nil)
	if err != nil {
		t.Fatalf("Request: %v", err)
	}
	if rec.State != ApprovalPending {
		t.Fatalf("expected pending, got %s", rec.State)
	}

	ClearApprovalQueues()
	q2, err := NewApprovalQueue(jsonPath)
	if err != nil {
		t.Fatalf("reload queue: %v", err)
	}
	got, ok := q2.GetRecord(rec.ID)
	if !ok {
		t.Fatal("expected record after restart")
	}
	if got.Command != "Bash(ls -la)" || got.SessionID != "session-db-1" {
		t.Fatalf("unexpected record: %+v", got)
	}
	if q2.StoreBackend() != "db" {
		t.Fatalf("expected db backend, got %s", q2.StoreBackend())
	}
}

func TestApprovalQueueImportFromJSON(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", dir)
	t.Cleanup(func() {
		ClearApprovalQueues()
		sharedApprovalRepo = nil
		_ = openoctadb.CloseDB()
	})

	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	jsonPath := filepath.Join(dir, "agents", "approvals", "approvals.json")
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		t.Fatal(err)
	}
	snapshot := approvalSnapshot{
		Records: []*ApprovalRecord{
			{
				ID:          "legacy-1",
				SessionID:   "session-legacy",
				Command:     "Bash(rm -rf /tmp/x)",
				State:       ApprovalPending,
				RequestedAt: time.Now().UTC(),
			},
		},
		Whitelist: map[string]time.Time{
			"session-legacy": time.Now().Add(time.Hour).UTC(),
		},
	}
	data, _ := json.MarshalIndent(snapshot, "", "  ")
	if err := os.WriteFile(jsonPath, data, 0o600); err != nil {
		t.Fatal(err)
	}

	q, err := NewApprovalQueue(jsonPath)
	if err != nil {
		t.Fatalf("NewApprovalQueue: %v", err)
	}
	rec, ok := q.GetRecord("legacy-1")
	if !ok {
		t.Fatal("expected imported record")
	}
	if rec.Command != "Bash(rm -rf /tmp/x)" {
		t.Fatalf("unexpected command: %s", rec.Command)
	}
	if _, ok := q.WhitelistSnapshot()["session-legacy"]; !ok {
		t.Fatal("expected imported whitelist")
	}
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("expected json file removed after import, err=%v", err)
	}
}

func TestApprovalQueueApproveWritesStep(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", dir)
	t.Cleanup(func() {
		ClearApprovalQueues()
		sharedApprovalRepo = nil
		_ = openoctadb.CloseDB()
	})
	if err := openoctadb.InitDB(dir); err != nil {
		t.Fatal(err)
	}
	jsonPath := filepath.Join(dir, "agents", "approvals", "approvals.json")
	q, err := NewApprovalQueue(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	rec, err := q.Request("session-approve", "Bash(echo hi)", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Approve(rec.ID, "admin", time.Minute); err != nil {
		t.Fatal(err)
	}

	var stepCount int
	if err := openoctadb.GetDB().QueryRow(`SELECT COUNT(*) FROM approval_steps WHERE approval_id = ?`, rec.ID).Scan(&stepCount); err != nil {
		t.Fatal(err)
	}
	if stepCount != 1 {
		t.Fatalf("expected 1 approval step, got %d", stepCount)
	}
}
