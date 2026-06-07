package cron

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestCronJSONService(t *testing.T) {
	// Ensure DB is not initialized to use JSON backend
	_ = db.CloseDB()

	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "jobs.json")

	svc, err := NewService(storePath)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// 1. Add a job
	input := JobCreate{
		Name:        "Test JSON Job",
		Description: "A description",
		AgentID:     "main",
		Schedule:    CronSchedule{Kind: "every", EveryMs: 60000},
		Payload:     CronPayload{Kind: "systemEvent", Text: "hello"},
		Enabled:     true,
	}

	job, err := svc.Add(input)
	if err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	if job.Name != "Test JSON Job" || !job.Enabled {
		t.Errorf("unexpected job: %+v", job)
	}

	// 2. List jobs
	jobs, err := svc.List(true)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) < 1 {
		t.Fatalf("expected at least 1 job, got %d", len(jobs))
	}

	// 3. Get job
	fetched, ok := svc.GetJob(job.ID)
	if !ok {
		t.Fatalf("failed to get job by id %s", job.ID)
	}
	if fetched.Name != job.Name {
		t.Errorf("expected job name %q, got %q", job.Name, fetched.Name)
	}

	// 4. Update job
	enabled := false
	patched, err := svc.Update(job.ID, JobPatch{Enabled: &enabled})
	if err != nil {
		t.Fatalf("failed to update job: %v", err)
	}
	if patched.Enabled {
		t.Errorf("expected job to be disabled")
	}

	// 5. Remove job
	err = svc.Remove(job.ID)
	if err != nil {
		t.Fatalf("failed to remove job: %v", err)
	}
	_, ok = svc.GetJob(job.ID)
	if ok {
		t.Errorf("expected job to be removed")
	}
}

func TestCronSQLiteService(t *testing.T) {
	// Initialize DB to use SQLite backend
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		_ = db.CloseDB()
	}()

	storePath := filepath.Join(tempDir, "jobs.json")
	svc, err := NewService(storePath)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// 1. Add a job
	input := JobCreate{
		Name:        "Test SQLite Job",
		Description: "A description",
		AgentID:     "main",
		Schedule:    CronSchedule{Kind: "every", EveryMs: 60000},
		Payload:     CronPayload{Kind: "systemEvent", Text: "hello"},
		Enabled:     true,
	}

	job, err := svc.Add(input)
	if err != nil {
		t.Fatalf("failed to add job: %v", err)
	}

	if job.Name != "Test SQLite Job" || !job.Enabled {
		t.Errorf("unexpected job: %+v", job)
	}

	// 2. List jobs
	jobs, err := svc.List(true)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}
	if len(jobs) < 1 {
		t.Fatalf("expected at least 1 job, got %d", len(jobs))
	}

	// 3. Get job
	fetched, ok := svc.GetJob(job.ID)
	if !ok {
		t.Fatalf("failed to get job by id %s", job.ID)
	}
	if fetched.Name != job.Name {
		t.Errorf("expected job name %q, got %q", job.Name, fetched.Name)
	}

	// 4. Update job
	enabled := false
	patched, err := svc.Update(job.ID, JobPatch{Enabled: &enabled})
	if err != nil {
		t.Fatalf("failed to update job: %v", err)
	}
	if patched.Enabled {
		t.Errorf("expected job to be disabled")
	}

	// 5. Remove job
	err = svc.Remove(job.ID)
	if err != nil {
		t.Fatalf("failed to remove job: %v", err)
	}
	_, ok = svc.GetJob(job.ID)
	if ok {
		t.Errorf("expected job to be removed")
	}
}

func TestCronMigration(t *testing.T) {
	// First run in JSON mode
	_ = db.CloseDB()

	tempDir := t.TempDir()
	storePath := filepath.Join(tempDir, "jobs.json")

	svcJSON, err := NewService(storePath)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	// Add a job in JSON mode
	job1, err := svcJSON.Add(JobCreate{
		Name:     "Migrated Job",
		AgentID:  "main",
		Schedule: CronSchedule{Kind: "every", EveryMs: 60000},
		Payload:  CronPayload{Kind: "systemEvent", Text: "migrated"},
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("failed to add job in JSON mode: %v", err)
	}

	// Verify JSON file exists
	if _, err := os.Stat(storePath); err != nil {
		t.Fatalf("JSON store file does not exist: %v", err)
	}

	// Now switch to SQLite mode by initializing the DB
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		_ = db.CloseDB()
	}()

	svcSQL, err := NewService(storePath)
	if err != nil {
		t.Fatalf("failed to create SQLite service: %v", err)
	}

	// Verify the job was migrated
	fetched, ok := svcSQL.GetJob(job1.ID)
	if !ok {
		t.Fatalf("failed to find migrated job %s in SQLite", job1.ID)
	}
	if fetched.Name != "Migrated Job" {
		t.Errorf("expected migrated job name 'Migrated Job', got %q", fetched.Name)
	}

	// Verify old JSON file has been renamed to .bak
	if _, err := os.Stat(storePath); err == nil {
		t.Errorf("old JSON file should have been renamed/deleted, but still exists")
	}
	if _, err := os.Stat(storePath + ".bak"); err != nil {
		t.Errorf("bak file does not exist: %v", err)
	}
}

func TestCronConcurrent(t *testing.T) {
	// Initialize DB
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		_ = db.CloseDB()
	}()

	storePath := filepath.Join(tempDir, "jobs.json")
	svc, err := NewService(storePath)
	if err != nil {
		t.Fatalf("failed to create service: %v", err)
	}

	var wg sync.WaitGroup
	workers := 10
	jobsPerWorker := 5

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < jobsPerWorker; j++ {
				// Add a job
				job, err := svc.Add(JobCreate{
					Name:     "Concurrent Job",
					AgentID:  "main",
					Schedule: CronSchedule{Kind: "every", EveryMs: 60000},
					Payload:  CronPayload{Kind: "systemEvent", Text: "concurrent"},
					Enabled:  true,
				})
				if err != nil {
					t.Errorf("worker %d failed to add job: %v", workerID, err)
					return
				}

				// Immediately update it
				enabled := false
				_, err = svc.Update(job.ID, JobPatch{Enabled: &enabled})
				if err != nil {
					t.Errorf("worker %d failed to update job: %v", workerID, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify we have total expected jobs
	jobs, err := svc.List(true)
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}

	// Plus the default jobs, which ensureDefaultJobs adds (5 default jobs)
	expectedMin := workers * jobsPerWorker
	if len(jobs) < expectedMin {
		t.Errorf("expected at least %d jobs, got %d", expectedMin, len(jobs))
	}
}
