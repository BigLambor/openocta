package workqueue

import (
	"fmt"
	"strings"

	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

// AlertTriggerEnvelope builds a high-priority work queue envelope for alert diagnosis.
func AlertTriggerEnvelope(group ops.AlertGroup, employeeID, sessionKey, message string) TriggerEnvelope {
	scenarioKey := "ops-diagnosis"
	parentRunID := strings.TrimSpace(group.RunID)
	if parentRunID == "" {
		parentRunID = strings.TrimSpace(group.ID)
	}
	return TriggerEnvelope{
		TriggerType:    jobrun.TriggerAlert,
		TriggerRef:     ops.AlertDiagnosisJobID,
		ScenarioKey:    scenarioKey,
		Priority:       PriorityHigh,
		IdempotencyKey: fmt.Sprintf("alert:%s", strings.TrimSpace(group.ID)),
		ParentRunID:    parentRunID,
		Message:        strings.TrimSpace(message),
		Domain:         strings.TrimSpace(group.Domain),
		ClusterID:      strings.TrimSpace(group.ClusterID),
		Component:      strings.TrimSpace(group.Component),
		Scope: TriggerScope{
			ObjectType: "alert_group",
			ObjectIDs:  []string{strings.TrimSpace(group.ID)},
			ClusterID:  strings.TrimSpace(group.ClusterID),
			Domain:     strings.TrimSpace(group.Domain),
			Component:  strings.TrimSpace(group.Component),
		},
		CronJob: CronJobSnapshot{
			ID:                ops.AlertDiagnosisJobID,
			DigitalEmployeeID: strings.TrimSpace(employeeID),
			SessionTarget:     "isolated",
			SessionKey:        strings.TrimSpace(sessionKey),
			PayloadKind:       "agentTurn",
			PayloadMessage:    strings.TrimSpace(message),
		},
	}
}
