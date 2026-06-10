package employees

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func initTestTaskStore(t *testing.T, tempDir string) {
	t.Helper()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}
	if err := InitTaskStore(tempDir); err != nil {
		t.Fatalf("InitTaskStore: %v", err)
	}
}

func TestTaskStore(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "openocta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	initTestTaskStore(t, tempDir)

	mockEnv := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" || key == "HOME" {
			return tempDir
		}
		return ""
	}

	task1 := &EmployeeTask{
		ID:              "test-task-1",
		EmployeeID:      "emp-sre-1",
		DomainKey:       "hadoop",
		CapabilityKey:   "health-inspection",
		ExecutionStatus: ExecutionSucceeded,
		WorkflowStatus:  WorkflowWaitingApproval,
		Input:           "check cluster health",
		Output:          "cluster is healthy",
		Conclusion:      "healthy",
		StartedAt:       1000,
		FinishedAt:      2000,
		Evaluation:      "unrated",
	}

	err = SaveTask(task1, mockEnv)
	if err != nil {
		t.Errorf("SaveTask failed: %v", err)
	}

	loaded, err := LoadTask("test-task-1", mockEnv)
	if err != nil {
		t.Errorf("LoadTask failed: %v", err)
	}

	if loaded.EmployeeID != task1.EmployeeID || loaded.Status != task1.Status {
		t.Errorf("Loaded task fields do not match saved ones: %+v vs %+v", loaded, task1)
	}
	if loaded.ExecutionStatus != ExecutionSucceeded || loaded.Status != "success" {
		t.Errorf("Expected normalized execution status, got %+v", loaded)
	}

	task2 := &EmployeeTask{
		ID:              "test-task-2",
		EmployeeID:      "emp-sre-1",
		DomainKey:       "fi",
		CapabilityKey:   "diagnosis-incident",
		ExecutionStatus: ExecutionFailed,
		WorkflowStatus:  WorkflowOpen,
		StartedAt:       3000,
		FinishedAt:      4000,
		Evaluation:      "unrated",
	}

	err = SaveTask(task2, mockEnv)
	if err != nil {
		t.Errorf("SaveTask 2 failed: %v", err)
	}

	tasks, err := ListTasks(mockEnv)
	if err != nil {
		t.Errorf("ListTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(tasks))
	}

	if tasks[0].ID != "test-task-2" || tasks[1].ID != "test-task-1" {
		t.Errorf("ListTasks did not sort correctly by StartedAt descending")
	}

	err = DeleteTask("test-task-1", mockEnv)
	if err != nil {
		t.Errorf("DeleteTask failed: %v", err)
	}

	_, err = LoadTask("test-task-1", mockEnv)
	if err == nil {
		t.Errorf("Expected test-task-1 to be deleted, but it was loaded successfully")
	}

	tasks, err = ListTasks(mockEnv)
	if err != nil {
		t.Errorf("ListTasks failed after delete: %v", err)
	}

	if len(tasks) != 1 || tasks[0].ID != "test-task-2" {
		t.Errorf("Expected 1 task (test-task-2) remaining, got %d tasks", len(tasks))
	}
}

func TestTaskStoreRejectsUnsafeIDs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "openocta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	initTestTaskStore(t, tempDir)

	mockEnv := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" || key == "HOME" {
			return tempDir
		}
		return ""
	}

	unsafeIDs := []string{"../escape", "..", "a/b", `a\b`}
	for _, id := range unsafeIDs {
		if err := SaveTask(&EmployeeTask{ID: id, EmployeeID: "emp"}, mockEnv); err == nil {
			t.Fatalf("SaveTask accepted unsafe id %q", id)
		}
		if _, err := LoadTask(id, mockEnv); err == nil {
			t.Fatalf("LoadTask accepted unsafe id %q", id)
		}
		if err := DeleteTask(id, mockEnv); err == nil {
			t.Fatalf("DeleteTask accepted unsafe id %q", id)
		}
	}
}

func TestSaveTaskGeneratesID(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "openocta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	initTestTaskStore(t, tempDir)

	mockEnv := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" || key == "HOME" {
			return tempDir
		}
		return ""
	}

	task := &EmployeeTask{EmployeeID: "emp", ExecutionStatus: ExecutionSucceeded}
	if err := SaveTask(task, mockEnv); err != nil {
		t.Fatalf("SaveTask generated id failed: %v", err)
	}
	if task.ID == "" || !IsValidTaskID(task.ID) {
		t.Fatalf("SaveTask generated invalid id: %q", task.ID)
	}
	if _, err := LoadTask(task.ID, mockEnv); err != nil {
		t.Fatalf("generated task not loadable: %v", err)
	}
}

func TestTaskStoreImportsLegacyJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "openocta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	mockEnv := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" || key == "HOME" {
			return tempDir
		}
		return ""
	}

	dir := ResolveEmployeeTasksDir(mockEnv)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir employee_tasks: %v", err)
	}
	legacy := &EmployeeTask{
		ID:              "legacy-task-1",
		EmployeeID:      "emp-legacy",
		DomainKey:       "hadoop",
		CapabilityKey:   "health-inspection",
		ExecutionStatus: ExecutionSucceeded,
		StartedAt:       5000,
		FinishedAt:      6000,
		Evaluation:      EvaluationUnrated,
	}
	if err := saveTaskToJSON(legacy, mockEnv); err != nil {
		t.Fatalf("seed legacy json: %v", err)
	}

	initTestTaskStore(t, tempDir)

	if _, err := os.Stat(filepath.Join(dir, "legacy-task-1.json")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy json to be moved after import, stat err=%v", err)
	}

	loaded, err := LoadTask("legacy-task-1", mockEnv)
	if err != nil {
		t.Fatalf("LoadTask after import: %v", err)
	}
	if loaded.EmployeeID != "emp-legacy" {
		t.Fatalf("unexpected imported employee id: %s", loaded.EmployeeID)
	}

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("re-init db: %v", err)
	}
	if err := InitTaskStore(tempDir); err != nil {
		t.Fatalf("re-init task store: %v", err)
	}
	tasks, err := ListTasks(mockEnv)
	if err != nil {
		t.Fatalf("ListTasks after re-init: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != "legacy-task-1" {
		t.Fatalf("expected idempotent import to keep one task, got %+v", tasks)
	}
}

func TestTaskStorePersistsAcrossRestart(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "openocta-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	initTestTaskStore(t, tempDir)

	mockEnv := func(key string) string {
		if key == "OPENOCTA_STATE_DIR" || key == "HOME" {
			return tempDir
		}
		return ""
	}

	task := &EmployeeTask{
		ID:              "persist-task-1",
		EmployeeID:      "emp-persist",
		DomainKey:       "fi",
		CapabilityKey:   "diagnosis-incident",
		ExecutionStatus: ExecutionSucceeded,
		StartedAt:       9000,
	}
	if err := SaveTask(task, mockEnv); err != nil {
		t.Fatalf("SaveTask: %v", err)
	}
	db.CloseDB()

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("restart db: %v", err)
	}
	if err := InitTaskStore(tempDir); err != nil {
		t.Fatalf("restart task store: %v", err)
	}

	loaded, err := LoadTask("persist-task-1", mockEnv)
	if err != nil {
		t.Fatalf("LoadTask after restart: %v", err)
	}
	if loaded.EmployeeID != "emp-persist" {
		t.Fatalf("unexpected employee after restart: %s", loaded.EmployeeID)
	}
}
