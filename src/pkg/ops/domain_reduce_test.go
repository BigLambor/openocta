package ops

import (
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestBuildRuleDomainReduce(t *testing.T) {
	score := 45
	inputs := []DomainReduceObjectInput{
		{
			ObjectID: "job_a", ObjectType: HealthObjectJob, L2Status: "succeeded",
			Signal: HealthSignal{
				ObjectID: "job_a", Score: &score, Status: HealthStatusCritical,
				Evidence: map[string]interface{}{"diagnosis": "严重积压"},
			},
		},
		{
			ObjectID: "job_b", ObjectType: HealthObjectJob, L2Status: "failed", L2Error: "timeout",
			Signal: HealthSignal{
				ObjectID: "job_b", Score: &score,
				Evidence: map[string]interface{}{"diagnosis": "频繁重启"},
			},
		},
	}
	result := BuildRuleDomainReduce("run-parent", "plan-1", ScenarioFlinkHealth, "job-inspect-flink", inputs)
	if result.EscalatedCount != 2 || result.L2Succeeded != 1 || result.L2Failed != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if len(result.TopRisks) < 2 {
		t.Fatalf("expected top risks, got %+v", result.TopRisks)
	}
	if result.DomainSnapshotID != FlinkDomainSnapshotID {
		t.Fatalf("snapshot id = %q", result.DomainSnapshotID)
	}
}

func TestPersistDomainReduceSummary(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := db.InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}

	result := BuildRuleDomainReduce("run-reduce", "plan-1", ScenarioSparkHealth, "job-inspect-spark", nil)
	if err := PersistDomainReduceSummary(result); err != nil {
		t.Fatalf("PersistDomainReduceSummary: %v", err)
	}

	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatalf("ListHealthSignals: %v", err)
	}
	found := false
	for _, s := range signals {
		if s.Source == SourceDomainReduce && s.ObjectID == SparkDomainSnapshotID {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected domain reduce signal")
	}
}
