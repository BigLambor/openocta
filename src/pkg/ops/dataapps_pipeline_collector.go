package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DataAppsPipelineL0Opts configures batch pipeline L0.
type DataAppsPipelineL0Opts struct {
	RunID       string
	ScenarioKey string
	Domain      string
}

// DataAppsPipelineL0Result summarizes L0 batch run.
type DataAppsPipelineL0Result struct {
	PipelineCount int `json:"pipelineCount"`
	HealthyCount  int `json:"healthyCount"`
	WarningCount  int `json:"warningCount"`
	CriticalCount int `json:"criticalCount"`
	AnomalyCount  int `json:"anomalyCount"`
}

func scoreDataAppPipeline(p DataAppPipeline) (int, string) {
	score := 100
	diagnosis := "管道运行正常。"
	if p.FailedTasks > 0 {
		score -= minInt(p.FailedTasks*15, 60)
		diagnosis = "存在失败任务，需排查上游依赖。"
	}
	if p.SLABreach {
		score -= 25
		if diagnosis == "管道运行正常。" {
			diagnosis = "SLA 未达标，跑批延迟或失败。"
		}
	}
	if p.DelaySeconds > 3600 {
		score -= 15
	}
	if strings.EqualFold(p.Status, "failed") {
		score = minInt(score, 35)
		diagnosis = "管道失败，需立即介入。"
	}
	if score < 0 {
		score = 0
	}
	return score, diagnosis
}

func pipelineStatusFromScore(score int) string {
	switch {
	case score >= 85:
		return HealthStatusHealthy
	case score >= 70:
		return HealthStatusWarning
	default:
		return HealthStatusCritical
	}
}

// RunDataAppsPipelineHealthL0 batch-collects pipelines and writes Facts.
func RunDataAppsPipelineHealthL0(ctx context.Context, opts DataAppsPipelineL0Opts) (DataAppsPipelineL0Result, error) {
	if healthStore == nil {
		return DataAppsPipelineL0Result{}, fmt.Errorf("health store 未初始化")
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = uuid.New().String()
	}
	scenarioKey := strings.TrimSpace(opts.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ScenarioDataAppsPipelineHealth
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = DomainDataApps
	}

	inventory, err := ListDataAppPipelines()
	if err != nil {
		return DataAppsPipelineL0Result{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	signals := make([]HealthSignal, 0, len(inventory))
	var healthy, warning, critical, anomaly int

	for _, pipe := range inventory {
		score, diagnosis := scoreDataAppPipeline(pipe)
		status := pipelineStatusFromScore(score)
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
			ID:            "sig-pipeline-" + pipe.ID,
			RunID:         runID,
			ScenarioKey:   scenarioKey,
			ObjectType:    HealthObjectPipeline,
			ObjectID:      pipe.ID,
			ClusterID:     pipe.Cluster,
			Domain:        domain,
			Type:          SignalTypeSchedulerAPI,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "collector:dataapps_pipeline_batch",
			SourceKind:    SourceKindCollector,
			Evidence: map[string]interface{}{
				"name":         pipe.Name,
				"owner":        pipe.Owner,
				"cluster":      pipe.Cluster,
				"pipeStatus":   pipe.Status,
				"slaBreach":    pipe.SLABreach,
				"failedTasks":  pipe.FailedTasks,
				"delaySeconds": pipe.DelaySeconds,
				"diagnosis":    diagnosis,
				"score":        score,
			},
			ObservedAt: now,
			TTLSec:     ttl,
			Freshness:  FreshnessOK,
		})
	}

	snapshot := aggregateBatchDomainSnapshot(domain, DataAppsPipelinesDomainSnapshotID, "dataapps:pipeline:l0:v1", signals, now, ttl)
	if err := healthStore.UpsertSignals(signals); err != nil {
		return DataAppsPipelineL0Result{}, err
	}
	if err := healthStore.UpsertSnapshots([]HealthSnapshot{snapshot}); err != nil {
		return DataAppsPipelineL0Result{}, err
	}

	return DataAppsPipelineL0Result{
		PipelineCount: len(inventory), HealthyCount: healthy, WarningCount: warning,
		CriticalCount: critical, AnomalyCount: anomaly,
	}, nil
}
