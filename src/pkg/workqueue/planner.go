package workqueue

import (
	"strings"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/ops"
)

// BuildPlan creates a WorkPlan from a trigger envelope.
func BuildPlan(env TriggerEnvelope) (WorkPlan, error) {
	scenarioKey := strings.TrimSpace(env.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ops.ScenarioKeyForInspection(ops.InspectionReport{
			JobID:  env.TriggerRef,
			Domain: firstNonEmpty(env.Domain, env.Scope.Domain),
		})
	}
	if ops.IsBatchL0Scenario(scenarioKey) {
		parentRunID := strings.TrimSpace(env.ParentRunID)
		if parentRunID == "" {
			parentRunID = uuid.New().String()
		}
		return WorkPlan{
			ID:             uuid.New().String(),
			ParentRunID:    parentRunID,
			TriggerType:    env.TriggerType,
			TriggerRef:     env.TriggerRef,
			ScenarioKey:    scenarioKey,
			Status:         PlanStatusQueued,
			Priority:       env.Priority,
			IdempotencyKey: env.IdempotencyKey,
			ScheduledAtMs:  env.ScheduledAtMs,
			Steps:          []PlanStep{{Tier: TierL0, Action: ActionCollectAndScore}},
			Envelope:       env,
		}, nil
	}

	_, hasNative := ops.GetOpsScenario(scenarioKey)

	steps := []PlanStep{}
	switch strings.TrimSpace(env.TriggerType) {
	case "alert":
		steps = append(steps, PlanStep{Tier: TierL2, Action: ActionAIDiagnose})
	default:
		if hasNative {
			steps = append(steps, PlanStep{Tier: TierL1, Action: ActionScenarioInspect})
		}
		if env.CronJob.SessionTarget == "isolated" && env.CronJob.PayloadKind == "agentTurn" {
			steps = append(steps, PlanStep{Tier: TierL2, Action: ActionAIDiagnose, MaxConcurrency: 1})
		}
		if len(steps) == 0 {
			steps = append(steps, PlanStep{Tier: TierL2, Action: ActionAIDiagnose, MaxConcurrency: 1})
		}
	}

	parentRunID := strings.TrimSpace(env.ParentRunID)
	if parentRunID == "" {
		parentRunID = uuid.New().String()
	}

	return WorkPlan{
		ID:             uuid.New().String(),
		ParentRunID:    parentRunID,
		TriggerType:    env.TriggerType,
		TriggerRef:     env.TriggerRef,
		ScenarioKey:    scenarioKey,
		Status:         PlanStatusQueued,
		Priority:       env.Priority,
		IdempotencyKey: env.IdempotencyKey,
		ScheduledAtMs:  env.ScheduledAtMs,
		Steps:          steps,
		Envelope:       env,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
