package employees

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/openocta/openocta/pkg/paths"
)

// ResolveEmployeeTasksDir returns the directory path where employee tasks are stored.
func ResolveEmployeeTasksDir(env func(string) string) string {
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "employee_tasks")
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
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, os.ErrNotExist
	}
	dir := ResolveEmployeeTasksDir(env)
	taskPath := filepath.Join(dir, id+".json")
	data, err := os.ReadFile(taskPath)
	if err != nil {
		return nil, os.ErrNotExist
	}
	var task EmployeeTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// SaveTask writes a SRE task record to disk.
func SaveTask(t *EmployeeTask, env func(string) string) error {
	if t == nil {
		return os.ErrInvalid
	}
	id := strings.TrimSpace(t.ID)
	if id == "" {
		return os.ErrInvalid
	}
	dir := ResolveEmployeeTasksDir(env)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	taskPath := filepath.Join(dir, id+".json")
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(taskPath, data, 0644)
}

// DeleteTask removes an SRE task record from disk.
func DeleteTask(id string, env func(string) string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return os.ErrInvalid
	}
	dir := ResolveEmployeeTasksDir(env)
	taskPath := filepath.Join(dir, id+".json")
	return os.Remove(taskPath)
}
