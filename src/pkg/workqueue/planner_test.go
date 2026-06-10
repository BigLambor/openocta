package workqueue

import (
	"testing"
)

func TestBuildPlanNativeAndAgentTurn(t *testing.T) {
	env := TriggerEnvelope{
		TriggerType:    "cron",
		TriggerRef:     "job-inspect-hadoop",
		ScenarioKey:    "ops-bch-health",
		IdempotencyKey: "cron:job-inspect-hadoop:1700000000000",
		ScheduledAtMs:  1_700_000_000_000,
		CronJob: CronJobSnapshot{
			ID:            "job-inspect-hadoop",
			SessionTarget: "isolated",
			PayloadKind:   "agentTurn",
			PayloadMessage: "inspect",
		},
	}
	plan, err := BuildPlan(env)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Steps) < 2 {
		t.Fatalf("expected L1+L2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tier != TierL1 || plan.Steps[1].Tier != TierL2 {
		t.Fatalf("unexpected steps: %+v", plan.Steps)
	}
}

func TestBuildPlanAgentOnly(t *testing.T) {
	env := TriggerEnvelope{
		TriggerType: "cron",
		TriggerRef:  "custom-job",
		ScenarioKey: "unknown-scenario",
		CronJob: CronJobSnapshot{
			ID:            "custom-job",
			SessionTarget: "isolated",
			PayloadKind:   "agentTurn",
		},
	}
	plan, err := BuildPlan(env)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Tier != TierL2 {
		t.Fatalf("expected single L2 step, got %+v", plan.Steps)
	}
}

func TestBuildPlanBatchL0Only(t *testing.T) {
	cases := []struct {
		scenario string
		ref      string
	}{
		{"ops-flink-health", "job-inspect-flink"},
		{"ops-spark-health", "job-inspect-spark"},
		{"ops-yarn-health", "job-inspect-yarn"},
		{"ops-gbase-instance-health", "job-inspect-gbase-instances"},
		{"ops-dataapps-pipeline-health", "job-inspect-dataapps-pipelines"},
	}
	for _, tc := range cases {
		env := TriggerEnvelope{
			TriggerType: "cron", TriggerRef: tc.ref, ScenarioKey: tc.scenario,
		}
		plan, err := BuildPlan(env)
		if err != nil {
			t.Fatalf("%s BuildPlan: %v", tc.scenario, err)
		}
		if len(plan.Steps) != 1 || plan.Steps[0].Tier != TierL0 {
			t.Fatalf("%s expected L0-only, got %+v", tc.scenario, plan.Steps)
		}
	}
}

func TestBuildPlanFlinkL0Only(t *testing.T) {
	env := TriggerEnvelope{
		TriggerType:    "cron",
		TriggerRef:     "job-inspect-flink",
		ScenarioKey:    "ops-flink-health",
		IdempotencyKey: "cron:job-inspect-flink:1700000000000",
		ScheduledAtMs:  1_700_000_000_000,
		CronJob: CronJobSnapshot{
			ID:            "job-inspect-flink",
			SessionTarget: "isolated",
			PayloadKind:   "agentTurn",
		},
	}
	plan, err := BuildPlan(env)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Steps) != 1 {
		t.Fatalf("expected single L0 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].Tier != TierL0 || plan.Steps[0].Action != ActionCollectAndScore {
		t.Fatalf("unexpected step: %+v", plan.Steps[0])
	}
}

func TestBuildPlanAlertL2Only(t *testing.T) {
	env := TriggerEnvelope{
		TriggerType:    "alert",
		TriggerRef:     "alert-diagnosis",
		ScenarioKey:    "ops-diagnosis",
		IdempotencyKey: "alert:alert-group-001",
		Priority:       PriorityHigh,
	}
	plan, err := BuildPlan(env)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	if len(plan.Steps) != 1 || plan.Steps[0].Tier != TierL2 {
		t.Fatalf("expected single L2 step for alert, got %+v", plan.Steps)
	}
}
