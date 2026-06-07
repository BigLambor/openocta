package http

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/openocta/openocta/pkg/cron"
	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/session"
)

func TestSQLiteMigration(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Prepare JSON files for migration
	// A. Prepare ops health signals and snapshots JSON
	opsDir := filepath.Join(tempDir, "ops")
	if err := os.MkdirAll(opsDir, 0755); err != nil {
		t.Fatalf("MkdirAll ops: %v", err)
	}
	signalsPath := filepath.Join(opsDir, "health_signals.json")
	snapshotsPath := filepath.Join(opsDir, "health_snapshots.json")

	testSignals := []ops.HealthSignal{
		{ObjectType: "cluster", ObjectID: "cluster-test-1", Type: "metrics", Source: "test-src", Status: "healthy"},
	}
	testSnapshots := []ops.HealthSnapshot{
		{ObjectType: "cluster", ObjectID: "cluster-test-1", Domain: "hadoop", ScoreStatus: "ok"},
	}

	sigData, _ := json.Marshal(map[string]interface{}{"version": 1, "items": testSignals})
	snapData, _ := json.Marshal(map[string]interface{}{"version": 1, "items": testSnapshots})
	_ = os.WriteFile(signalsPath, sigData, 0644)
	_ = os.WriteFile(snapshotsPath, snapData, 0644)

	// B. Prepare cron jobs JSON
	cronDir := filepath.Join(tempDir, "cron")
	if err := os.MkdirAll(cronDir, 0755); err != nil {
		t.Fatalf("MkdirAll cron: %v", err)
	}
	jobsPath := filepath.Join(cronDir, "jobs.json")
	testJobs := []cron.CronJob{
		{ID: "job-test-1", Name: "Test Job", Enabled: true},
	}
	jobsData, _ := json.Marshal(map[string]interface{}{"version": 1, "jobs": testJobs})
	_ = os.WriteFile(jobsPath, jobsData, 0644)

	// C. Prepare sessions JSON
	sessionDir := filepath.Join(tempDir, "sessions")
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatalf("MkdirAll sessions: %v", err)
	}
	sessionsPath := filepath.Join(sessionDir, "sessions.json")
	testSessions := session.SessionStore{
		"session-key-1": {SessionID: "sess-1", Label: "Test Session"},
	}
	sessionsData, _ := json.Marshal(testSessions)
	_ = os.WriteFile(sessionsPath, sessionsData, 0644)

	// 2. Initialize openocta.db
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}
	sqliteDB := db.GetDB()

	// 3. Trigger initializations which trigger migration
	if err := ops.InitHealthStore(tempDir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	cronSvc, err := cron.NewService(jobsPath)
	if err != nil {
		t.Fatalf("cron.NewService: %v", err)
	}
	_, err = session.LoadSessionStore(sessionsPath)
	if err != nil {
		t.Fatalf("LoadSessionStore: %v", err)
	}

	// 4. Verify DB records
	// A. Verify ops signals
	var sigCount int
	err = sqliteDB.QueryRow("SELECT COUNT(*) FROM health_signals").Scan(&sigCount)
	if err != nil || sigCount != 1 {
		t.Errorf("Expected 1 signal in DB, got count=%d, err=%v", sigCount, err)
	}
	// B. Verify ops snapshots
	var snapCount int
	err = sqliteDB.QueryRow("SELECT COUNT(*) FROM health_snapshots").Scan(&snapCount)
	if err != nil || snapCount != 1 {
		t.Errorf("Expected 1 snapshot in DB, got count=%d, err=%v", snapCount, err)
	}
	// C. Verify cron jobs
	jobs, err := cronSvc.List(true)
	if err != nil || len(jobs) < 1 { // default jobs might be added too, so check >= 1
		t.Errorf("Expected cron jobs to be migrated, got list=%+v, err=%v", jobs, err)
	}
	var jobCount int
	err = sqliteDB.QueryRow("SELECT COUNT(*) FROM cron_jobs WHERE id = 'job-test-1'").Scan(&jobCount)
	if err != nil || jobCount != 1 {
		t.Errorf("Expected test job to be in DB, got count=%d, err=%v", jobCount, err)
	}
	// D. Verify sessions
	var sessCount int
	err = sqliteDB.QueryRow("SELECT COUNT(*) FROM sessions WHERE session_key = 'session-key-1'").Scan(&sessCount)
	if err != nil || sessCount != 1 {
		t.Errorf("Expected test session to be in DB, got count=%d, err=%v", sessCount, err)
	}

	// 5. Verify JSON files renamed to .bak
	if _, err := os.Stat(signalsPath); !os.IsNotExist(err) {
		t.Errorf("Expected signals JSON to be renamed/deleted, but it exists")
	}
	if _, err := os.Stat(signalsPath + ".bak"); err != nil {
		t.Errorf("Expected signals JSON .bak backup file to exist")
	}
	if _, err := os.Stat(jobsPath); !os.IsNotExist(err) {
		t.Errorf("Expected jobs JSON to be renamed/deleted, but it exists")
	}
	if _, err := os.Stat(jobsPath + ".bak"); err != nil {
		t.Errorf("Expected jobs JSON .bak backup file to exist")
	}
	if _, err := os.Stat(sessionsPath); !os.IsNotExist(err) {
		t.Errorf("Expected sessions JSON to be renamed/deleted, but it exists")
	}
	if _, err := os.Stat(sessionsPath + ".bak"); err != nil {
		t.Errorf("Expected sessions JSON .bak backup file to exist")
	}
}

func TestSQLiteSessionConcurrentWrite(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}

	sessionsPath := filepath.Join(tempDir, "sessions", "sessions.json")
	_, err := session.LoadSessionStore(sessionsPath)
	if err != nil {
		t.Fatalf("LoadSessionStore: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10
	iterations := 20

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				// Each iteration, Load and Update under mutex lock
				session.SessionMu.Lock()
				localStore, err := session.LoadSessionStore(sessionsPath)
				if err != nil {
					session.SessionMu.Unlock()
					t.Errorf("Concurrent LoadSessionStore failed: %v", err)
					return
				}
				key := fmt.Sprintf("worker-%d-sess-%d", workerID, j)
				localStore[key] = session.SessionEntry{
					SessionID: fmt.Sprintf("sess-%d", j),
					Label:     "Concurrent test label",
				}
				err = session.SaveSessionStore(sessionsPath, localStore)
				session.SessionMu.Unlock()
				if err != nil {
					t.Errorf("Concurrent SaveSessionStore failed: %v", err)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// Verify the final session count is correct
	finalStore, err := session.LoadSessionStore(sessionsPath)
	if err != nil {
		t.Fatalf("Final LoadSessionStore failed: %v", err)
	}
	expectedCount := workers * iterations
	if len(finalStore) != expectedCount {
		t.Errorf("Expected %d sessions, got %d", expectedCount, len(finalStore))
	}
}
