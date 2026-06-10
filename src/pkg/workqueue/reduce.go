package workqueue

import (
	"encoding/json"
	"strings"

	"github.com/openocta/openocta/pkg/ops"
)

func (s *Service) maybeEnqueueDomainReduce(plan storedPlan, env TriggerEnvelope) error {
	if s == nil || s.repo == nil || !s.cfg.DomainReduceEnabled {
		return nil
	}
	if !ops.IsBatchL0Scenario(env.ScenarioKey) {
		return nil
	}

	tasks, err := s.repo.listTasksByPlan(plan.ID)
	if err != nil {
		return err
	}

	hasReduce := false
	l2EscalationTotal := 0
	l2EscalationDone := 0

	for _, t := range tasks {
		if t.Action == ActionDomainReduce {
			hasReduce = true
			continue
		}
		if t.Status == TaskStatusQueued || t.Status == TaskStatusRunning {
			return nil
		}
		if t.Tier != TierL2 || t.Action != ActionAIDiagnose {
			continue
		}
		esc, _ := t.Input["escalation"].(bool)
		if !esc {
			continue
		}
		l2EscalationTotal++
		switch t.Status {
		case TaskStatusSucceeded, TaskStatusFailed, TaskStatusTimeout, TaskStatusCancelled:
			l2EscalationDone++
		}
	}

	if l2EscalationTotal == 0 || l2EscalationDone < l2EscalationTotal || hasReduce {
		return nil
	}

	taskInputs := domainReduceTaskInputs(tasks)
	inputsJSON, _ := json.Marshal(taskInputs)

	now := nowMs()
	task := WorkTask{
		ID:             newTaskID(),
		PlanID:         plan.ID,
		TenantID:       "default",
		Tier:           TierL2,
		Action:         ActionDomainReduce,
		ObjectType:     ops.HealthObjectDomain,
		ObjectID:       ops.DomainSnapshotIDForScenario(env.ScenarioKey),
		ParentRunID:    plan.ParentRunID,
		ChildRunID:     newTaskID(),
		Status:         TaskStatusQueued,
		Priority:       plan.Priority,
		IdempotencyKey: plan.IdempotencyKey + ":reduce",
		CreatedAt:      now,
		UpdatedAt:      now,
		Input: map[string]interface{}{
			"scenarioKey":        env.ScenarioKey,
			"action":             ActionDomainReduce,
			"escalationTasksJson": string(inputsJSON),
		},
	}
	return s.repo.insertTask(task)
}

func domainReduceTaskInputs(tasks []WorkTask) []ops.DomainReduceTaskInput {
	out := make([]ops.DomainReduceTaskInput, 0)
	for _, t := range tasks {
		if t.Tier != TierL2 || t.Action != ActionAIDiagnose {
			continue
		}
		esc, _ := t.Input["escalation"].(bool)
		if !esc {
			continue
		}
		out = append(out, ops.DomainReduceTaskInput{
			ObjectID:   strings.TrimSpace(t.ObjectID),
			ObjectType: strings.TrimSpace(t.ObjectType),
			Status:     t.Status,
			Error:      t.Error,
			ChildRunID: strings.TrimSpace(t.ChildRunID),
		})
	}
	return out
}
