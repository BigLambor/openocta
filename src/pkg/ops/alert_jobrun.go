package ops

import (
	"strings"

	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/jobrun"
)

const AlertDiagnosisJobID = "alert-diagnosis"

// StartAlertDiagnosisRun creates a running JobRun for an alert group diagnosis.
func StartAlertDiagnosisRun(g AlertGroup) error {
	jr := jobrun.Default()
	if jr == nil {
		return nil
	}
	runID := strings.TrimSpace(g.RunID)
	if runID == "" {
		return nil
	}
	if _, err := jr.Get(runID); err == nil {
		return nil
	}
	input := map[string]interface{}{
		"alertGroupId": g.ID,
		"source":       g.Source,
		"domain":       g.Domain,
		"severity":     g.Severity,
		"title":        g.Title,
		"sessionKey":   g.SessionKey,
		"eventCount":   g.OriginalCount,
	}
	if len(g.Events) > 0 {
		input["alertname"] = g.Events[0].Alertname
		input["service"] = g.Events[0].Service
		input["clusterId"] = g.Events[0].ClusterID
		input["component"] = g.Events[0].Component
	}
	_, err := jr.Start(jobrun.StartInput{
		RunID:       runID,
		JobID:       AlertDiagnosisJobID,
		TriggerType: jobrun.TriggerAlert,
		TriggerRef:  g.ID,
		Input:       input,
	})
	return err
}

// BindAlertDiagnosisChatRun associates a manual/workbench chat run with an alert group JobRun.
func BindAlertDiagnosisChatRun(alertGroupID, runID, sessionKey string) error {
	alertGroupID = strings.TrimSpace(alertGroupID)
	runID = strings.TrimSpace(runID)
	if alertGroupID == "" || runID == "" {
		return nil
	}

	alertsMu.Lock()
	defer alertsMu.Unlock()

	idx := findAlertGroupIndexLocked(alertGroupID)
	if idx < 0 {
		return nil
	}
	g := alertGroups[idx]
	if strings.TrimSpace(g.RunID) != runID {
		g.RunID = runID
	}
	if sk := strings.TrimSpace(sessionKey); sk != "" {
		g.SessionKey = sk
	}
	g = appendDiagnosisStartedTimeline(g)
	alertGroups[idx] = g
	if err := StartAlertDiagnosisRun(g); err != nil {
		return err
	}
	return persistAlertsLocked()
}

// SyncAlertDiagnosisAfterChat updates JobRun state after an alert-related chat turn finishes.
func SyncAlertDiagnosisAfterChat(sessionKey, runID string, chatSucceeded bool, errMsg string) {
	sessionKey = strings.TrimSpace(sessionKey)
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return
	}

	alertsMu.Lock()
	defer alertsMu.Unlock()

	idx := findAlertGroupIndexBySessionOrRunLocked(sessionKey, runID)
	if idx < 0 {
		return
	}
	stored := alertGroups[idx]
	if strings.TrimSpace(stored.RunID) == "" {
		stored.RunID = runID
	}
	enriched := enrichAlertGroupFromSession(stored)

	jr := jobrun.Default()
	if jr == nil {
		return
	}
	if !chatSucceeded {
		_ = jr.Fail(runID, strings.TrimSpace(errMsg), map[string]interface{}{
			"alertGroupId": stored.ID,
			"sessionKey":   sessionKey,
		})
		stored = appendDiagnosisFailedTimeline(stored, runID, errMsg)
		alertGroups[idx] = stored
		_ = persistAlertsLocked()
		return
	}

	if updated, changed := finishAlertDiagnosisRunUnlocked(stored, enriched); changed {
		alertGroups[idx] = updated
		_ = persistAlertsLocked()
	}
}

// MaybeFinishAlertDiagnosisRun completes JobRun when diagnosis output is ready.
// Caller must hold alertsMu.
func MaybeFinishAlertDiagnosisRun(stored, enriched AlertGroup) (AlertGroup, bool) {
	return finishAlertDiagnosisRunUnlocked(stored, enriched)
}

func finishAlertDiagnosisRunUnlocked(stored, enriched AlertGroup) (AlertGroup, bool) {
	runID := strings.TrimSpace(stored.RunID)
	if runID == "" || enriched.DiagnosticStatus != "completed" || strings.TrimSpace(enriched.RootCauseMarkdown) == "" {
		return stored, false
	}

	jr := jobrun.Default()
	if jr == nil {
		return stored, false
	}
	run, err := jr.Get(runID)
	if err != nil {
		return stored, false
	}
	if run.Status == jobrun.StatusSucceeded || run.Status == jobrun.StatusFailed || run.Status == jobrun.StatusCancelled {
		return stored, false
	}

	appendAlertDiagnosisSteps(jr, runID, tools.SessionIDFromSessionKey(enriched.SessionKey))

	output := map[string]interface{}{
		"diagnosticStatus": enriched.DiagnosticStatus,
		"rootCauseSummary": enriched.RootCauseSummary,
		"impactAnalysis":   enriched.ImpactAnalysis,
		"suggestedActions": enriched.SuggestedActions,
		"alertGroupId":     stored.ID,
	}
	if enriched.Evidence != nil {
		output["evidence"] = enriched.Evidence
	}
	_ = jr.Succeed(runID, jobrun.FinishInput{Output: output})

	stored.DiagnosticStatus = "completed"
	stored.RootCauseMarkdown = enriched.RootCauseMarkdown
	stored.RootCauseSummary = enriched.RootCauseSummary
	stored.ImpactMarkdown = enriched.ImpactMarkdown
	stored.ImpactAnalysis = enriched.ImpactAnalysis
	stored.SuggestedActions = enriched.SuggestedActions
	if enriched.Evidence != nil {
		stored.Evidence = enriched.Evidence
	}
	stored = appendDiagnosisCompletedTimeline(stored, runID)
	stored.UpdatedAtMs = nowMs()
	return stored, true
}

func appendAlertDiagnosisSteps(jr *jobrun.Service, runID, sessionID string) {
	if jr == nil || strings.TrimSpace(runID) == "" {
		return
	}
	existing, err := jr.ListSteps(runID)
	if err == nil && len(existing) > 0 {
		return
	}
	for _, toolRun := range parseAlertGroupToolRuns(sessionID) {
		status := jobrun.StatusSucceeded
		errText := ""
		if !toolRun.Success {
			status = jobrun.StatusFailed
			errText = toolRun.Error
		}
		jobrun.RecordToolExecution(jobrun.ToolExecutionInput{
			RunID:    runID,
			ToolName: toolRun.ToolName,
			Output:   toolRun.Output,
			Status:   status,
			Error:    errText,
		})
	}
}

func appendDiagnosisStartedTimeline(g AlertGroup) AlertGroup {
	for _, item := range g.Timeline {
		if item.Type == "diagnosis_started" {
			return g
		}
	}
	now := nowMs()
	g.Timeline = append(g.Timeline, AlertTimelineEvent{
		Type:        "diagnosis_started",
		Operator:    "system",
		TimestampMs: now,
		RunID:       strings.TrimSpace(g.RunID),
		Message:     "AI 诊断 JobRun 已启动",
	})
	g.UpdatedAtMs = now
	return g
}

func appendDiagnosisCompletedTimeline(g AlertGroup, runID string) AlertGroup {
	for _, item := range g.Timeline {
		if item.Type == "diagnosis_completed" {
			return g
		}
	}
	now := nowMs()
	g.Timeline = append(g.Timeline, AlertTimelineEvent{
		Type:        "diagnosis_completed",
		Operator:    "system",
		TimestampMs: now,
		RunID:       strings.TrimSpace(runID),
		Message:     "AI 诊断已完成，根因报告已生成",
	})
	g.UpdatedAtMs = now
	return g
}

func appendDiagnosisFailedTimeline(g AlertGroup, runID, errMsg string) AlertGroup {
	now := nowMs()
	msg := "AI 诊断失败"
	if trimmed := strings.TrimSpace(errMsg); trimmed != "" {
		msg = msg + "：" + trimmed
	}
	g.Timeline = append(g.Timeline, AlertTimelineEvent{
		Type:        "diagnosis_failed",
		Operator:    "system",
		TimestampMs: now,
		RunID:       strings.TrimSpace(runID),
		Message:     msg,
	})
	g.UpdatedAtMs = now
	return g
}

func findAlertGroupIndexLocked(id string) int {
	id = strings.TrimSpace(id)
	for i, g := range alertGroups {
		if g.ID == id {
			return i
		}
	}
	return -1
}

func findAlertGroupIndexBySessionOrRunLocked(sessionKey, runID string) int {
	sessionKey = strings.TrimSpace(sessionKey)
	runID = strings.TrimSpace(runID)
	for i, g := range alertGroups {
		if runID != "" && strings.TrimSpace(g.RunID) == runID {
			return i
		}
		if sessionKey != "" && strings.TrimSpace(g.SessionKey) == sessionKey {
			return i
		}
	}
	return -1
}
