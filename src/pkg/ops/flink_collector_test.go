package ops

import (
	"context"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestRunFlinkHealthL0WritesSignalsAndSnapshot(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}

	result, err := RunFlinkHealthL0(context.Background(), FlinkL0Opts{
		RunID:       "run-flink-l0-test",
		ScenarioKey: ScenarioFlinkHealth,
		Domain:      DomainHadoop,
	})
	if err != nil {
		t.Fatalf("RunFlinkHealthL0: %v", err)
	}
	if result.JobCount == 0 {
		t.Fatal("expected at least one Flink job")
	}

	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatalf("ListHealthSignals: %v", err)
	}
	jobSignals := 0
	for _, s := range signals {
		if s.ObjectType == HealthObjectJob && s.ScenarioKey == ScenarioFlinkHealth {
			jobSignals++
		}
	}
	if jobSignals != result.JobCount {
		t.Fatalf("expected %d job signals, got %d", result.JobCount, jobSignals)
	}

	snapshots, err := ListHealthSnapshots()
	if err != nil {
		t.Fatalf("ListHealthSnapshots: %v", err)
	}
	foundDomain := false
	for _, snap := range snapshots {
		if snap.ObjectType == HealthObjectDomain && snap.ObjectID == FlinkDomainSnapshotID {
			foundDomain = true
			if snap.Score == nil {
				t.Fatal("domain snapshot missing score")
			}
			break
		}
	}
	if !foundDomain {
		t.Fatalf("expected domain snapshot %q", FlinkDomainSnapshotID)
	}
}

func TestListFlinkJobsHealthPrefersSignals(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	if _, err := RunFlinkHealthL0(context.Background(), FlinkL0Opts{
		RunID: "run-list-test",
	}); err != nil {
		t.Fatalf("RunFlinkHealthL0: %v", err)
	}

	jobs, err := ListFlinkJobsHealth()
	if err != nil {
		t.Fatalf("ListFlinkJobsHealth: %v", err)
	}
	if len(jobs) == 0 {
		t.Fatal("expected jobs from signals")
	}
	for _, j := range jobs {
		if j.ID == "" {
			t.Fatal("job missing id")
		}
	}
}
