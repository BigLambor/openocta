package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// YarnL0Opts configures a batch YARN queue L0 run.
type YarnL0Opts struct {
	RunID       string
	ScenarioKey string
	Domain      string
	ClusterID   string
}

// YarnL0Result summarizes an L0 batch run.
type YarnL0Result struct {
	QueueCount    int `json:"queueCount"`
	HealthyCount  int `json:"healthyCount"`
	WarningCount  int `json:"warningCount"`
	CriticalCount int `json:"criticalCount"`
	AnomalyCount  int `json:"anomalyCount"`
}

// RunYarnHealthL0 collects YARN queues, scores, and writes L3 Facts.
func RunYarnHealthL0(ctx context.Context, opts YarnL0Opts) (YarnL0Result, error) {
	if healthStore == nil {
		return YarnL0Result{}, fmt.Errorf("health store 未初始化")
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = uuid.New().String()
	}
	scenarioKey := strings.TrimSpace(opts.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ScenarioYarnHealth
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = DomainHadoop
	}

	inventory, err := BchServiceProvider().ListYarnQueues()
	if err != nil {
		return YarnL0Result{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	signals := make([]HealthSignal, 0, len(inventory))
	var healthy, warning, critical, anomaly int

	for _, base := range inventory {
		scored := ScoreYarnQueue(base)
		status := yarnStatusFromScore(scored.Score)
		score := scored.Score
		switch status {
		case HealthStatusHealthy:
			healthy++
		case HealthStatusWarning:
			warning++
			anomaly++
		default:
			critical++
			anomaly++
		}
		signals = append(signals, HealthSignal{
			SchemaVersion: "1",
			ID:            "sig-yarn-" + strings.ReplaceAll(scored.ID, ".", "_"),
			RunID:         runID,
			ScenarioKey:   scenarioKey,
			ObjectType:    HealthObjectQueue,
			ObjectID:      scored.ID,
			ClusterID:     scored.Cluster,
			Domain:        domain,
			Type:          SignalTypeBCHWorkload,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "collector:yarn_queue_batch",
			SourceKind:    SourceKindCollector,
			Evidence: map[string]interface{}{
				"name":       scored.Name,
				"cluster":    scored.Cluster,
				"queueStatus": scored.Status,
				"riskLevel":  scored.RiskLevel,
				"metrics":    scored.Metrics,
				"reasons":    scored.Reasons,
				"advice":     scored.Advice,
				"action":     scored.Action,
				"score":      scored.Score,
			},
			ObservedAt: now,
			TTLSec:     ttl,
			Freshness:  FreshnessOK,
		})
	}

	snapshot := aggregateBatchDomainSnapshot(domain, YarnDomainSnapshotID, "yarn:l0:v1", signals, now, ttl)
	if err := healthStore.UpsertSignals(signals); err != nil {
		return YarnL0Result{}, err
	}
	if err := healthStore.UpsertSnapshots([]HealthSnapshot{snapshot}); err != nil {
		return YarnL0Result{}, err
	}

	return YarnL0Result{
		QueueCount: len(inventory), HealthyCount: healthy, WarningCount: warning,
		CriticalCount: critical, AnomalyCount: anomaly,
	}, nil
}

// ListYarnQueuesHealth returns YARN queues from L3 Facts when available.
func ListYarnQueuesHealth() ([]YarnQueueEvaluation, error) {
	if healthStore != nil {
		signals, err := healthStore.ListSignals()
		if err == nil {
			queues := yarnQueuesFromSignals(signals)
			if len(queues) > 0 {
				return queues, nil
			}
		}
	}
	return BchServiceProvider().ListYarnQueues()
}

func yarnQueuesFromSignals(signals []HealthSignal) []YarnQueueEvaluation {
	out := make([]YarnQueueEvaluation, 0)
	for _, s := range signals {
		if s.ObjectType != HealthObjectQueue || s.ScenarioKey != ScenarioYarnHealth {
			continue
		}
		q := YarnQueueEvaluation{ID: s.ObjectID, Cluster: s.ClusterID}
		if s.Evidence != nil {
			if v, ok := s.Evidence["name"].(string); ok {
				q.Name = v
			}
			if v, ok := s.Evidence["queueStatus"].(string); ok {
				q.Status = v
			}
			if v, ok := s.Evidence["riskLevel"].(string); ok {
				q.RiskLevel = v
			}
			if v, ok := s.Evidence["advice"].(string); ok {
				q.Advice = v
			}
			if v, ok := s.Evidence["action"].(string); ok {
				q.Action = v
			}
			if raw, ok := s.Evidence["metrics"].(map[string]interface{}); ok {
				q.Metrics = decodeYarnQueueMetric(raw)
			}
		}
		out = append(out, q)
	}
	return out
}

func decodeYarnQueueMetric(raw map[string]interface{}) YarnQueueMetric {
	var m YarnQueueMetric
	if v, ok := raw["pendingContainers"].(float64); ok {
		m.ActiveApps = int(v)
	}
	if v, ok := raw["avgCpuPercent"].(float64); ok {
		m.AvgCpuPercent = v
	}
	if v, ok := raw["maxCpuPercent"].(float64); ok {
		m.MaxCpuPercent = v
	}
	return m
}
