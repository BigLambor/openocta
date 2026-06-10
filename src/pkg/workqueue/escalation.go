package workqueue

import (
	"strings"

	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
)

func (s *Service) enqueueBatchEscalations(plan storedPlan, env TriggerEnvelope) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if !ops.IsBatchL0Scenario(env.ScenarioKey) {
		return nil
	}
	if env.TriggerType == jobrun.TriggerAlert {
		return nil
	}

	signals, err := ops.ListSignalsForRun(plan.ParentRunID, env.ScenarioKey, "")
	if err != nil || len(signals) == 0 {
		return err
	}

	max := s.cfg.MaxL2PerParentRun
	if max <= 0 {
		max = defaultMaxL2PerParentRun
	}
	cooldown := s.cfg.DefaultL2CooldownMs
	now := nowMs()
	enqueued := 0

	for _, sig := range signals {
		if enqueued >= max {
			break
		}
		if !ops.ShouldEscalateSignal(sig) {
			continue
		}
		objectID := strings.TrimSpace(sig.ObjectID)
		if objectID == "" {
			continue
		}
		if cooldown > 0 {
			if last, ok := s.repo.lastSuccessfulL2At(env.ScenarioKey, sig.ObjectType, objectID, jobrun.TriggerAlert); ok {
				if now-last < cooldown {
					continue
				}
			}
		}

		task := WorkTask{
			ID:             newTaskID(),
			PlanID:         plan.ID,
			TenantID:       "default",
			Tier:           TierL2,
			Action:         ActionAIDiagnose,
			ObjectType:     sig.ObjectType,
			ObjectID:       objectID,
			ParentRunID:    plan.ParentRunID,
			ChildRunID:     newTaskID(),
			Status:         TaskStatusQueued,
			Priority:       plan.Priority,
			IdempotencyKey: plan.IdempotencyKey + ":L2:" + objectID,
			CreatedAt:      now,
			UpdatedAt:      now,
			Input: map[string]interface{}{
				"scenarioKey": env.ScenarioKey,
				"action":      ActionAIDiagnose,
				"message":     ops.BuildEscalationMessage(sig),
				"escalation":  true,
			},
		}
		if err := s.repo.insertTask(task); err != nil {
			return err
		}
		enqueued++
	}
	return nil
}

// enqueueFlinkEscalations is kept for backward-compatible tests.
func (s *Service) enqueueFlinkEscalations(plan storedPlan, env TriggerEnvelope) error {
	return s.enqueueBatchEscalations(plan, env)
}
