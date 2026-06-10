package cron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestJobRepositoryStoresInNormalizedTables(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })

	repo := newJobRepository(db.GetDB())
	next := int64(2000)
	job := CronJob{
		ID:            "job-normalized-1",
		AgentID:       "main",
		Name:          "Normalized Job",
		Enabled:       true,
		CreatedAtMs:   1000,
		UpdatedAtMs:   1000,
		SessionTarget: "main",
		WakeMode:      "next-heartbeat",
		Schedule:      CronSchedule{Kind: "every", EveryMs: 60000},
		Payload:       CronPayload{Kind: "systemEvent", Text: "ping"},
		State:         CronJobState{NextRunAtMs: &next},
	}
	if err := repo.Upsert(job); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	var kind, scheduleExpr string
	if err := db.GetDB().QueryRow(`
		SELECT kind, schedule_expr FROM jobs WHERE id = ?
	`, job.ID).Scan(&kind, &scheduleExpr); err != nil {
		t.Fatalf("query jobs: %v", err)
	}
	if kind != jobKindCron || scheduleExpr != "every:60000" {
		t.Fatalf("unexpected jobs row kind=%q schedule_expr=%q", kind, scheduleExpr)
	}

	var scheduleKind string
	if err := db.GetDB().QueryRow(`
		SELECT schedule_kind FROM job_schedules WHERE job_id = ?
	`, job.ID).Scan(&scheduleKind); err != nil {
		t.Fatalf("query job_schedules: %v", err)
	}
	if scheduleKind != "every" {
		t.Fatalf("unexpected schedule_kind: %q", scheduleKind)
	}

	loaded, ok, err := repo.Get(job.ID)
	if err != nil || !ok {
		t.Fatalf("Get: ok=%v err=%v", ok, err)
	}
	if loaded.Name != job.Name || loaded.Schedule.EveryMs != 60000 {
		t.Fatalf("unexpected loaded job: %+v", loaded)
	}
}

func TestJobRepositoryImportsLegacyCronJobsBlob(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })

	legacy := CronJob{
		ID:            "legacy-blob-job",
		AgentID:       "main",
		Name:          "Legacy Blob",
		Enabled:       true,
		CreatedAtMs:   1000,
		UpdatedAtMs:   1000,
		SessionTarget: "main",
		WakeMode:      "next-heartbeat",
		Schedule:      CronSchedule{Kind: "every", EveryMs: 30000},
		Payload:       CronPayload{Kind: "systemEvent", Text: "legacy"},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := db.GetDB().Exec(`INSERT INTO cron_jobs (id, detail_json) VALUES (?, ?)`, legacy.ID, string(b)); err != nil {
		t.Fatalf("seed cron_jobs: %v", err)
	}

	repo := newJobRepository(db.GetDB())
	imported, err := repo.ImportLegacyCronJobs()
	if err != nil {
		t.Fatalf("ImportLegacyCronJobs: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1 imported job, got %d", imported)
	}

	var cronJobsCount int
	if err := db.GetDB().QueryRow(`SELECT COUNT(*) FROM cron_jobs`).Scan(&cronJobsCount); err != nil {
		t.Fatalf("count cron_jobs: %v", err)
	}
	if cronJobsCount != 0 {
		t.Fatalf("expected cron_jobs cleared after import, got %d", cronJobsCount)
	}

	loaded, ok, err := repo.Get(legacy.ID)
	if err != nil || !ok {
		t.Fatalf("Get imported job: ok=%v err=%v", ok, err)
	}
	if loaded.Name != legacy.Name {
		t.Fatalf("unexpected imported name: %s", loaded.Name)
	}
}

func TestJobRepositoryImportJSONIsIdempotent(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })

	storePath := filepath.Join(tempDir, "cron", "jobs.json")
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	job := CronJob{
		ID:            "json-import-job",
		AgentID:       "main",
		Name:          "JSON Import",
		Enabled:       true,
		CreatedAtMs:   1000,
		UpdatedAtMs:   1000,
		SessionTarget: "main",
		WakeMode:      "next-heartbeat",
		Schedule:      CronSchedule{Kind: "every", EveryMs: 45000},
		Payload:       CronPayload{Kind: "systemEvent", Text: "json"},
	}
	if err := SaveStore(storePath, &StoreFile{Version: 1, Jobs: []CronJob{job}}); err != nil {
		t.Fatalf("SaveStore: %v", err)
	}

	repo := newJobRepository(db.GetDB())
	imported, err := repo.ImportJSON(storePath)
	if err != nil {
		t.Fatalf("ImportJSON: %v", err)
	}
	if imported != 1 {
		t.Fatalf("expected 1 imported job, got %d", imported)
	}
	if _, err := os.Stat(storePath); !os.IsNotExist(err) {
		t.Fatalf("expected jobs.json moved after import")
	}

	importedAgain, err := repo.ImportJSON(storePath)
	if err != nil {
		t.Fatalf("ImportJSON second: %v", err)
	}
	if importedAgain != 0 {
		t.Fatalf("expected idempotent import, got %d", importedAgain)
	}
}

func hasBackupWithPrefix(t *testing.T, dir, prefix string) bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), prefix) {
			return true
		}
	}
	return false
}
