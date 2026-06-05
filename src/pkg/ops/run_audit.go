package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/paths"
)

// RunAudit logs the execution details of an L2 operation (Scenario runner, inspection, or chat).
type RunAudit struct {
	RunID          string   `json:"runId"`
	ScenarioKey    string   `json:"scenarioKey"`
	EmployeeID     string   `json:"employeeId,omitempty"`
	ObjectID       string   `json:"objectId"`
	ToolsCalled    []string `json:"toolsCalled,omitempty"`
	MCPCalled      []string `json:"mcpCalled,omitempty"`
	SignalsWritten int      `json:"signalsWritten"`
	MissingSources []string `json:"missingSources,omitempty"`
	DurationMs     int64    `json:"durationMs"`
	Operator       string   `json:"operator"` // "system", "user", "cron", "alert"
	Timestamp      string   `json:"timestamp"`
}

var (
	auditMu sync.Mutex
)

// RecordRunAudit appends a run audit record to the local JSONL store.
func RecordRunAudit(audit RunAudit) error {
	auditMu.Lock()
	defer auditMu.Unlock()

	stateDir := paths.ResolveStateDir(os.Getenv)
	auditDir := filepath.Join(stateDir, "ops")
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		return err
	}

	auditPath := filepath.Join(auditDir, "run_audit.jsonl")
	f, err := os.OpenFile(auditPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if audit.Timestamp == "" {
		audit.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(audit)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
}
