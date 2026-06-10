package ops

import "strings"

// YarnScoredQueue holds rule-based YARN queue scoring output.
type YarnScoredQueue struct {
	ID        string
	Name      string
	Cluster   string
	Status    string
	RiskLevel string
	Score     int
	Metrics   YarnQueueMetric
	Reasons   []string
	Advice    string
	Action    string
}

// ScoreYarnQueue applies rule-based scoring for YARN queue L0.
func ScoreYarnQueue(q YarnQueueEvaluation) YarnScoredQueue {
	score := 100
	status := strings.TrimSpace(q.Status)
	risk := strings.TrimSpace(q.RiskLevel)

	switch status {
	case "under_allocated":
		score = 45
		if q.PendingContainers > 50 {
			score = 30
		}
	case "over_allocated":
		score = 72
	case "idle":
		score = 68
	case "healthy":
		score = 92
	}
	if risk == "high" {
		score -= 20
	} else if risk == "medium" {
		score -= 8
	}
	if q.WaitingApps > 0 {
		score -= 10
	}
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return YarnScoredQueue{
		ID: q.ID, Name: q.Name, Cluster: q.Cluster,
		Status: status, RiskLevel: risk, Score: score,
		Metrics: q.Metrics, Reasons: q.Reasons, Advice: q.Advice, Action: q.Action,
	}
}

func yarnStatusFromScore(score int) string {
	switch {
	case score >= 85:
		return HealthStatusHealthy
	case score >= 70:
		return HealthStatusWarning
	default:
		return HealthStatusCritical
	}
}
