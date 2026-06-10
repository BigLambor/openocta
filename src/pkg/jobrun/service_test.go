package jobrun

import (
	"testing"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

func initTestJobRunService(t *testing.T) *Service {
	t.Helper()
	if err := openoctadb.InitDB(t.TempDir()); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = openoctadb.CloseDB() })
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	svc := Default()
	if svc == nil {
		t.Fatal("expected default job run service")
	}
	return svc
}

func TestJobRunLifecycle(t *testing.T) {
	svc := initTestJobRunService(t)

	run, err := svc.Start(StartInput{
		JobID:       "job-inspect-hadoop",
		TriggerType: TriggerCron,
		TriggerRef:  "ops-hadoop-health",
		Input: map[string]interface{}{
			"domain": "hadoop",
		},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if run.Status != StatusRunning || run.JobID != "job-inspect-hadoop" {
		t.Fatalf("unexpected run: %+v", run)
	}

	if _, err := svc.AddStep(run.ID, StepInput{
		Kind:          "tool",
		Name:          "query_vm_metrics",
		Status:        StatusSucceeded,
		OutputSummary: "metrics ok",
	}); err != nil {
		t.Fatalf("AddStep: %v", err)
	}

	if err := svc.Succeed(run.ID, FinishInput{
		Output: map[string]interface{}{"status": "ok", "score": 88},
	}); err != nil {
		t.Fatalf("Succeed: %v", err)
	}

	loaded, err := svc.Get(run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if loaded.Status != StatusSucceeded || loaded.FinishedAt == 0 {
		t.Fatalf("unexpected loaded run: %+v", loaded)
	}

	steps, err := svc.ListSteps(run.ID)
	if err != nil {
		t.Fatalf("ListSteps: %v", err)
	}
	if len(steps) != 1 || steps[0].Name != "query_vm_metrics" {
		t.Fatalf("unexpected steps: %+v", steps)
	}

	runs, err := svc.ListByJobID("job-inspect-hadoop", 10)
	if err != nil || len(runs) != 1 {
		t.Fatalf("ListByJobID: %+v err=%v", runs, err)
	}
}

func TestJobRunListFilter(t *testing.T) {
	svc := initTestJobRunService(t)

	if _, err := svc.Start(StartInput{JobID: "job-a", TriggerType: TriggerCron}); err != nil {
		t.Fatalf("Start job-a: %v", err)
	}
	runB, err := svc.Start(StartInput{JobID: "job-b", TriggerType: TriggerInspection, TriggerRef: "ops-fi-health"})
	if err != nil {
		t.Fatalf("Start job-b: %v", err)
	}

	runs, err := svc.List(ListFilter{JobID: "job-b", Limit: 10})
	if err != nil || len(runs) != 1 || runs[0].ID != runB.ID {
		t.Fatalf("List by job: %+v err=%v", runs, err)
	}

	runs, err = svc.List(ListFilter{TriggerType: TriggerInspection, Limit: 10})
	if err != nil || len(runs) != 1 {
		t.Fatalf("List by trigger: %+v err=%v", runs, err)
	}

	detail, err := svc.GetDetail(runB.ID)
	if err != nil || detail.Run.ID != runB.ID {
		t.Fatalf("GetDetail: %+v err=%v", detail, err)
	}
}

func TestJobRunFailAndCancel(t *testing.T) {
	svc := initTestJobRunService(t)

	run, err := svc.Start(StartInput{JobID: "job-1", TriggerType: TriggerManual})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := svc.Fail(run.ID, "boom", map[string]interface{}{"status": "error"}); err != nil {
		t.Fatalf("Fail: %v", err)
	}
	loaded, err := svc.Get(run.ID)
	if err != nil || loaded.Status != StatusFailed || loaded.Error != "boom" {
		t.Fatalf("unexpected failed run: %+v err=%v", loaded, err)
	}

	run2, err := svc.Start(StartInput{JobID: "job-2", TriggerType: TriggerManual})
	if err != nil {
		t.Fatalf("Start run2: %v", err)
	}
	if err := svc.Cancel(run2.ID, "user cancelled"); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	loaded2, err := svc.Get(run2.ID)
	if err != nil || loaded2.Status != StatusCancelled {
		t.Fatalf("unexpected cancelled run: %+v err=%v", loaded2, err)
	}
}

func TestJobRunWaitApproval(t *testing.T) {
	svc := initTestJobRunService(t)
	run, err := svc.Start(StartInput{JobID: "job-approval", TriggerType: TriggerInspection})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := svc.WaitApproval(run.ID); err != nil {
		t.Fatalf("WaitApproval: %v", err)
	}
	if err := svc.Succeed(run.ID, FinishInput{Output: map[string]interface{}{"approved": true}}); err != nil {
		t.Fatalf("Succeed after approval: %v", err)
	}
}
