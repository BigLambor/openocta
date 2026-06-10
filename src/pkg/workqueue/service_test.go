package workqueue

import (
	"context"
	"testing"
	"time"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

func TestWorkQueueSubmitAndWaitL1(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()
	if err := jobrun.Init(); err != nil {
		t.Fatalf("jobrun.Init: %v", err)
	}

	cfg := RuntimeConfig{
		MaxConcurrentL2Runs: 2,
		ParentRunTimeoutMs:  60_000,
		TaskLeaseMs:         5_000,
		PollIntervalMs:      200,
		L2RunTimeoutMs:      5_000,
	}
	deps := &ExecutorDeps{
		RunScenario: func(ctx context.Context, scenarioKey, objectID string, opts ops.RunOpts) (ops.InspectionResult, error) {
			return ops.InspectionResult{ScenarioKey: scenarioKey, ScoreStatus: "ok"}, nil
		},
	}
	svc := NewService(db.GetDB(), cfg, deps)

	env := TriggerEnvelope{
		TriggerType:    "cron",
		TriggerRef:     "job-inspect-gbase",
		ScenarioKey:    "ops-gbase-health",
		IdempotencyKey: "cron:job-inspect-gbase:1000",
		ScheduledAtMs:  1000,
	}
	plan, err := BuildPlan(env)
	if err != nil {
		t.Fatalf("BuildPlan: %v", err)
	}
	plan.Steps = []PlanStep{{Tier: TierL1, Action: ActionScenarioInspect}}
	plan.Envelope = env

	if err := svc.persistPlan(plan); err != nil {
		t.Fatalf("persistPlan: %v", err)
	}
	if err := svc.startParentRun(plan); err != nil {
		t.Fatalf("startParentRun: %v", err)
	}
	_ = svc.repo.updatePlanStatus(plan.ID, PlanStatusRunning, "")

	tasks, err := svc.repo.listTasksByPlan(plan.ID)
	if err != nil {
		t.Fatalf("listTasksByPlan: %v", err)
	}
	for _, task := range tasks {
		svc.runTask(task)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := svc.waitForPlan(ctx, plan.ID, plan.ParentRunID)
	if err != nil {
		t.Fatalf("waitForPlan: %v", err)
	}
	if result.Status != PlanStatusSucceeded {
		t.Fatalf("expected succeeded, got %s err=%s", result.Status, result.Error)
	}
}

func TestReclaimStaleTasks(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()

	repo := newRepository(db.GetDB())
	now := time.Now().UnixMilli()
	plan := storedPlan{
		ID: "plan-1", TenantID: "default", Status: PlanStatusRunning,
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.insertPlan(plan); err != nil {
		t.Fatalf("insertPlan: %v", err)
	}
	task := WorkTask{
		ID: "task-1", PlanID: "plan-1", Tier: TierL1, Action: ActionScenarioInspect,
		Status: TaskStatusRunning, LeaseUntil: now - 1000, WorkerID: "w1",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.insertTask(task); err != nil {
		t.Fatalf("insertTask: %v", err)
	}
	n, err := repo.reclaimStaleTasks(now)
	if err != nil {
		t.Fatalf("reclaim: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 reclaimed, got %d", n)
	}
}
