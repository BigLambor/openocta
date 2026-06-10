package ops

import (
	"testing"

	"github.com/openocta/openocta/pkg/jobrun"
)

func TestRecordMergedAlertGroupStartsJobRun(t *testing.T) {
	initTestAlertJobRunStore(t)

	g, err := RecordMergedAlertGroup("hadoop-prod", "agent:main:alert:abc", "alert-run-1", []MergedAlertInput{
		{AlertID: "a1", Title: "Hadoop Alert", Severity: "critical", Alertname: "HadoopDown", Service: "hdfs"},
	})
	if err != nil {
		t.Fatalf("RecordMergedAlertGroup: %v", err)
	}
	if g.RunID != "alert-run-1" {
		t.Fatalf("unexpected run id: %s", g.RunID)
	}

	hasStarted := false
	for _, item := range g.Timeline {
		if item.Type == "diagnosis_started" && item.RunID == "alert-run-1" {
			hasStarted = true
		}
	}
	if !hasStarted {
		t.Fatalf("expected diagnosis_started timeline, got %+v", g.Timeline)
	}

	jr := jobrun.Default()
	run, err := jr.Get("alert-run-1")
	if err != nil {
		t.Fatalf("Get job run: %v", err)
	}
	if run.TriggerType != jobrun.TriggerAlert || run.TriggerRef != g.ID || run.JobID != AlertDiagnosisJobID {
		t.Fatalf("unexpected job run: %+v", run)
	}
	if run.Status != jobrun.StatusRunning {
		t.Fatalf("expected running status, got %s", run.Status)
	}
}

func TestBindAlertDiagnosisChatRun(t *testing.T) {
	initTestAlertJobRunStore(t)

	g, err := RecordMergedAlertGroup("gbase-prod", "agent:main:alert:old", "old-run", []MergedAlertInput{
		{AlertID: "a1", Title: "GBase Alert", Severity: "warning"},
	})
	if err != nil {
		t.Fatalf("RecordMergedAlertGroup: %v", err)
	}

	if err := BindAlertDiagnosisChatRun(g.ID, "manual-run-1", "agent:main:alert:manual"); err != nil {
		t.Fatalf("BindAlertDiagnosisChatRun: %v", err)
	}

	loaded, err := GetAlertGroup(g.ID)
	if err != nil {
		t.Fatalf("GetAlertGroup: %v", err)
	}
	if loaded.RunID != "manual-run-1" {
		t.Fatalf("expected updated run id, got %s", loaded.RunID)
	}

	run, err := jobrun.Default().Get("manual-run-1")
	if err != nil {
		t.Fatalf("Get manual run: %v", err)
	}
	if run.TriggerRef != g.ID {
		t.Fatalf("unexpected trigger ref: %s", run.TriggerRef)
	}
}
