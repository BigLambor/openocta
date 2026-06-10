package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/paths"
)

// InspectionReport is the structured L3 report schema for inspection runs.
type InspectionReport = InspectionResult

// PersistInspectionFacts writes structured inspection output into L3 Facts.
func PersistInspectionFacts(report InspectionReport) error {
	if report.SourceKind == "chat" && report.TriggerType != "manual_confirm" && report.TriggerType != "cron" && report.TriggerType != "alert_hook" {
		return saveDraftReport(report)
	}

	if strings.TrimSpace(report.ID) != "" {
		if err := persistInspectionReport(report); err != nil {
			return err
		}
	}

	if healthStore == nil || strings.TrimSpace(report.ClusterID) == "" || report.ClusterID == "all" {
		return nil
	}
	cluster, err := GetCluster(report.ClusterID)
	if err != nil {
		return nil
	}

	signals := make([]HealthSignal, 0, 2)
	if sig, ok := inspectionSignalFromReport(cluster, report); ok {
		signals = append(signals, sig)
	}
	if cluster.Domain == DomainGBase {
		if sig, ok := gbaseSQLSignalFromReport(cluster, report); ok {
			signals = append(signals, sig)
		}
	}
	if len(signals) == 0 {
		return nil
	}
	if err := healthStore.UpsertSignals(signals); err != nil {
		return err
	}

	allSignals, err := healthStore.ListSignals()
	if err != nil {
		return err
	}
	policy := defaultDomainHealthPolicy(cluster.Domain)
	if policy == nil {
		return nil
	}
	snapshot := AggregateHealthSnapshot(cluster, policy, signalsForObject(allSignals, HealthObjectCluster, cluster.ID))
	return healthStore.UpsertSnapshots([]HealthSnapshot{snapshot})
}

func inspectionSignalFromReport(cluster Cluster, report InspectionReport) (HealthSignal, bool) {
	if report.Score == nil && report.ScoreStatus == "" && len(report.Errors) == 0 {
		return HealthSignal{}, false
	}
	status := scoreStatusToHealthStatus(report.ScoreStatus)
	score := report.Score
	errText := ""
	if len(report.Errors) > 0 {
		errText = strings.Join(report.Errors, "; ")
	}
	confidence := strings.TrimSpace(report.Confidence)
	if confidence == "" {
		confidence = "medium"
	}
	return HealthSignal{
		SchemaVersion: "1",
		ID:            "sig-" + uuid.New().String(),
		RunID:         runIDFromInspectionReport(report),
		ScenarioKey:   ScenarioKeyForInspection(report),
		ObjectType:    HealthObjectCluster,
		ObjectID:      cluster.ID,
		ClusterID:     cluster.ID,
		Domain:        cluster.Domain,
		Region:        cluster.Region,
		Type:          SignalTypeInspection,
		Status:        status,
		Score:         score,
		Confidence:    confidence,
		Source:        "inspection:" + strings.TrimSpace(report.JobID),
		SourceKind:    SourceKindCollector,
		Evidence: map[string]interface{}{
			"jobId":                report.JobID,
			"sessionId":            report.ID,
			"component":            report.Component,
			"toolRuns":             report.ToolRuns,
			"startedAt":            report.StartedAt,
			"finishedAt":           report.FinishedAt,
			"summary":              report.Summary,
			"risks":                report.Risks,
			"recommendedActions":   report.RecommendedActions,
			"requiresApproval":     report.RequiresApproval,
			"validationStatus":     report.ValidationStatus,
			"metricsEvidence":      report.MetricsEvidence,
		},
		Error:      errText,
		ObservedAt: observedAtFromReport(report),
		TTLSec:     ttlForCluster(cluster),
		Freshness:  FreshnessOK,
	}, true
}

func gbaseSQLSignalFromReport(cluster Cluster, report InspectionReport) (HealthSignal, bool) {
	for _, run := range report.ToolRuns {
		if run.ToolName != "query_gbase_slow_sql" {
			continue
		}
		status := HealthStatusHealthy
		score := 100
		errText := ""
		evidence := map[string]interface{}{
			"jobId":     report.JobID,
			"sessionId": report.ID,
			"toolName":  run.ToolName,
		}
		if !run.Success {
			status = HealthStatusCritical
			score = 0
			errText = run.Error
			evidence["error"] = run.Error
		} else {
			slowSQLCount, parsedEvidence := parseSlowSQLEvidence(run.Output)
			evidence["slowSqlCount"] = slowSQLCount
			for k, v := range parsedEvidence {
				evidence[k] = v
			}
			if slowSQLCount > 0 {
				status = HealthStatusWarning
				score = 100 - slowSQLCount*8
				if score < 40 {
					score = 40
				}
			}
		}
		return HealthSignal{
			SchemaVersion: "1",
			ID:            "sig-" + uuid.New().String(),
			RunID:         runIDFromInspectionReport(report),
			ScenarioKey:   "ops-gbase-health",
			ObjectType:    HealthObjectCluster,
			ObjectID:      cluster.ID,
			ClusterID:     cluster.ID,
			Domain:        cluster.Domain,
			Region:        cluster.Region,
			Type:          SignalTypeGBaseSQL,
			Status:        status,
			Score:         &score,
			Confidence:    "high",
			Source:        "platform_tool:query_gbase_slow_sql",
			SourceKind:    SourceKindPlatformTool,
			Evidence:      evidence,
			Error:         errText,
			ObservedAt:    observedAtFromReport(report),
			TTLSec:        ttlForCluster(cluster),
			Freshness:     FreshnessOK,
		}, true
	}
	return HealthSignal{}, false
}

func parseSlowSQLEvidence(output string) (int, map[string]interface{}) {
	output = strings.TrimSpace(output)
	if output == "" || output == "[]" {
		return 0, map[string]interface{}{}
	}
	var envelope struct {
		Type         string                   `json:"type"`
		Status       string                   `json:"status"`
		SlowSQLCount int                      `json:"slowSqlCount"`
		Rows         []map[string]interface{} `json:"rows"`
		Error        string                   `json:"error"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err == nil && (envelope.Type == SignalTypeGBaseSQL || envelope.SlowSQLCount > 0 || envelope.Rows != nil) {
		evidence := map[string]interface{}{}
		if envelope.Status != "" {
			evidence["toolStatus"] = envelope.Status
		}
		if envelope.Rows != nil {
			evidence["rows"] = envelope.Rows
		}
		if envelope.Error != "" {
			evidence["error"] = envelope.Error
		}
		return envelope.SlowSQLCount, evidence
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &rows); err == nil {
		return len(rows), map[string]interface{}{"rows": rows}
	}
	return 1, map[string]interface{}{"rawOutput": output}
}

func ScenarioKeyForInspection(report InspectionReport) string {
	if key := strings.TrimSpace(report.ScenarioKey); key != "" {
		return key
	}
	domain := strings.TrimSpace(report.Domain)
	if domain == "" {
		domain = DomainFromInspectJobID(report.JobID)
	}
	switch strings.TrimSpace(report.JobID) {
	case "job-inspect-flink":
		return ScenarioFlinkHealth
	case "job-inspect-spark":
		return ScenarioSparkHealth
	case "job-inspect-yarn":
		return ScenarioYarnHealth
	case "job-inspect-gbase-instances":
		return ScenarioGBaseInstanceHealth
	case "job-inspect-dataapps-pipelines":
		return ScenarioDataAppsPipelineHealth
	}
	switch domain {
	case DomainHadoop:
		return "ops-bch-health"
	case DomainFI:
		return "ops-fi-health"
	case DomainGBase:
		return "ops-gbase-health"
	case DomainGovernance:
		return "ops-governance-health"
	case DomainDataApps:
		return "ops-dataapps-health"
	default:
		return "system:inspection"
	}
}

func runIDFromInspectionReport(report InspectionReport) string {
	if strings.TrimSpace(report.ID) != "" {
		return "inspection-" + strings.TrimSpace(report.ID)
	}
	if strings.TrimSpace(report.JobID) != "" {
		return "inspection-" + strings.TrimSpace(report.JobID)
	}
	return "inspection-" + uuid.New().String()
}

func observedAtFromReport(report InspectionReport) string {
	if report.FinishedAt > 0 {
		return time.UnixMilli(report.FinishedAt).UTC().Format(time.RFC3339)
	}
	return time.Now().UTC().Format(time.RFC3339)
}

func ttlForCluster(cluster Cluster) int {
	if p := defaultDomainHealthPolicy(cluster.Domain); p != nil && p.DefaultTTLSec > 0 {
		return p.DefaultTTLSec
	}
	return 300
}

func scoreStatusToHealthStatus(status string) string {
	switch status {
	case ScoreStatusOK, "healthy":
		return HealthStatusHealthy
	case ScoreStatusWarning:
		return HealthStatusWarning
	case ScoreStatusCritical, ScoreStatusDegraded:
		return HealthStatusCritical
	default:
		return HealthStatusUnknown
	}
}

func saveDraftReport(report InspectionReport) error {
	stateDir := paths.ResolveStateDir(os.Getenv)
	draftDir := filepath.Join(stateDir, "ops")
	if err := os.MkdirAll(draftDir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(draftDir, "draft_reports.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	data, _ := json.Marshal(report)
	_, err = f.Write(append(data, '\n'))
	return err
}
