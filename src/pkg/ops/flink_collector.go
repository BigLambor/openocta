package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	HealthObjectJob   = "job"
	HealthObjectDomain = "domain"
	ScenarioFlinkHealth = "ops-flink-health"
	FlinkDomainSnapshotID = "hadoop:flink"
)

// BchServiceProvider supplies the BCH service for Flink inventory.
var BchServiceProvider = func() BchService { return NewMockBchService() }

// FlinkL0Opts configures a batch Flink health L0 run.
type FlinkL0Opts struct {
	RunID       string
	ScenarioKey string
	Domain      string
	ClusterID   string
}

// FlinkL0Result summarizes an L0 batch run.
type FlinkL0Result struct {
	JobCount      int    `json:"jobCount"`
	HealthyCount  int    `json:"healthyCount"`
	WarningCount  int    `json:"warningCount"`
	CriticalCount int    `json:"criticalCount"`
	AnomalyCount  int    `json:"anomalyCount"`
	MetricsSource string `json:"metricsSource"`
}

// RunFlinkHealthL0 collects Flink job metrics, scores, and writes L3 Facts.
func RunFlinkHealthL0(ctx context.Context, opts FlinkL0Opts) (FlinkL0Result, error) {
	if healthStore == nil {
		return FlinkL0Result{}, fmt.Errorf("health store 未初始化")
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = uuid.New().String()
	}
	scenarioKey := strings.TrimSpace(opts.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ScenarioFlinkHealth
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = DomainHadoop
	}

	inventory, err := BchServiceProvider().ListFlinkJobs()
	if err != nil {
		return FlinkL0Result{}, err
	}

	metricsSource := "bch"
	vmMetrics, vmErr := collectFlinkMetricsFromVM(ctx, opts.ClusterID)
	if vmErr == nil && len(vmMetrics.byJob) > 0 {
		metricsSource = "vm"
	}

	jobs := make([]FlinkJob, 0, len(inventory))
	for _, base := range inventory {
		in := flinkJobToMetricInput(base)
		if vmMetrics.byJob != nil {
			if vm, ok := vmMetrics.byJob[base.ID]; ok {
				in = mergeFlinkMetricInput(in, vm)
				if metricsSource == "bch" {
					metricsSource = "mixed"
				}
			}
		}
		jobs = append(jobs, ComputeFlinkJobAnalysis(base.ID, base.Name, base.Owner, base.Cluster, in))
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	signals := make([]HealthSignal, 0, len(jobs))
	var healthy, warning, critical, anomaly int

	for _, job := range jobs {
		status := flinkStatusFromScore(job.Score)
		score := job.Score
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
			ID:            "sig-flink-" + job.ID,
			RunID:         runID,
			ScenarioKey:   scenarioKey,
			ObjectType:    HealthObjectJob,
			ObjectID:      job.ID,
			ClusterID:     job.Cluster,
			Domain:        domain,
			Type:          SignalTypeBCHWorkload,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "collector:flink_metrics_batch",
			SourceKind:    SourceKindCollector,
			Evidence: map[string]interface{}{
				"name":       job.Name,
				"owner":      job.Owner,
				"cluster":    job.Cluster,
				"score":      job.Score,
				"metrics":    job.Metrics,
				"penalties":  job.Penalties,
				"rootCause":  job.RootCause,
				"diagnosis":  job.Diagnosis,
			},
			ObservedAt: now,
			TTLSec:     ttl,
			Freshness:  FreshnessOK,
		})
	}

	snapshot := aggregateFlinkDomainSnapshot(domain, runID, scenarioKey, jobs, metricsSource, now, ttl)
	if err := healthStore.UpsertSignals(signals); err != nil {
		return FlinkL0Result{}, err
	}
	if err := healthStore.UpsertSnapshots([]HealthSnapshot{snapshot}); err != nil {
		return FlinkL0Result{}, err
	}

	return FlinkL0Result{
		JobCount:      len(jobs),
		HealthyCount:  healthy,
		WarningCount:  warning,
		CriticalCount: critical,
		AnomalyCount:  anomaly,
		MetricsSource: metricsSource,
	}, nil
}

func mergeFlinkMetricInput(base, overlay FlinkMetricInput) FlinkMetricInput {
	if overlay.MaxLag > 0 || overlay.LagTrend > 0 {
		base.MaxLag = overlay.MaxLag
		base.AvgLag = overlay.AvgLag
		base.LagTrend = overlay.LagTrend
	}
	if overlay.Restarts > 0 {
		base.Restarts = overlay.Restarts
	}
	if overlay.IsBP {
		base.IsBP = true
	}
	if overlay.CpuMax > 0 {
		base.CpuMax = overlay.CpuMax
		base.CpuAvg = overlay.CpuAvg
	}
	if overlay.HeapMax > 0 {
		base.HeapMax = overlay.HeapMax
	}
	if overlay.FullGc > 0 {
		base.FullGc = overlay.FullGc
	}
	return base
}

func aggregateFlinkDomainSnapshot(domain, runID, scenarioKey string, jobs []FlinkJob, metricsSource, observedAt string, ttl int) HealthSnapshot {
	var sum int
	for _, j := range jobs {
		sum += j.Score
	}
	avg := 0
	if len(jobs) > 0 {
		avg = sum / len(jobs)
	}
	status := scoreStatusFromScore(avg)
	return HealthSnapshot{
		SchemaVersion:            "1",
		AggregationPolicyVersion: "flink:l0:v1",
		ObjectType:               HealthObjectDomain,
		ObjectID:                 FlinkDomainSnapshotID,
		Domain:                   domain,
		Score:                    &avg,
		ScoreStatus:              status,
		Source:                   "collector:flink_metrics_batch",
		Coverage:                 1,
		PresentSources:           []string{SignalTypeBCHWorkload},
		Signals:                  nil,
		ObservedAt:               observedAt,
		MissingSources:           nil,
	}
}

// ListFlinkJobsHealth returns Flink jobs from L3 Facts when available, else BCH service.
func ListFlinkJobsHealth() ([]FlinkJob, error) {
	if healthStore != nil {
		signals, err := healthStore.ListSignals()
		if err == nil {
			jobs := flinkJobsFromSignals(signals)
			if len(jobs) > 0 {
				return jobs, nil
			}
		}
	}
	return BchServiceProvider().ListFlinkJobs()
}

func flinkJobsFromSignals(signals []HealthSignal) []FlinkJob {
	out := make([]FlinkJob, 0)
	for _, s := range signals {
		if s.ObjectType != HealthObjectJob || s.Type != SignalTypeBCHWorkload {
			continue
		}
		job := FlinkJob{
			ID:      s.ObjectID,
			Cluster: s.ClusterID,
			Status:  "RUNNING",
		}
		if s.Score != nil {
			job.Score = *s.Score
		}
		if s.Evidence != nil {
			if v, ok := s.Evidence["name"].(string); ok {
				job.Name = v
			}
			if v, ok := s.Evidence["owner"].(string); ok {
				job.Owner = v
			}
			if v, ok := s.Evidence["cluster"].(string); ok && job.Cluster == "" {
				job.Cluster = v
			}
			if v, ok := s.Evidence["diagnosis"].(string); ok {
				job.Diagnosis = v
			}
			if v, ok := s.Evidence["rootCause"].(string); ok {
				job.RootCause = v
			}
			if raw, ok := s.Evidence["metrics"].(map[string]interface{}); ok {
				job.Metrics = decodeFlinkJobMetric(raw)
			}
		}
		out = append(out, job)
	}
	return out
}

func decodeFlinkJobMetric(raw map[string]interface{}) FlinkJobMetric {
	var m FlinkJobMetric
	if v, ok := raw["lagTrend"].(float64); ok {
		m.LagTrend = int(v)
	}
	if v, ok := raw["maxLag"].(float64); ok {
		m.MaxLag = int64(v)
	}
	if v, ok := raw["avgLag"].(float64); ok {
		m.AvgLag = int64(v)
	}
	if v, ok := raw["isBP"].(bool); ok {
		m.IsBackpressured = v
	}
	if v, ok := raw["isBackpressured"].(bool); ok {
		m.IsBackpressured = v
	}
	if v, ok := raw["cpuMax"].(float64); ok {
		m.CpuMax = int(v)
	}
	if v, ok := raw["restarts"].(float64); ok {
		m.Restarts = int(v)
	}
	return m
}
