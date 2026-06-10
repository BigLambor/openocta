package employees

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type taskRepository struct {
	db *sql.DB
}

func newTaskRepository(db *sql.DB) *taskRepository {
	if db == nil {
		return nil
	}
	return &taskRepository{db: db}
}

func (r *taskRepository) List() ([]EmployeeTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("task repository 未初始化")
	}
	rows, err := r.db.Query(`SELECT workflow_json FROM tasks ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []EmployeeTask{}
	for rows.Next() {
		var detail string
		if err := rows.Scan(&detail); err != nil {
			return nil, err
		}
		var task EmployeeTask
		if err := json.Unmarshal([]byte(detail), &task); err != nil {
			return nil, err
		}
		NormalizeTask(&task)
		out = append(out, task)
	}
	return out, rows.Err()
}

func (r *taskRepository) Get(id string) (*EmployeeTask, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("task repository 未初始化")
	}
	id = strings.TrimSpace(id)
	var detail string
	err := r.db.QueryRow(`SELECT workflow_json FROM tasks WHERE id = ?`, id).Scan(&detail)
	if err == sql.ErrNoRows {
		return nil, os.ErrNotExist
	}
	if err != nil {
		return nil, err
	}
	var task EmployeeTask
	if err := json.Unmarshal([]byte(detail), &task); err != nil {
		return nil, err
	}
	NormalizeTask(&task)
	return &task, nil
}

func (r *taskRepository) Upsert(t *EmployeeTask) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("task repository 未初始化")
	}
	if t == nil {
		return os.ErrInvalid
	}
	NormalizeTask(t)
	if strings.TrimSpace(t.ID) == "" {
		return os.ErrInvalid
	}
	title, status, triggerType, triggerRef, workflowJSON, evaluationJSON, createdAt, updatedAt := taskRowFields(*t)
	_, err := r.db.Exec(`
		INSERT INTO tasks (
			id, employee_id, domain, capability, title, status, trigger_type, trigger_ref,
			workflow_json, evaluation_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			employee_id = excluded.employee_id,
			domain = excluded.domain,
			capability = excluded.capability,
			title = excluded.title,
			status = excluded.status,
			trigger_type = excluded.trigger_type,
			trigger_ref = excluded.trigger_ref,
			workflow_json = excluded.workflow_json,
			evaluation_json = excluded.evaluation_json,
			updated_at = excluded.updated_at
	`, t.ID, strings.TrimSpace(t.EmployeeID), strings.TrimSpace(t.DomainKey), strings.TrimSpace(t.CapabilityKey),
		title, status, triggerType, triggerRef, workflowJSON, evaluationJSON, createdAt, updatedAt)
	return err
}

func (r *taskRepository) Delete(id string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("task repository 未初始化")
	}
	id = strings.TrimSpace(id)
	if !IsValidTaskID(id) {
		return os.ErrInvalid
	}
	res, err := r.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return os.ErrNotExist
	}
	return nil
}

func (r *taskRepository) ImportJSONDir(dir string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("task repository 未初始化")
	}
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return 0, nil
	}
	fi, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	if !fi.IsDir() {
		return 0, fmt.Errorf("employee tasks path is not a directory: %s", dir)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	imported := 0
	var importedFiles []string
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		taskPath := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(taskPath)
		if err != nil {
			return imported, err
		}
		var task EmployeeTask
		if err := json.Unmarshal(data, &task); err != nil {
			return imported, err
		}
		if strings.TrimSpace(task.ID) == "" {
			continue
		}
		if err := r.Upsert(&task); err != nil {
			return imported, err
		}
		imported++
		importedFiles = append(importedFiles, taskPath)
	}
	if imported == 0 {
		return 0, nil
	}
	if err := backupEmployeeTasksDir(dir, importedFiles); err != nil {
		return imported, err
	}
	return imported, nil
}

func taskRowFields(t EmployeeTask) (title, status, triggerType, triggerRef, workflowJSON, evaluationJSON string, createdAt, updatedAt int64) {
	title = strings.TrimSpace(firstNonEmpty(t.ObjectRef, t.CapabilityKey, t.ID))
	status = strings.TrimSpace(firstNonEmpty(t.ExecutionStatus, t.Status))
	if status == "" {
		status = ExecutionPending
	}
	triggerType = strings.TrimSpace(t.TriggerType)
	if triggerType == "" {
		triggerType = "manual"
	}
	triggerRef = strings.TrimSpace(firstNonEmpty(t.RunID, t.SessionID, t.ObjectRef))
	workflowBytes, err := json.Marshal(t)
	if err != nil {
		workflowJSON = "{}"
	} else {
		workflowJSON = string(workflowBytes)
	}
	evalPayload := map[string]string{"evaluation": strings.TrimSpace(t.Evaluation)}
	if evalPayload["evaluation"] == "" {
		evalPayload["evaluation"] = EvaluationUnrated
	}
	evalBytes, err := json.Marshal(evalPayload)
	if err != nil {
		evaluationJSON = "{}"
	} else {
		evaluationJSON = string(evalBytes)
	}
	createdAt = t.StartedAt
	if createdAt == 0 {
		createdAt = time.Now().UnixMilli()
	}
	updatedAt = t.FinishedAt
	if updatedAt == 0 {
		updatedAt = createdAt
	}
	if updatedAt < createdAt {
		updatedAt = createdAt
	}
	return title, status, triggerType, triggerRef, workflowJSON, evaluationJSON, createdAt, updatedAt
}

func backupEmployeeTasksDir(dir string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	backupRoot := filepath.Join(filepath.Dir(dir), "employee_tasks_backup", fmt.Sprintf("%d", time.Now().UnixMilli()))
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return err
	}
	for _, src := range files {
		dst := filepath.Join(backupRoot, filepath.Base(src))
		if err := os.Rename(src, dst); err != nil {
			return err
		}
	}
	return nil
}
