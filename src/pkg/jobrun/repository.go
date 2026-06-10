package jobrun

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type jobRunRepository struct {
	db *sql.DB
}

func newJobRunRepository(db *sql.DB) *jobRunRepository {
	if db == nil {
		return nil
	}
	return &jobRunRepository{db: db}
}

func (r *jobRunRepository) Insert(run JobRun) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("job run repository 未初始化")
	}
	inputJSON, err := json.Marshal(run.Input)
	if err != nil {
		return err
	}
	outputJSON, err := json.Marshal(run.Output)
	if err != nil {
		return err
	}
	_, err = r.db.Exec(`
		INSERT INTO job_runs (
			id, job_id, task_id, parent_run_id, trigger_type, trigger_ref, status,
			started_at, finished_at, error, input_json, output_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, run.ID, run.JobID, run.TaskID, run.ParentRunID, run.TriggerType, run.TriggerRef, run.Status,
		run.StartedAt, run.FinishedAt, run.Error, string(inputJSON), string(outputJSON),
		run.CreatedAt, run.UpdatedAt)
	return err
}

func (r *jobRunRepository) Update(run JobRun) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("job run repository 未初始化")
	}
	inputJSON, err := json.Marshal(run.Input)
	if err != nil {
		return err
	}
	outputJSON, err := json.Marshal(run.Output)
	if err != nil {
		return err
	}
	res, err := r.db.Exec(`
		UPDATE job_runs SET
			job_id = ?, task_id = ?, parent_run_id = ?, trigger_type = ?, trigger_ref = ?, status = ?,
			started_at = ?, finished_at = ?, error = ?, input_json = ?, output_json = ?, updated_at = ?
		WHERE id = ?
	`, run.JobID, run.TaskID, run.ParentRunID, run.TriggerType, run.TriggerRef, run.Status,
		run.StartedAt, run.FinishedAt, run.Error, string(inputJSON), string(outputJSON), run.UpdatedAt, run.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *jobRunRepository) Get(id string) (JobRun, error) {
	if r == nil || r.db == nil {
		return JobRun{}, fmt.Errorf("job run repository 未初始化")
	}
	id = strings.TrimSpace(id)
	rows, err := r.db.Query(`
		SELECT id, job_id, task_id, parent_run_id, trigger_type, trigger_ref, status,
			started_at, finished_at, error, input_json, output_json, created_at, updated_at
		FROM job_runs WHERE id = ?
	`, id)
	if err != nil {
		return JobRun{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return JobRun{}, sql.ErrNoRows
	}
	run, err := scanJobRun(rows)
	if err != nil {
		return JobRun{}, err
	}
	return run, rows.Err()
}

func (r *jobRunRepository) ListByJobID(jobID string, limit int) ([]JobRun, error) {
	return r.List(ListFilter{JobID: jobID, Limit: limit})
}

func (r *jobRunRepository) List(filter ListFilter) ([]JobRun, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("job run repository 未初始化")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	query := `
		SELECT id, job_id, task_id, parent_run_id, trigger_type, trigger_ref, status,
			started_at, finished_at, error, input_json, output_json, created_at, updated_at
		FROM job_runs
		WHERE 1=1`
	args := []interface{}{}
	if jobID := strings.TrimSpace(filter.JobID); jobID != "" {
		query += ` AND job_id = ?`
		args = append(args, jobID)
	}
	if parentRunID := strings.TrimSpace(filter.ParentRunID); parentRunID != "" {
		query += ` AND parent_run_id = ?`
		args = append(args, parentRunID)
	}
	if triggerType := strings.TrimSpace(filter.TriggerType); triggerType != "" {
		query += ` AND trigger_type = ?`
		args = append(args, triggerType)
	}
	if triggerRef := strings.TrimSpace(filter.TriggerRef); triggerRef != "" {
		query += ` AND trigger_ref = ?`
		args = append(args, triggerRef)
	}
	query += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []JobRun{}
	for rows.Next() {
		run, err := scanJobRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func scanJobRun(rows *sql.Rows) (JobRun, error) {
	var run JobRun
	var inputJSON, outputJSON string
	if err := rows.Scan(
		&run.ID, &run.JobID, &run.TaskID, &run.ParentRunID, &run.TriggerType, &run.TriggerRef, &run.Status,
		&run.StartedAt, &run.FinishedAt, &run.Error, &inputJSON, &outputJSON, &run.CreatedAt, &run.UpdatedAt,
	); err != nil {
		return JobRun{}, err
	}
	_ = json.Unmarshal([]byte(inputJSON), &run.Input)
	_ = json.Unmarshal([]byte(outputJSON), &run.Output)
	if run.Input == nil {
		run.Input = map[string]interface{}{}
	}
	if run.Output == nil {
		run.Output = map[string]interface{}{}
	}
	return run, nil
}

func (r *jobRunRepository) InsertStep(step RunStep) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("job run repository 未初始化")
	}
	now := time.Now().UnixMilli()
	_, err := r.db.Exec(`
		INSERT INTO run_steps (
			id, run_id, step_order, kind, name, status, started_at, finished_at, error,
			input_summary, output_summary, metadata_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '{}', ?, ?)
	`, step.ID, step.RunID, step.StepOrder, step.Kind, step.Name, step.Status,
		step.StartedAt, step.FinishedAt, step.Error, step.InputSummary, step.OutputSummary, now, now)
	return err
}

func (r *jobRunRepository) ListSteps(runID string) ([]RunStep, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("job run repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT id, run_id, step_order, kind, name, status, started_at, finished_at, error, input_summary, output_summary
		FROM run_steps
		WHERE run_id = ?
		ORDER BY step_order ASC
	`, strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RunStep{}
	for rows.Next() {
		var step RunStep
		if err := rows.Scan(
			&step.ID, &step.RunID, &step.StepOrder, &step.Kind, &step.Name, &step.Status,
			&step.StartedAt, &step.FinishedAt, &step.Error, &step.InputSummary, &step.OutputSummary,
		); err != nil {
			return nil, err
		}
		out = append(out, step)
	}
	return out, rows.Err()
}

func (r *jobRunRepository) nextStepOrder(runID string) (int, error) {
	var maxOrder sql.NullInt64
	err := r.db.QueryRow(`SELECT MAX(step_order) FROM run_steps WHERE run_id = ?`, runID).Scan(&maxOrder)
	if err != nil {
		return 0, err
	}
	if !maxOrder.Valid {
		return 1, nil
	}
	return int(maxOrder.Int64) + 1, nil
}

func (r *jobRunRepository) ListToolInvocations(runID string) ([]ToolInvocation, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("job run repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT id, run_id, session_id, step_id, tool_name, provider, input_summary, output_summary,
			status, duration_ms, error, created_at
		FROM tool_invocations
		WHERE run_id = ?
		ORDER BY created_at ASC
	`, strings.TrimSpace(runID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ToolInvocation{}
	for rows.Next() {
		var inv ToolInvocation
		if err := rows.Scan(
			&inv.ID, &inv.RunID, &inv.SessionID, &inv.StepID, &inv.ToolName, &inv.Provider,
			&inv.InputSummary, &inv.OutputSummary, &inv.Status, &inv.DurationMs, &inv.Error, &inv.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, inv)
	}
	return out, rows.Err()
}

func (r *jobRunRepository) InsertToolInvocation(inv ToolInvocation) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("job run repository 未初始化")
	}
	if strings.TrimSpace(inv.ID) == "" {
		inv.ID = uuid.New().String()
	}
	now := time.Now().UnixMilli()
	if inv.CreatedAt == 0 {
		inv.CreatedAt = now
	}
	_, err := r.db.Exec(`
		INSERT INTO tool_invocations (
			id, run_id, session_id, step_id, tool_name, provider, input_summary, output_summary,
			status, duration_ms, error, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, inv.ID, inv.RunID, inv.SessionID, inv.StepID, inv.ToolName, inv.Provider,
		inv.InputSummary, inv.OutputSummary, inv.Status, inv.DurationMs, inv.Error, inv.CreatedAt, now)
	return err
}

func (r *jobRunRepository) InsertModelUsage(input ModelUsageInput, status string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("job run repository 未初始化")
	}
	now := time.Now().UnixMilli()
	total := input.TotalTokens
	if total == 0 {
		total = input.InputTokens + input.OutputTokens
	}
	_, err := r.db.Exec(`
		INSERT INTO model_usage (
			id, run_id, session_id, provider, model, input_tokens, output_tokens, total_tokens,
			cost_micros, latency_ms, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, ?, ?, ?, ?)
	`, uuid.New().String(), strings.TrimSpace(input.RunID), strings.TrimSpace(input.SessionID),
		strings.TrimSpace(input.Provider), strings.TrimSpace(input.Model),
		input.InputTokens, input.OutputTokens, total, input.LatencyMs, status, now, now)
	return err
}

func newRunID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw != "" {
		return raw
	}
	return uuid.New().String()
}
