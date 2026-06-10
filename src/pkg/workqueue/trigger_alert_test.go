package workqueue

import (
	"testing"

	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

func TestAlertTriggerEnvelope(t *testing.T) {
	group := ops.AlertGroup{
		ID:         "alert-group-abc",
		RunID:      "run-alert-1",
		Domain:     "hadoop",
		ClusterID:  "prod-a",
		Component:  "yarn",
		SessionKey: "agent:main:alert:xyz",
	}
	env := AlertTriggerEnvelope(group, "emp_bch_duty", group.SessionKey, "analyze alerts")
	if env.TriggerType != jobrun.TriggerAlert {
		t.Fatalf("triggerType = %q", env.TriggerType)
	}
	if env.Priority != PriorityHigh {
		t.Fatalf("priority = %d, want %d", env.Priority, PriorityHigh)
	}
	if env.ScenarioKey != "ops-diagnosis" {
		t.Fatalf("scenarioKey = %q", env.ScenarioKey)
	}
	if env.IdempotencyKey != "alert:alert-group-abc" {
		t.Fatalf("idempotencyKey = %q", env.IdempotencyKey)
	}
}
