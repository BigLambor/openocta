package ops

import (
	"context"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestRunSparkHealthL0WritesSignals(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	result, err := RunSparkHealthL0(context.Background(), SparkL0Opts{
		RunID: "run-spark-l0", ScenarioKey: ScenarioSparkHealth,
	})
	if err != nil {
		t.Fatalf("RunSparkHealthL0: %v", err)
	}
	if result.JobCount == 0 {
		t.Fatal("expected spark jobs")
	}
	signals, err := ListSignalsForRun("run-spark-l0", ScenarioSparkHealth, HealthObjectJob)
	if err != nil || len(signals) == 0 {
		t.Fatalf("signals: %v err=%v", len(signals), err)
	}
}

func TestRunYarnHealthL0WritesSignals(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	result, err := RunYarnHealthL0(context.Background(), YarnL0Opts{
		RunID: "run-yarn-l0", ScenarioKey: ScenarioYarnHealth,
	})
	if err != nil {
		t.Fatalf("RunYarnHealthL0: %v", err)
	}
	if result.QueueCount == 0 {
		t.Fatal("expected yarn queues")
	}
}

func TestRunGBaseInstanceHealthL0(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	result, err := RunGBaseInstanceHealthL0(context.Background(), GBaseInstanceL0Opts{
		RunID: "run-gbase-l0", ScenarioKey: ScenarioGBaseInstanceHealth,
	})
	if err != nil {
		t.Fatalf("RunGBaseInstanceHealthL0: %v", err)
	}
	if result.InstanceCount != 3 {
		t.Fatalf("expected 3 instances, got %d", result.InstanceCount)
	}
}

func TestRunDataAppsPipelineHealthL0(t *testing.T) {
	_ = db.CloseDB()
	dir := t.TempDir()
	if err := InitHealthStore(dir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}
	result, err := RunDataAppsPipelineHealthL0(context.Background(), DataAppsPipelineL0Opts{
		RunID: "run-pipeline-l0", ScenarioKey: ScenarioDataAppsPipelineHealth,
	})
	if err != nil {
		t.Fatalf("RunDataAppsPipelineHealthL0: %v", err)
	}
	if result.PipelineCount != 4 {
		t.Fatalf("expected 4 pipelines, got %d", result.PipelineCount)
	}
}

func TestIsBatchL0Scenario(t *testing.T) {
	if !IsBatchL0Scenario(ScenarioSparkHealth) {
		t.Fatal("spark should be batch L0")
	}
	if IsBatchL0Scenario("ops-bch-health") {
		t.Fatal("cluster scenario should not be batch L0 only")
	}
}
