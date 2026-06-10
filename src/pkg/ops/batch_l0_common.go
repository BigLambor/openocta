package ops

import "strings"

func aggregateBatchDomainSnapshot(domain, objectID, policyVersion string, signals []HealthSignal, observedAt string, ttl int) HealthSnapshot {
	var sum int
	count := 0
	for _, s := range signals {
		if s.Score == nil {
			continue
		}
		sum += *s.Score
		count++
	}
	avg := 0
	if count > 0 {
		avg = sum / count
	}
	return HealthSnapshot{
		SchemaVersion:            "1",
		AggregationPolicyVersion: policyVersion,
		ObjectType:               HealthObjectDomain,
		ObjectID:                 objectID,
		Domain:                   domain,
		Score:                    &avg,
		ScoreStatus:              scoreStatusFromScore(avg),
		Source:                   "collector:batch_l0",
		Coverage:                 1,
		PresentSources:           []string{SignalTypeBCHWorkload},
		ObservedAt:               observedAt,
	}
}

// ListSignalsForRun returns signals for a scenario and optional run ID.
func ListSignalsForRun(runID, scenarioKey, objectType string) ([]HealthSignal, error) {
	runID = strings.TrimSpace(runID)
	scenarioKey = strings.TrimSpace(scenarioKey)
	objectType = strings.TrimSpace(objectType)
	signals, err := ListHealthSignals()
	if err != nil {
		return nil, err
	}
	out := make([]HealthSignal, 0)
	for _, s := range signals {
		if scenarioKey != "" && s.ScenarioKey != scenarioKey {
			continue
		}
		if objectType != "" && s.ObjectType != objectType {
			continue
		}
		if runID != "" && strings.TrimSpace(s.RunID) != runID {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}
