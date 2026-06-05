package handlers

import (
	"os"
	"testing"

	"github.com/openocta/openocta/pkg/employees"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

func TestEmployeeTasksCreateHandlerPersistsBchAlertDecision(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tmp)

	var (
		ok      bool
		payload interface{}
		errResp *protocol.ErrorShape
	)
	err := EmployeeTasksCreateHandler(HandlerOpts{
		Params: map[string]interface{}{
			"employeeId":      "builtin-bch-oncall",
			"domainKey":       "hadoop",
			"capabilityKey":   "observability-alert",
			"scenarioKey":     "bch-alert-root-cause",
			"objectRef":       "alert-group-001",
			"triggerType":     "alert",
			"executionStatus": "succeeded",
			"workflowStatus":  "closed",
			"input":           "告警组: HDFS NameNode RPC 延迟升高",
			"output":          "根因候选: JournalNode 写入延迟",
			"conclusion":      "根因分析建议已确认：按 Runbook 完成处置",
			"operator":        "sre-admin",
			"evaluation":      "accepted",
			"artifacts":       []interface{}{"ops-alert-group:alert-group-001", "workbench-mode:root-cause"},
			"metrics": map[string]interface{}{
				"rawAlertCount":     float64(12),
				"reducedAlertCount": float64(1),
				"savedHours":        float64(0.88),
			},
		},
		Respond: func(nextOk bool, nextPayload interface{}, nextErr *protocol.ErrorShape, _ map[string]interface{}) {
			ok = nextOk
			payload = nextPayload
			errResp = nextErr
		},
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if !ok || errResp != nil {
		t.Fatalf("unexpected response ok=%v payload=%v err=%v", ok, payload, errResp)
	}
	res, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected payload type: %T", payload)
	}
	id, _ := res["id"].(string)
	if id == "" {
		t.Fatalf("expected generated task id, got payload=%v", payload)
	}

	task, err := employees.LoadTask(id, os.Getenv)
	if err != nil {
		t.Fatalf("load task: %v", err)
	}
	if task.EmployeeID != "builtin-bch-oncall" {
		t.Fatalf("unexpected employee id: %s", task.EmployeeID)
	}
	if task.DomainKey != employees.DomainHadoop {
		t.Fatalf("unexpected domain: %s", task.DomainKey)
	}
	if task.CapabilityKey != employees.CapabilityObservabilityAlert {
		t.Fatalf("unexpected capability: %s", task.CapabilityKey)
	}
	if task.WorkflowStatus != employees.WorkflowClosed || task.Evaluation != employees.EvaluationAccepted {
		t.Fatalf("unexpected workflow/evaluation: %s/%s", task.WorkflowStatus, task.Evaluation)
	}
	if task.Metrics.RawAlertCount != 12 || task.Metrics.ReducedAlertCount != 1 {
		t.Fatalf("unexpected alert metrics: %+v", task.Metrics)
	}
}
