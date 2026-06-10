package ops

import (
	"encoding/json"
	"fmt"
	"strings"
)

// EscalationPolicy defines when L0 results should trigger L2 diagnosis.
type EscalationPolicy struct {
	MinScore       int  `json:"minScore"`
	OnRestarts     bool `json:"onRestarts"`
	OnBackpressure bool `json:"onBackpressure"`
}

// FlinkEscalationPolicy returns the default Flink L0→L2 escalation rules.
func FlinkEscalationPolicy() EscalationPolicy {
	return EscalationPolicy{
		MinScore:       70,
		OnRestarts:     true,
		OnBackpressure: true,
	}
}

// ShouldEscalateFlinkSignal reports whether a Flink job signal needs L2 diagnosis.
func ShouldEscalateFlinkSignal(sig HealthSignal, policy EscalationPolicy) bool {
	if sig.ObjectType != HealthObjectJob || sig.Type != SignalTypeBCHWorkload {
		return false
	}
	minScore := policy.MinScore
	if minScore <= 0 {
		minScore = 70
	}
	score := 100
	if sig.Score != nil {
		score = *sig.Score
	}
	if score < minScore {
		return true
	}
	if sig.Status == HealthStatusCritical {
		return true
	}
	metrics := flinkMetricsFromEvidence(sig.Evidence)
	if policy.OnRestarts && metrics.Restarts > 0 {
		return true
	}
	if policy.OnBackpressure && metrics.IsBackpressured {
		return true
	}
	return false
}

// ListFlinkSignalsForRun returns Flink job signals produced by a specific L0 run.
func ListFlinkSignalsForRun(runID, scenarioKey string) ([]HealthSignal, error) {
	if scenarioKey == "" {
		scenarioKey = ScenarioFlinkHealth
	}
	return ListSignalsForRun(runID, scenarioKey, HealthObjectJob)
}

// ShouldEscalateSignal dispatches escalation checks by scenario.
func ShouldEscalateSignal(sig HealthSignal) bool {
	switch strings.TrimSpace(sig.ScenarioKey) {
	case ScenarioFlinkHealth:
		return ShouldEscalateFlinkSignal(sig, FlinkEscalationPolicy())
	case ScenarioSparkHealth:
		return shouldEscalateByScore(sig, 70)
	case ScenarioYarnHealth:
		return shouldEscalateYarnSignal(sig)
	case ScenarioGBaseInstanceHealth:
		return shouldEscalateGBaseInstanceSignal(sig)
	case ScenarioDataAppsPipelineHealth:
		return shouldEscalatePipelineSignal(sig)
	default:
		return false
	}
}

// BuildEscalationMessage builds an L2 prompt for a batch L0 signal.
func BuildEscalationMessage(sig HealthSignal) string {
	switch strings.TrimSpace(sig.ScenarioKey) {
	case ScenarioFlinkHealth:
		return BuildFlinkEscalationMessage(sig)
	default:
		return buildGenericEscalationMessage(sig)
	}
}

func shouldEscalateByScore(sig HealthSignal, minScore int) bool {
	score := 100
	if sig.Score != nil {
		score = *sig.Score
	}
	return score < minScore || sig.Status == HealthStatusCritical
}

func shouldEscalateYarnSignal(sig HealthSignal) bool {
	if shouldEscalateByScore(sig, 70) {
		return true
	}
	if sig.Evidence == nil {
		return false
	}
	if v, ok := sig.Evidence["queueStatus"].(string); ok {
		switch strings.TrimSpace(v) {
		case "under_allocated", "idle":
			return true
		}
	}
	if v, ok := sig.Evidence["riskLevel"].(string); ok && strings.EqualFold(v, "high") {
		return true
	}
	return false
}

func shouldEscalateGBaseInstanceSignal(sig HealthSignal) bool {
	if shouldEscalateByScore(sig, 70) {
		return true
	}
	if sig.Evidence == nil {
		return false
	}
	if v, ok := sig.Evidence["slowSqlCount"].(float64); ok && v > 5 {
		return true
	}
	if active, ok := sig.Evidence["activeConnections"].(float64); ok {
		if max, ok2 := sig.Evidence["maxConnections"].(float64); ok2 && max > 0 && active/max > 0.8 {
			return true
		}
	}
	return false
}

func shouldEscalatePipelineSignal(sig HealthSignal) bool {
	if shouldEscalateByScore(sig, 70) {
		return true
	}
	if sig.Evidence == nil {
		return false
	}
	if v, ok := sig.Evidence["slaBreach"].(bool); ok && v {
		return true
	}
	if v, ok := sig.Evidence["failedTasks"].(float64); ok && v > 0 {
		return true
	}
	if v, ok := sig.Evidence["pipeStatus"].(string); ok && strings.EqualFold(v, "failed") {
		return true
	}
	return false
}

func buildGenericEscalationMessage(sig HealthSignal) string {
	ctx := map[string]interface{}{
		"objectId":     sig.ObjectID,
		"objectType":   sig.ObjectType,
		"scenarioKey":  sig.ScenarioKey,
		"status":       sig.Status,
		"cluster":      sig.ClusterID,
		"domain":       sig.Domain,
		"evidence":     sig.Evidence,
	}
	if sig.Score != nil {
		ctx["score"] = *sig.Score
	}
	raw, _ := json.Marshal(ctx)
	var b strings.Builder
	b.WriteString("你是大数据运维专家。以下对象在 L0 批量巡检中被标记为异常，请基于结构化上下文进行根因分析与处置建议。\n\n")
	b.WriteString("## 结构化上下文\n```json\n")
	b.Write(raw)
	b.WriteString("\n```\n\n请用简体中文输出结构化 Markdown 报告。\n")
	return b.String()
}

// BuildFlinkEscalationMessage builds a structured L2 prompt for one Flink job.
func BuildFlinkEscalationMessage(sig HealthSignal) string {
	jobID := strings.TrimSpace(sig.ObjectID)
	name := jobID
	owner := ""
	cluster := strings.TrimSpace(sig.ClusterID)
	score := 0
	if sig.Score != nil {
		score = *sig.Score
	}
	metrics := flinkMetricsFromEvidence(sig.Evidence)
	if sig.Evidence != nil {
		if v, ok := sig.Evidence["name"].(string); ok {
			name = v
		}
		if v, ok := sig.Evidence["owner"].(string); ok {
			owner = v
		}
		if v, ok := sig.Evidence["cluster"].(string); ok && cluster == "" {
			cluster = v
		}
	}

	ctx := map[string]interface{}{
		"objectId":   jobID,
		"objectType": HealthObjectJob,
		"name":       name,
		"owner":      owner,
		"cluster":    cluster,
		"score":      score,
		"status":     sig.Status,
		"metrics":    metrics,
	}
	if sig.Evidence != nil {
		if v, ok := sig.Evidence["penalties"]; ok {
			ctx["penalties"] = v
		}
		if v, ok := sig.Evidence["rootCause"]; ok {
			ctx["rootCause"] = v
		}
		if v, ok := sig.Evidence["diagnosis"]; ok {
			ctx["diagnosis"] = v
		}
	}
	raw, _ := json.Marshal(ctx)

	var b strings.Builder
	b.WriteString("你是 Flink 运维专家。以下作业在 L0 批量巡检中被标记为异常，请基于结构化上下文进行根因分析与处置建议。\n\n")
	b.WriteString("## 结构化上下文\n")
	b.WriteString("```json\n")
	b.Write(raw)
	b.WriteString("\n```\n\n")
	b.WriteString("## 要求\n")
	b.WriteString("1. 结合 metrics 与 penalties 判断根因（积压、反压、重启、资源等）。\n")
	b.WriteString("2. 给出可执行的处置步骤（含优先级）。\n")
	b.WriteString("3. 用简体中文输出结构化 Markdown 报告。\n")
	if prefix := strings.TrimSpace(cluster); prefix != "" {
		b.WriteString(fmt.Sprintf("\n[运维上下文] domain=hadoop clusterId=%s component=flink\n", prefix))
	}
	return b.String()
}

func flinkMetricsFromEvidence(evidence map[string]interface{}) FlinkJobMetric {
	if evidence == nil {
		return FlinkJobMetric{}
	}
	raw, ok := evidence["metrics"].(map[string]interface{})
	if !ok {
		return FlinkJobMetric{}
	}
	return decodeFlinkJobMetric(raw)
}
