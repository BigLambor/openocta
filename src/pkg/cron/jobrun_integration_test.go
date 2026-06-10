package cron

import (
	"testing"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
)

func TestCronRunPersistsJobRun(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := jobrun.Init(); err != nil {
		t.Fatalf("jobrun.Init: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })

	storePath := tempDir + "/cron/jobs.json"
	svc, err := NewService(storePath)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	job, err := svc.Add(JobCreate{
		Name:          "JobRun Test",
		AgentID:       "main",
		Schedule:      CronSchedule{Kind: "every", EveryMs: 60000},
		Payload:       CronPayload{Kind: "systemEvent", Text: "ping"},
		SessionTarget: "main",
		Enabled:       true,
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	before, _ := jobrun.Default().ListByJobID(job.ID, 10)
	if err := svc.Run(job.ID, "force", "hadoop", "", "", ""); err != nil {
		t.Fatalf("Run: %v", err)
	}
	after, err := jobrun.Default().ListByJobID(job.ID, 10)
	if err != nil {
		t.Fatalf("ListByJobID: %v", err)
	}
	if len(after) != len(before)+1 {
		t.Fatalf("expected one new job run, before=%d after=%d", len(before), len(after))
	}
	if after[0].Status != jobrun.StatusSucceeded && after[0].Status != jobrun.StatusFailed {
		t.Fatalf("unexpected run status: %s", after[0].Status)
	}
}
