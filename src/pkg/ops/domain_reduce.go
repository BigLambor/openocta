package ops

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	ScenarioReduceSuffix = ":reduce"
	SourceDomainReduce   = "reduce:domain"
)

// DomainReduceObjectInput is one escalated object's context for Reduce.
type DomainReduceObjectInput struct {
	ObjectID   string                 `json:"objectId"`
	ObjectType string                 `json:"objectType"`
	L2Status   string                 `json:"l2Status"`
	L2Error    string                 `json:"l2Error,omitempty"`
	ChildRunID string                 `json:"childRunId,omitempty"`
	Signal     HealthSignal           `json:"signal"`
	Report     *InspectionReport      `json:"report,omitempty"`
}

// DomainReduceResult is the aggregated domain-level summary.
type DomainReduceResult struct {
	ParentRunID        string                   `json:"parentRunId"`
	PlanID             string                   `json:"planId"`
	ScenarioKey        string                   `json:"scenarioKey"`
	Domain             string                   `json:"domain"`
	DomainSnapshotID   string                   `json:"domainSnapshotId"`
	TriggerRef         string                   `json:"triggerRef"`
	Mode               string                   `json:"mode"`
	EscalatedCount     int                      `json:"escalatedCount"`
	L2Succeeded        int                      `json:"l2Succeeded"`
	L2Failed           int                      `json:"l2Failed"`
	TopRisks           []string                 `json:"topRisks"`
	ObjectSummaries    []map[string]interface{} `json:"objectSummaries"`
	Summary            string                   `json:"summary"`
	ReportMarkdown     string                   `json:"reportMarkdown"`
}

// DomainReduceTaskInput describes one completed L2 escalation task.
type DomainReduceTaskInput struct {
	ObjectID   string
	ObjectType string
	Status     string
	Error      string
	ChildRunID string
}

// DomainSnapshotIDForScenario maps batch scenarios to domain snapshot object IDs.
func DomainSnapshotIDForScenario(scenarioKey string) string {
	switch strings.TrimSpace(scenarioKey) {
	case ScenarioFlinkHealth:
		return FlinkDomainSnapshotID
	case ScenarioSparkHealth:
		return SparkDomainSnapshotID
	case ScenarioYarnHealth:
		return YarnDomainSnapshotID
	case ScenarioGBaseInstanceHealth:
		return GBaseInstancesDomainSnapshotID
	case ScenarioDataAppsPipelineHealth:
		return DataAppsPipelinesDomainSnapshotID
	default:
		return ""
	}
}

// CollectDomainReduceInputs gathers L0 signals and optional L2 reports for Reduce.
func CollectDomainReduceInputs(parentRunID, scenarioKey string, tasks []DomainReduceTaskInput) ([]DomainReduceObjectInput, error) {
	signals, err := ListSignalsForRun(parentRunID, scenarioKey, "")
	if err != nil {
		return nil, err
	}
	byObject := map[string]HealthSignal{}
	for _, s := range signals {
		byObject[s.ObjectID] = s
	}

	childRunIDs := make([]string, 0, len(tasks))
	for _, t := range tasks {
		if id := strings.TrimSpace(t.ChildRunID); id != "" {
			childRunIDs = append(childRunIDs, id)
		}
	}
	reportsByRun, _ := ListInspectionReportsByRunIDs(childRunIDs)

	out := make([]DomainReduceObjectInput, 0, len(tasks))
	for _, t := range tasks {
		sig := byObject[t.ObjectID]
		in := DomainReduceObjectInput{
			ObjectID:   t.ObjectID,
			ObjectType: t.ObjectType,
			L2Status:   t.Status,
			L2Error:    t.Error,
			ChildRunID: t.ChildRunID,
			Signal:     sig,
		}
		if rep, ok := reportsByRun[t.ChildRunID]; ok {
			cp := rep
			in.Report = &cp
		} else if rep, ok := reportsByRun[strings.TrimSpace(t.ChildRunID)]; ok {
			cp := rep
			in.Report = &cp
		}
		out = append(out, in)
	}
	return out, nil
}

// BuildRuleDomainReduce builds a deterministic domain summary from Map outputs.
func BuildRuleDomainReduce(parentRunID, planID, scenarioKey, triggerRef string, inputs []DomainReduceObjectInput) DomainReduceResult {
	domain := domainForScenario(scenarioKey)
	result := DomainReduceResult{
		ParentRunID:      parentRunID,
		PlanID:           planID,
		ScenarioKey:      scenarioKey,
		Domain:           domain,
		DomainSnapshotID: DomainSnapshotIDForScenario(scenarioKey),
		TriggerRef:       triggerRef,
		Mode:             "rule",
		EscalatedCount:   len(inputs),
		TopRisks:         []string{},
		ObjectSummaries:  []map[string]interface{}{},
	}

	riskSet := map[string]struct{}{}
	for _, in := range inputs {
		switch in.L2Status {
		case "succeeded":
			result.L2Succeeded++
		default:
			result.L2Failed++
		}
		summary := map[string]interface{}{
			"objectId":   in.ObjectID,
			"objectType": in.ObjectType,
			"l2Status":   in.L2Status,
		}
		if in.Signal.Score != nil {
			summary["l0Score"] = *in.Signal.Score
		}
		if in.Signal.Status != "" {
			summary["l0Status"] = in.Signal.Status
		}
		if in.Signal.Evidence != nil {
			if v, ok := in.Signal.Evidence["diagnosis"].(string); ok && strings.TrimSpace(v) != "" {
				summary["diagnosis"] = v
				riskSet[v] = struct{}{}
			}
			if v, ok := in.Signal.Evidence["name"].(string); ok {
				summary["name"] = v
			}
		}
		if in.Report != nil && strings.TrimSpace(in.Report.Summary) != "" {
			summary["l2Summary"] = in.Report.Summary
			riskSet[in.Report.Summary] = struct{}{}
		}
		if in.L2Error != "" {
			summary["l2Error"] = in.L2Error
		}
		result.ObjectSummaries = append(result.ObjectSummaries, summary)
	}

	for r := range riskSet {
		result.TopRisks = append(result.TopRisks, r)
	}
	sort.Strings(result.TopRisks)
	if len(result.TopRisks) > 10 {
		result.TopRisks = result.TopRisks[:10]
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("域级汇总（%s）：本轮 L0 升级 %d 个对象，L2 成功 %d、失败/超时 %d。\n",
		scenarioKey, result.EscalatedCount, result.L2Succeeded, result.L2Failed))
	if len(result.TopRisks) > 0 {
		b.WriteString("\n## Top 风险\n")
		for i, r := range result.TopRisks {
			b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r))
		}
	}
	result.Summary = strings.TrimSpace(b.String())
	result.ReportMarkdown = result.Summary
	return result
}

// BuildDomainReduceLLMMessage builds the Reduce LLM prompt from Map inputs.
func BuildDomainReduceLLMMessage(result DomainReduceResult, inputs []DomainReduceObjectInput) string {
	payload := map[string]interface{}{
		"scenarioKey":    result.ScenarioKey,
		"domain":         result.Domain,
		"escalatedCount": result.EscalatedCount,
		"l2Succeeded":    result.L2Succeeded,
		"l2Failed":       result.L2Failed,
		"objects":        inputs,
	}
	raw, _ := json.Marshal(payload)
	var b strings.Builder
	b.WriteString("你是运维域级分析专家。以下是一次批量巡检中多个异常对象的 L2 诊断上下文，请做一次 Reduce 汇总。\n\n")
	b.WriteString("## 输入（Map 结果 JSON）\n```json\n")
	b.Write(raw)
	b.WriteString("\n```\n\n")
	b.WriteString("## 要求\n")
	b.WriteString("1. 归纳跨对象的趋势与共性根因。\n")
	b.WriteString("2. 列出 Top 5 风险（按影响排序）。\n")
	b.WriteString("3. 给出域级处置优先级建议。\n")
	b.WriteString("4. 用简体中文输出结构化 Markdown。\n")
	return b.String()
}

// PersistDomainReduceSummary writes Reduce output to Facts and inspection_reports.
func PersistDomainReduceSummary(result DomainReduceResult) error {
	if healthStore == nil {
		return fmt.Errorf("health store 未初始化")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 3600
	score := aggregateReduceScore(result)
	status := scoreStatusFromScore(score)

	reportID := strings.TrimSpace(result.ParentRunID) + ScenarioReduceSuffix
	report := InspectionReport{
		ID:             reportID,
		JobID:          result.TriggerRef,
		Domain:         result.Domain,
		ScenarioKey:    result.ScenarioKey + ScenarioReduceSuffix,
		Score:          &score,
		ScoreStatus:    status,
		SourceKind:     SourceKindCollector,
		TriggerType:    "cron",
		Confidence:     "high",
		Summary:        result.Summary,
		Risks:          result.TopRisks,
		ReportMarkdown: result.ReportMarkdown,
		StartedAt:      time.Now().UnixMilli(),
		FinishedAt:     time.Now().UnixMilli(),
	}
	if err := persistInspectionReport(report); err != nil {
		return err
	}

	signal := HealthSignal{
		SchemaVersion: "1",
		ID:            "sig-reduce-" + result.DomainSnapshotID,
		RunID:         result.ParentRunID,
		ScenarioKey:   result.ScenarioKey,
		ObjectType:    HealthObjectDomain,
		ObjectID:      result.DomainSnapshotID,
		Domain:        result.Domain,
		Type:          SignalTypeInspection,
		Status:        scoreStatusToHealthStatus(status),
		Score:         &score,
		Confidence:    "high",
		Source:        SourceDomainReduce,
		SourceKind:    SourceKindCollector,
		Evidence: map[string]interface{}{
			"mode":            result.Mode,
			"escalatedCount":  result.EscalatedCount,
			"l2Succeeded":     result.L2Succeeded,
			"l2Failed":        result.L2Failed,
			"topRisks":        result.TopRisks,
			"objectSummaries": result.ObjectSummaries,
			"summary":         result.Summary,
			"reportId":        reportID,
		},
		ObservedAt: now,
		TTLSec:     ttl,
		Freshness:  FreshnessOK,
	}
	return healthStore.UpsertSignals([]HealthSignal{signal})
}

// MergeLLMIntoDomainReduce updates rule summary with LLM markdown when available.
func MergeLLMIntoDomainReduce(result DomainReduceResult, llmMarkdown string) DomainReduceResult {
	llmMarkdown = strings.TrimSpace(llmMarkdown)
	if llmMarkdown == "" {
		return result
	}
	result.Mode = "llm"
	result.ReportMarkdown = llmMarkdown
	if len(llmMarkdown) > 280 {
		result.Summary = llmMarkdown[:280] + "..."
	} else {
		result.Summary = llmMarkdown
	}
	return result
}

func aggregateReduceScore(result DomainReduceResult) int {
	if result.EscalatedCount == 0 {
		return 100
	}
	successRate := float64(result.L2Succeeded) / float64(result.EscalatedCount)
	score := int(successRate * 100)
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func domainForScenario(scenarioKey string) string {
	switch strings.TrimSpace(scenarioKey) {
	case ScenarioFlinkHealth, ScenarioSparkHealth, ScenarioYarnHealth:
		return DomainHadoop
	case ScenarioGBaseInstanceHealth:
		return DomainGBase
	case ScenarioDataAppsPipelineHealth:
		return DomainDataApps
	default:
		if s, ok := GetOpsScenario(scenarioKey); ok {
			return s.DomainKey
		}
		return ""
	}
}

// NewDomainReduceReportID returns a stable reduce report id.
func NewDomainReduceReportID(parentRunID string) string {
	id := strings.TrimSpace(parentRunID)
	if id == "" {
		id = uuid.New().String()
	}
	return id + ScenarioReduceSuffix
}
