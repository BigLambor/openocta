package workqueue

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type repository struct {
	db *sql.DB
}

func newRepository(db *sql.DB) *repository {
	if db == nil {
		return nil
	}
	return &repository{db: db}
}

type storedPlan struct {
	ID             string
	TenantID       string
	TriggerType    string
	TriggerRef     string
	ScenarioKey    string
	ParentRunID    string
	Status         string
	Priority       int
	IdempotencyKey string
	ScheduledAtMs  int64
	EnvelopeJSON   string
	PlanJSON       string
	Error          string
	CreatedAt      int64
	UpdatedAt      int64
}

func (r *repository) insertPlan(plan storedPlan) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("work queue repository 未初始化")
	}
	_, err := r.db.Exec(`
		INSERT INTO work_plans (
			id, tenant_id, trigger_type, trigger_ref, scenario_key, parent_run_id,
			status, priority, idempotency_key, scheduled_at_ms,
			envelope_json, plan_json, error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, plan.ID, plan.TenantID, plan.TriggerType, plan.TriggerRef, plan.ScenarioKey, plan.ParentRunID,
		plan.Status, plan.Priority, plan.IdempotencyKey, plan.ScheduledAtMs,
		plan.EnvelopeJSON, plan.PlanJSON, plan.Error, plan.CreatedAt, plan.UpdatedAt)
	return err
}

func (r *repository) getPlanByID(id string) (storedPlan, error) {
	if r == nil || r.db == nil {
		return storedPlan{}, fmt.Errorf("work queue repository 未初始化")
	}
	row := r.db.QueryRow(`
		SELECT id, tenant_id, trigger_type, trigger_ref, scenario_key, parent_run_id,
			status, priority, idempotency_key, scheduled_at_ms,
			envelope_json, plan_json, error, created_at, updated_at
		FROM work_plans WHERE id = ?
	`, strings.TrimSpace(id))
	return scanPlan(row)
}

func (r *repository) getPlanByIdempotency(key string) (storedPlan, error) {
	if r == nil || r.db == nil {
		return storedPlan{}, fmt.Errorf("work queue repository 未初始化")
	}
	key = strings.TrimSpace(key)
	if key == "" {
		return storedPlan{}, sql.ErrNoRows
	}
	row := r.db.QueryRow(`
		SELECT id, tenant_id, trigger_type, trigger_ref, scenario_key, parent_run_id,
			status, priority, idempotency_key, scheduled_at_ms,
			envelope_json, plan_json, error, created_at, updated_at
		FROM work_plans WHERE idempotency_key = ?
	`, key)
	return scanPlan(row)
}

func scanPlan(row *sql.Row) (storedPlan, error) {
	var p storedPlan
	err := row.Scan(
		&p.ID, &p.TenantID, &p.TriggerType, &p.TriggerRef, &p.ScenarioKey, &p.ParentRunID,
		&p.Status, &p.Priority, &p.IdempotencyKey, &p.ScheduledAtMs,
		&p.EnvelopeJSON, &p.PlanJSON, &p.Error, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

func (r *repository) updatePlanStatus(id, status, errMsg string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("work queue repository 未初始化")
	}
	now := time.Now().UnixMilli()
	_, err := r.db.Exec(`
		UPDATE work_plans SET status = ?, error = ?, updated_at = ? WHERE id = ?
	`, status, strings.TrimSpace(errMsg), now, strings.TrimSpace(id))
	return err
}

func (r *repository) insertTask(task WorkTask) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("work queue repository 未初始化")
	}
	inputJSON, _ := json.Marshal(task.Input)
	outputJSON, _ := json.Marshal(task.Output)
	_, err := r.db.Exec(`
		INSERT INTO work_tasks (
			id, plan_id, tenant_id, tier, action, object_type, object_id,
			parent_run_id, child_run_id, status, priority, idempotency_key,
			lease_until, worker_id, input_json, output_json, error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ID, task.PlanID, task.TenantID, task.Tier, task.Action, task.ObjectType, task.ObjectID,
		task.ParentRunID, task.ChildRunID, task.Status, task.Priority, task.IdempotencyKey,
		task.LeaseUntil, task.WorkerID, string(inputJSON), string(outputJSON), task.Error,
		task.CreatedAt, task.UpdatedAt)
	return err
}

func (r *repository) claimNextTask(workerID string, leaseMs int64, l2Running int, maxL2 int) (*WorkTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("work queue repository 未初始化")
	}
	now := time.Now().UnixMilli()
	leaseUntil := now + leaseMs

	tx, err := r.db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.Query(`
		SELECT id, plan_id, tenant_id, tier, action, object_type, object_id,
			parent_run_id, child_run_id, status, priority, idempotency_key,
			lease_until, worker_id, input_json, output_json, error, created_at, updated_at
		FROM work_tasks
		WHERE status = 'queued'
		ORDER BY priority DESC, created_at ASC
		LIMIT 20
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var candidate *WorkTask
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		if task.Tier == TierL2 && l2Running >= maxL2 {
			continue
		}
		candidate = &task
		break
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if candidate == nil {
		return nil, nil
	}

	res, err := tx.Exec(`
		UPDATE work_tasks SET status = ?, worker_id = ?, lease_until = ?, updated_at = ?
		WHERE id = ? AND status = 'queued'
	`, TaskStatusRunning, workerID, leaseUntil, now, candidate.ID)
	if err != nil {
		return nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	candidate.Status = TaskStatusRunning
	candidate.WorkerID = workerID
	candidate.LeaseUntil = leaseUntil
	candidate.UpdatedAt = now
	return candidate, nil
}

func scanTask(scanner interface {
	Scan(dest ...interface{}) error
}) (WorkTask, error) {
	var task WorkTask
	var inputJSON, outputJSON string
	err := scanner.Scan(
		&task.ID, &task.PlanID, &task.TenantID, &task.Tier, &task.Action, &task.ObjectType, &task.ObjectID,
		&task.ParentRunID, &task.ChildRunID, &task.Status, &task.Priority, &task.IdempotencyKey,
		&task.LeaseUntil, &task.WorkerID, &inputJSON, &outputJSON, &task.Error, &task.CreatedAt, &task.UpdatedAt,
	)
	if err != nil {
		return WorkTask{}, err
	}
	_ = json.Unmarshal([]byte(inputJSON), &task.Input)
	_ = json.Unmarshal([]byte(outputJSON), &task.Output)
	if task.Input == nil {
		task.Input = map[string]interface{}{}
	}
	if task.Output == nil {
		task.Output = map[string]interface{}{}
	}
	return task, nil
}

func (r *repository) finishTask(id, status, errMsg string, output map[string]interface{}) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("work queue repository 未初始化")
	}
	outputJSON, _ := json.Marshal(output)
	now := time.Now().UnixMilli()
	_, err := r.db.Exec(`
		UPDATE work_tasks SET status = ?, error = ?, output_json = ?, lease_until = 0, updated_at = ?
		WHERE id = ?
	`, status, strings.TrimSpace(errMsg), string(outputJSON), now, strings.TrimSpace(id))
	return err
}

func (r *repository) listTasksByPlan(planID string) ([]WorkTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("work queue repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT id, plan_id, tenant_id, tier, action, object_type, object_id,
			parent_run_id, child_run_id, status, priority, idempotency_key,
			lease_until, worker_id, input_json, output_json, error, created_at, updated_at
		FROM work_tasks WHERE plan_id = ? ORDER BY created_at ASC
	`, strings.TrimSpace(planID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []WorkTask{}
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func (r *repository) reclaimStaleTasks(nowMs int64) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("work queue repository 未初始化")
	}
	res, err := r.db.Exec(`
		UPDATE work_tasks
		SET status = 'queued', worker_id = '', lease_until = 0, updated_at = ?
		WHERE status = 'running' AND lease_until > 0 AND lease_until < ?
	`, nowMs, nowMs)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (r *repository) lastSuccessfulL2At(scenarioKey, objectType, objectID, excludeTriggerType string) (int64, bool) {
	if r == nil || r.db == nil {
		return 0, false
	}
	scenarioKey = strings.TrimSpace(scenarioKey)
	objectType = strings.TrimSpace(objectType)
	objectID = strings.TrimSpace(objectID)
	if scenarioKey == "" || objectID == "" {
		return 0, false
	}
	var last int64
	err := r.db.QueryRow(`
		SELECT MAX(t.updated_at)
		FROM work_tasks t
		JOIN work_plans p ON p.id = t.plan_id
		WHERE t.tier = ? AND t.status = ?
			AND t.object_type = ? AND t.object_id = ?
			AND p.scenario_key = ?
			AND p.trigger_type != ?
	`, TierL2, TaskStatusSucceeded, objectType, objectID, scenarioKey, strings.TrimSpace(excludeTriggerType)).Scan(&last)
	if err != nil || last <= 0 {
		return 0, false
	}
	return last, true
}

func (r *repository) countRunningL2() (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("work queue repository 未初始化")
	}
	var n int
	err := r.db.QueryRow(`
		SELECT COUNT(*) FROM work_tasks WHERE tier = ? AND status = ?
	`, TierL2, TaskStatusRunning).Scan(&n)
	return n, err
}

func newPlanID() string {
	return uuid.New().String()
}

func newTaskID() string {
	return uuid.New().String()
}
