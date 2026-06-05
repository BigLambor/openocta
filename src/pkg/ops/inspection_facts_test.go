package ops

import (
	"testing"
	"time"
)

func TestPersistInspectionFactsCreatesGBaseSQLSignalAndSnapshot(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}
	if err := InitHealthStore(dir); err != nil {
		t.Fatal(err)
	}
	cluster, err := CreateCluster(ClusterCreate{
		Name:          "GBase Facts Test",
		Domain:        DomainGBase,
		Status:        "healthy",
		MonitorLabels: `job="gbase-facts"`,
		GBaseDsnRef:   "user:pass@tcp(localhost:3306)/db",
	})
	if err != nil {
		t.Fatal(err)
	}

	score := 88
	report := InspectionReport{
		ID:          "session-gbase-facts",
		JobID:       "job-inspect-gbase",
		Domain:      DomainGBase,
		ClusterID:   cluster.ID,
		Score:       &score,
		ScoreStatus: ScoreStatusWarning,
		ToolRuns: []ToolRunReport{
			{
				ToolName: "query_gbase_slow_sql",
				Success:  true,
				Output:   `[{"sql_text":"select * from t","exec_time_sec":12}]`,
			},
		},
		StartedAt:  time.Now().Add(-time.Second).UnixMilli(),
		FinishedAt: time.Now().UnixMilli(),
	}
	if err := PersistInspectionFacts(report); err != nil {
		t.Fatal(err)
	}

	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatal(err)
	}
	if !hasSignalType(signals, SignalTypeInspection) || !hasSignalType(signals, SignalTypeGBaseSQL) {
		t.Fatalf("expected inspection and gbase_sql signals, got %+v", signals)
	}
	snapshots, err := ListHealthSnapshots()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) == 0 {
		t.Fatal("expected snapshot")
	}
	if snapshots[0].Score == nil {
		t.Fatalf("expected composite score, got %+v", snapshots[0])
	}
	if snapshots[0].ScoreStatus != ScoreStatusOK {
		t.Fatalf("expected ok composite snapshot, got %+v", snapshots[0])
	}
}

func hasSignalType(signals []HealthSignal, typ string) bool {
	for _, s := range signals {
		if s.Type == typ {
			return true
		}
	}
	return false
}
