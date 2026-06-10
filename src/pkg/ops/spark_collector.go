package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SparkL0Opts configures a batch Spark health L0 run.
type SparkL0Opts struct {
	RunID       string
	ScenarioKey string
	Domain      string
	ClusterID   string
}

// SparkL0Result summarizes an L0 batch run.
type SparkL0Result struct {
	JobCount      int `json:"jobCount"`
	HealthyCount  int `json:"healthyCount"`
	WarningCount  int `json:"warningCount"`
	CriticalCount int `json:"criticalCount"`
	AnomalyCount  int `json:"anomalyCount"`
}

// RunSparkHealthL0 collects Spark jobs, scores, and writes L3 Facts.
func RunSparkHealthL0(ctx context.Context, opts SparkL0Opts) (SparkL0Result, error) {
	if healthStore == nil {
		return SparkL0Result{}, fmt.Errorf("health store 未初始化")
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = uuid.New().String()
	}
	scenarioKey := strings.TrimSpace(opts.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ScenarioSparkHealth
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = DomainHadoop
	}

	inventory, err := BchServiceProvider().ListSparkJobs()
	if err != nil {
		return SparkL0Result{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	signals := make([]HealthSignal, 0, len(inventory))
	var healthy, warning, critical, anomaly int

	for _, base := range inventory {
		scored := ScoreSparkJob(base)
		status := sparkStatusFromScore(scored.Score)
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
			ID:            "sig-spark-" + scored.ID,
			RunID:         runID,
			ScenarioKey:   scenarioKey,
			ObjectType:    HealthObjectJob,
			ObjectID:      scored.ID,
			ClusterID:     scored.Cluster,
			Domain:        domain,
			Type:          SignalTypeBCHWorkload,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "collector:spark_metrics_batch",
			SourceKind:    SourceKindCollector,
			Evidence: map[string]interface{}{
				"name":       scored.Name,
				"owner":      scored.Owner,
				"cluster":    scored.Cluster,
				"jobStatus":  scored.Status,
				"labels":     scored.Labels,
				"score":      scored.Score,
				"metrics":    scored.Metrics,
				"diagnosis":  scored.Diagnosis,
			},
			ObservedAt: now,
			TTLSec:     ttl,
			Freshness:  FreshnessOK,
		})
	}

	snapshot := aggregateBatchDomainSnapshot(domain, SparkDomainSnapshotID, "spark:l0:v1", signals, now, ttl)
	if err := healthStore.UpsertSignals(signals); err != nil {
		return SparkL0Result{}, err
	}
	if err := healthStore.UpsertSnapshots([]HealthSnapshot{snapshot}); err != nil {
		return SparkL0Result{}, err
	}

	return SparkL0Result{
		JobCount: len(inventory), HealthyCount: healthy, WarningCount: warning,
		CriticalCount: critical, AnomalyCount: anomaly,
	}, nil
}

// ListSparkJobsHealth returns Spark jobs from L3 Facts when available.
func ListSparkJobsHealth() ([]SparkJob, error) {
	if healthStore != nil {
		signals, err := healthStore.ListSignals()
		if err == nil {
			jobs := sparkJobsFromSignals(signals)
			if len(jobs) > 0 {
				return jobs, nil
			}
		}
	}
	return BchServiceProvider().ListSparkJobs()
}

func sparkJobsFromSignals(signals []HealthSignal) []SparkJob {
	out := make([]SparkJob, 0)
	for _, s := range signals {
		if s.ObjectType != HealthObjectJob || s.ScenarioKey != ScenarioSparkHealth {
			continue
		}
		job := SparkJob{ID: s.ObjectID, Cluster: s.ClusterID}
		if s.Score != nil {
			_ = s.Score
		}
		if s.Evidence != nil {
			if v, ok := s.Evidence["name"].(string); ok {
				job.Name = v
			}
			if v, ok := s.Evidence["owner"].(string); ok {
				job.Owner = v
			}
			if v, ok := s.Evidence["jobStatus"].(string); ok {
				job.Status = v
			}
			if raw, ok := s.Evidence["metrics"].(map[string]interface{}); ok {
				job.Metrics = decodeSparkJobMetric(raw)
			}
			if v, ok := s.Evidence["diagnosis"].(string); ok {
				job.TuningAdvice = v
			}
		}
		out = append(out, job)
	}
	return out
}

func decodeSparkJobMetric(raw map[string]interface{}) SparkJobMetric {
	var m SparkJobMetric
	if v, ok := raw["failedTasks"].(float64); ok {
		m.FailedTasks = int(v)
	}
	if v, ok := raw["cpuSkewRatio"].(float64); ok {
		m.CpuSkewRatio = v
	}
	if v, ok := raw["memorySkewRatio"].(float64); ok {
		m.MemorySkewRatio = v
	}
	if v, ok := raw["maxTaskDurationSec"].(float64); ok {
		m.MaxTaskDurationSec = int(v)
	}
	if v, ok := raw["avgTaskDurationSec"].(float64); ok {
		m.AvgTaskDurationSec = int(v)
	}
	return m
}
