package employees

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/paths"
)

var taskIDPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,127}$`)

// ResolveEmployeeTasksDir returns the directory path where employee tasks are stored.
func ResolveEmployeeTasksDir(env func(string) string) string {
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "employee_tasks")
}

func NewTaskID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "task-" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000"), ".", "")
	}
	return "task-" + time.Now().UTC().Format("20060102150405") + "-" + hex.EncodeToString(b[:])
}

func IsValidTaskID(id string) bool {
	return taskIDPattern.MatchString(strings.TrimSpace(id))
}

func taskPathForID(id string, env func(string) string) (string, error) {
	id = strings.TrimSpace(id)
	if !IsValidTaskID(id) {
		return "", os.ErrInvalid
	}
	dir := ResolveEmployeeTasksDir(env)
	taskPath := filepath.Join(dir, id+".json")
	cleanDir := filepath.Clean(dir)
	cleanPath := filepath.Clean(taskPath)
	if filepath.Dir(cleanPath) != cleanDir {
		return "", os.ErrInvalid
	}
	return cleanPath, nil
}

func NormalizeTask(t *EmployeeTask) {
	if t == nil {
		return
	}
	t.ID = strings.TrimSpace(t.ID)
	t.SessionID = strings.TrimSpace(t.SessionID)
	t.RunID = strings.TrimSpace(t.RunID)
	t.EmployeeID = strings.TrimSpace(t.EmployeeID)
	t.DomainKey = NormalizeDomainKey(t.DomainKey)
	t.CapabilityKey = NormalizeCapabilityKey(t.CapabilityKey)
	t.ExecutionStatus = NormalizeExecutionStatus(firstNonEmpty(t.ExecutionStatus, t.Status))
	t.WorkflowStatus = NormalizeWorkflowStatus(t.WorkflowStatus)
	t.Status = LegacyStatusFromExecution(t.ExecutionStatus)
	t.TriggerType = strings.TrimSpace(t.TriggerType)
	if t.TriggerType == "" {
		t.TriggerType = "manual"
	}
	t.Evaluation = strings.ToLower(strings.TrimSpace(t.Evaluation))
	if t.Evaluation == "" {
		t.Evaluation = EvaluationUnrated
	}
	if t.Evaluation == EvaluationAccepted {
		t.WorkflowStatus = WorkflowClosed
	} else if t.Evaluation == EvaluationRejected {
		t.WorkflowStatus = WorkflowRejected
	} else if t.WorkflowStatus == WorkflowOpen && t.ExecutionStatus == ExecutionSucceeded {
		t.WorkflowStatus = WorkflowWaitingApproval
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ListTasks returns all saved SRE task records, sorted by startedAt descending.
func ListTasks(env func(string) string) ([]EmployeeTask, error) {
	var out []EmployeeTask
	dir := ResolveEmployeeTasksDir(env)

	if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
					continue
				}
				taskPath := filepath.Join(dir, e.Name())
				data, err := os.ReadFile(taskPath)
				if err != nil {
					continue
				}
				var task EmployeeTask
				if err := json.Unmarshal(data, &task); err != nil {
					continue
				}
				NormalizeTask(&task)
				out = append(out, task)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt > out[j].StartedAt
	})
	return out, nil
}

// LoadTask loads a specific SRE task record by ID.
func LoadTask(id string, env func(string) string) (*EmployeeTask, error) {
	taskPath, err := taskPathForID(id, env)
	if err != nil {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, os.ErrNotExist
	}
	var task EmployeeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	NormalizeTask(&task)
	return &task, nil
}

// SaveTask writes a SRE task record to disk.
func SaveTask(t *EmployeeTask, env func(string) string) error {
	if t == nil {
		return os.ErrInvalid
	}
	id := strings.TrimSpace(t.ID)
	if id == "" {
		id = NewTaskID()
		t.ID = id
	}
	taskPath, err := taskPathForID(id, env)
	if err != nil {
		return err
	}
	NormalizeTask(t)
	dir := filepath.Dir(taskPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(taskPath, data, 0644)
}

// DeleteTask removes an SRE task record from disk.
func DeleteTask(id string, env func(string) string) error {
	taskPath, err := taskPathForID(id, env)
	if err != nil {
		return err
	}
	return os.Remove(taskPath)
}
