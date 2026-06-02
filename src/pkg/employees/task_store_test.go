package employees

import (
	"os"
	"testing"
)

func TestTaskStore(t *testing.T) {
	// Setup temporary state dir
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

	// 1. Test Save & Load
	task1 := &EmployeeTask{
		ID:            "test-task-1",
		EmployeeID:    "emp-sre-1",
		DomainKey:     "hadoop",
		CapabilityKey: "health-inspection",
		Status:        "success",
		Input:         "check cluster health",
		Output:        "cluster is healthy",
		Conclusion:    "healthy",
		StartedAt:     1000,
		FinishedAt:    2000,
		Evaluation:    "unrated",
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

	// 2. Test ListTasks
	task2 := &EmployeeTask{
		ID:            "test-task-2",
		EmployeeID:    "emp-sre-1",
		DomainKey:     "fi",
		CapabilityKey: "diagnosis-incident",
		Status:        "failed",
		StartedAt:     3000, // newer than task1
		FinishedAt:    4000,
		Evaluation:    "unrated",
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

	// ListTasks should sort by StartedAt descending (newest first)
	if tasks[0].ID != "test-task-2" || tasks[1].ID != "test-task-1" {
		t.Errorf("ListTasks did not sort correctly by StartedAt descending")
	}

	// 3. Test DeleteTask
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
