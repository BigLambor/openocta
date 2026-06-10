package workqueue

import (
	"testing"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

func TestMaybeEnqueueDomainReduce(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()

	now := nowMs()
	planID := "plan-reduce"
	parentRunID := "run-parent-reduce"
	plan := storedPlan{
		ID: planID, TenantID: "default", TriggerType: jobrun.TriggerCron,
		ScenarioKey: ops.ScenarioFlinkHealth, ParentRunID: parentRunID,
		IdempotencyKey: "cron:job-inspect-flink:3000",
		Status: PlanStatusRunning, CreatedAt: now, UpdatedAt: now,
	}
	repo := newRepository(db.GetDB())
	if err := repo.insertPlan(plan); err != nil {
		t.Fatalf("insertPlan: %v", err)
	}
	if err := repo.insertTask(WorkTask{
		ID: "l0", PlanID: planID, Tier: TierL0, Action: ActionCollectAndScore,
		Status: TaskStatusSucceeded, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("insert l0: %v", err)
	}
	if err := repo.insertTask(WorkTask{
		ID: "l2-1", PlanID: planID, Tier: TierL2, Action: ActionAIDiagnose,
		ObjectType: ops.HealthObjectJob, ObjectID: "job_a", ChildRunID: "child-1",
		Status: TaskStatusSucceeded, CreatedAt: now, UpdatedAt: now,
		Input: map[string]interface{}{"escalation": true},
	}); err != nil {
		t.Fatalf("insert l2: %v", err)
	}

	svc := NewService(db.GetDB(), RuntimeConfig{DomainReduceEnabled: true}, &ExecutorDeps{})
	env := TriggerEnvelope{TriggerType: jobrun.TriggerCron, ScenarioKey: ops.ScenarioFlinkHealth, TriggerRef: "job-inspect-flink"}
	if err := svc.maybeEnqueueDomainReduce(plan, env); err != nil {
		t.Fatalf("maybeEnqueueDomainReduce: %v", err)
	}

	tasks, err := repo.listTasksByPlan(planID)
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	reduceFound := false
	for _, task := range tasks {
		if task.Action == ActionDomainReduce {
			reduceFound = true
		}
	}
	if !reduceFound {
		t.Fatal("expected domain reduce task")
	}
}

func TestMaybeEnqueueDomainReduceDisabled(t *testing.T) {
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() { _ = db.CloseDB() }()

	repo := newRepository(db.GetDB())
	now := nowMs()
	plan := storedPlan{ID: "p1", ParentRunID: "r1", CreatedAt: now, UpdatedAt: now}
	_ = repo.insertPlan(plan)
	_ = repo.insertTask(WorkTask{
		ID: "l2", PlanID: "p1", Tier: TierL2, Action: ActionAIDiagnose,
		Status: TaskStatusSucceeded, Input: map[string]interface{}{"escalation": true},
		CreatedAt: now, UpdatedAt: now,
	})

	svc := NewService(db.GetDB(), RuntimeConfig{DomainReduceEnabled: false}, &ExecutorDeps{})
	if err := svc.maybeEnqueueDomainReduce(plan, TriggerEnvelope{ScenarioKey: ops.ScenarioFlinkHealth}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	tasks, _ := repo.listTasksByPlan("p1")
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
}
