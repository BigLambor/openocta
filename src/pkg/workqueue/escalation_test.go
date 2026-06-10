package workqueue

import (
	"context"
	"testing"
	"time"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

func TestLastSuccessfulL2At(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()

	repo := newRepository(db.GetDB())
	now := time.Now().UnixMilli()
	plan := storedPlan{
		ID: "plan-1", TenantID: "default", TriggerType: jobrun.TriggerCron,
		ScenarioKey: ops.ScenarioFlinkHealth, ParentRunID: "run-1",
		Status: PlanStatusRunning, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.insertPlan(plan); err != nil {
		t.Fatalf("insertPlan: %v", err)
	}
	task := WorkTask{
		ID: "task-1", PlanID: "plan-1", Tier: TierL2, Action: ActionAIDiagnose,
		ObjectType: ops.HealthObjectJob, ObjectID: "job_risk_calc",
		Status: TaskStatusSucceeded, UpdatedAt: now,
		CreatedAt: now,
	}
	if err := repo.insertTask(task); err != nil {
		t.Fatalf("insertTask: %v", err)
	}

	last, ok := repo.lastSuccessfulL2At(ops.ScenarioFlinkHealth, ops.HealthObjectJob, "job_risk_calc", jobrun.TriggerAlert)
	if !ok || last != now {
		t.Fatalf("lastSuccessfulL2At = (%d, %v), want (%d, true)", last, ok, now)
	}
}

func TestEnqueueFlinkEscalationsCreatesL2Tasks(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()
	if err := ops.InitHealthStore(tempDir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}

	parentRunID := "run-flink-escalation"
	if _, err := ops.RunFlinkHealthL0(context.Background(), ops.FlinkL0Opts{
		RunID: parentRunID, ScenarioKey: ops.ScenarioFlinkHealth,
	}); err != nil {
		t.Fatalf("RunFlinkHealthL0: %v", err)
	}

	now := time.Now().UnixMilli()
	planID := "plan-escalation"
	plan := storedPlan{
		ID: planID, TenantID: "default", TriggerType: jobrun.TriggerCron,
		ScenarioKey: ops.ScenarioFlinkHealth, ParentRunID: parentRunID,
		IdempotencyKey: "cron:job-inspect-flink:1000",
		Status: PlanStatusRunning, CreatedAt: now, UpdatedAt: now,
	}
	repo := newRepository(db.GetDB())
	if err := repo.insertPlan(plan); err != nil {
		t.Fatalf("insertPlan: %v", err)
	}

	svc := NewService(db.GetDB(), RuntimeConfig{
		MaxL2PerParentRun:   50,
		DefaultL2CooldownMs: 0,
	}, &ExecutorDeps{})

	env := TriggerEnvelope{
		TriggerType: jobrun.TriggerCron, ScenarioKey: ops.ScenarioFlinkHealth,
	}
	if err := svc.enqueueFlinkEscalations(plan, env); err != nil {
		t.Fatalf("enqueueFlinkEscalations: %v", err)
	}

	tasks, err := repo.listTasksByPlan(planID)
	if err != nil {
		t.Fatalf("listTasksByPlan: %v", err)
	}
	queued := 0
	for _, task := range tasks {
		if task.Tier == TierL2 && task.Status == TaskStatusQueued {
			queued++
		}
	}
	if queued == 0 {
		t.Fatal("expected escalated L2 tasks")
	}
}

func TestEnqueueFlinkEscalationsRespectsCooldown(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()
	if err := ops.InitHealthStore(tempDir); err != nil {
		t.Fatalf("InitHealthStore: %v", err)
	}

	parentRunID := "run-flink-cooldown"
	if _, err := ops.RunFlinkHealthL0(context.Background(), ops.FlinkL0Opts{
		RunID: parentRunID, ScenarioKey: ops.ScenarioFlinkHealth,
	}); err != nil {
		t.Fatalf("RunFlinkHealthL0: %v", err)
	}

	now := time.Now().UnixMilli()
	repo := newRepository(db.GetDB())
	cooldownPlan := storedPlan{
		ID: "plan-cooldown", TenantID: "default", TriggerType: jobrun.TriggerCron,
		ScenarioKey: ops.ScenarioFlinkHealth, ParentRunID: "run-old",
		Status: PlanStatusSucceeded, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.insertPlan(cooldownPlan); err != nil {
		t.Fatalf("insertPlan cooldown: %v", err)
	}
	if err := repo.insertTask(WorkTask{
		ID: "task-old", PlanID: "plan-cooldown", Tier: TierL2, Action: ActionAIDiagnose,
		ObjectType: ops.HealthObjectJob, ObjectID: "job_risk_calc",
		Status: TaskStatusSucceeded, UpdatedAt: now - 1000,
		CreatedAt: now - 2000,
	}); err != nil {
		t.Fatalf("insert cooldown task: %v", err)
	}

	planID := "plan-new"
	plan := storedPlan{
		ID: planID, TenantID: "default", TriggerType: jobrun.TriggerCron,
		ScenarioKey: ops.ScenarioFlinkHealth, ParentRunID: parentRunID,
		IdempotencyKey: "cron:job-inspect-flink:2000",
		Status: PlanStatusRunning, CreatedAt: now, UpdatedAt: now,
	}
	if err := repo.insertPlan(plan); err != nil {
		t.Fatalf("insertPlan: %v", err)
	}

	svc := NewService(db.GetDB(), RuntimeConfig{
		MaxL2PerParentRun:   50,
		DefaultL2CooldownMs: 3600000,
	}, &ExecutorDeps{})

	if err := svc.enqueueFlinkEscalations(plan, TriggerEnvelope{
		TriggerType: jobrun.TriggerCron, ScenarioKey: ops.ScenarioFlinkHealth,
	}); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	tasks, err := repo.listTasksByPlan(planID)
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	for _, task := range tasks {
		if task.ObjectID == "job_risk_calc" && task.Status == TaskStatusQueued {
			t.Fatal("cooldown should block job_risk_calc escalation")
		}
	}
}
