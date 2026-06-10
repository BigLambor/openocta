package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// GBaseInstanceL0Opts configures batch GBase instance L0.
type GBaseInstanceL0Opts struct {
	RunID       string
	ScenarioKey string
	Domain      string
}

// GBaseInstanceL0Result summarizes L0 batch run.
type GBaseInstanceL0Result struct {
	InstanceCount int `json:"instanceCount"`
	HealthyCount  int `json:"healthyCount"`
	WarningCount  int `json:"warningCount"`
	CriticalCount int `json:"criticalCount"`
	AnomalyCount  int `json:"anomalyCount"`
}

func scoreGBaseInstance(inst GBaseInstance) (int, string) {
	score := 100
	diagnosis := "实例运行正常。"
	if inst.MaxConnections > 0 {
		util := float64(inst.ActiveConnections) / float64(inst.MaxConnections)
		if util > 0.85 {
			score -= 25
			diagnosis = "连接池使用率过高。"
		} else if util > 0.7 {
			score -= 10
		}
	}
	if inst.SlowSQLCount > 20 {
		score -= 30
		diagnosis = "慢 SQL 数量过多，需优化索引或 SQL。"
	} else if inst.SlowSQLCount > 5 {
		score -= 15
		diagnosis = "存在慢 SQL，建议排查。"
	}
	if strings.EqualFold(inst.Status, "critical") {
		score = 30
	}
	if score < 0 {
		score = 0
	}
	return score, diagnosis
}

func gbaseInstanceStatusFromScore(score int) string {
	switch {
	case score >= 85:
		return HealthStatusHealthy
	case score >= 70:
		return HealthStatusWarning
	default:
		return HealthStatusCritical
	}
}

// RunGBaseInstanceHealthL0 batch-collects GBase instances and writes Facts.
func RunGBaseInstanceHealthL0(ctx context.Context, opts GBaseInstanceL0Opts) (GBaseInstanceL0Result, error) {
	if healthStore == nil {
		return GBaseInstanceL0Result{}, fmt.Errorf("health store 未初始化")
	}
	runID := strings.TrimSpace(opts.RunID)
	if runID == "" {
		runID = uuid.New().String()
	}
	scenarioKey := strings.TrimSpace(opts.ScenarioKey)
	if scenarioKey == "" {
		scenarioKey = ScenarioGBaseInstanceHealth
	}
	domain := strings.TrimSpace(opts.Domain)
	if domain == "" {
		domain = DomainGBase
	}

	inventory, err := ListGBaseInstances()
	if err != nil {
		return GBaseInstanceL0Result{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	signals := make([]HealthSignal, 0, len(inventory))
	var healthy, warning, critical, anomaly int

	for _, inst := range inventory {
		score, diagnosis := scoreGBaseInstance(inst)
		status := gbaseInstanceStatusFromScore(score)
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
			ID:            "sig-gbase-inst-" + inst.ID,
			RunID:         runID,
			ScenarioKey:   scenarioKey,
			ObjectType:    HealthObjectDBInstance,
			ObjectID:      inst.ID,
			ClusterID:     inst.Cluster,
			Domain:        domain,
			Type:          SignalTypeGBaseSQL,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "collector:gbase_instance_batch",
			SourceKind:    SourceKindCollector,
			Evidence: map[string]interface{}{
				"name":              inst.Name,
				"cluster":           inst.Cluster,
				"activeConnections": inst.ActiveConnections,
				"maxConnections":    inst.MaxConnections,
				"slowSqlCount":      inst.SlowSQLCount,
				"qps":               inst.QPS,
				"diagnosis":         diagnosis,
				"score":             score,
			},
			ObservedAt: now,
			TTLSec:     ttl,
			Freshness:  FreshnessOK,
		})
	}

	snapshot := aggregateBatchDomainSnapshot(domain, GBaseInstancesDomainSnapshotID, "gbase:instance:l0:v1", signals, now, ttl)
	if err := healthStore.UpsertSignals(signals); err != nil {
		return GBaseInstanceL0Result{}, err
	}
	if err := healthStore.UpsertSnapshots([]HealthSnapshot{snapshot}); err != nil {
		return GBaseInstanceL0Result{}, err
	}

	return GBaseInstanceL0Result{
		InstanceCount: len(inventory), HealthyCount: healthy, WarningCount: warning,
		CriticalCount: critical, AnomalyCount: anomaly,
	}, nil
}
