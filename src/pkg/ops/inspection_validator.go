package ops

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	ValidationStatusValid            = "valid"
	ValidationStatusInvalid          = "invalid"
	ValidationStatusMissing          = "missing"
	ScoreSourceStructured            = "structured"
	ScoreSourceInvalidStructured     = "invalid_structured"
	ScoreSourceLegacyText            = "legacy_text"
	ScoreSourceNone                  = "none"
)

var allowedScoreStatuses = map[string]struct{}{
	ScoreStatusOK:       {},
	ScoreStatusWarning:  {},
	ScoreStatusCritical: {},
	ScoreStatusUnknown:  {},
	ScoreStatusDegraded: {},
	ScoreStatusPartial:  {},
	"healthy":           {},
}

var allowedConfidenceLevels = map[string]struct{}{
	"high":   {},
	"medium": {},
	"low":    {},
}

// ValidateInspectionReportPayload checks a parsed InspectionReport for schema constraints.
func ValidateInspectionReportPayload(report InspectionResult) (valid bool, errors []string) {
	hasSignal := report.Score != nil ||
		strings.TrimSpace(report.ScoreStatus) != "" ||
		len(report.ToolRuns) > 0 ||
		report.MetricsEvidence != nil ||
		strings.TrimSpace(report.Summary) != "" ||
		strings.TrimSpace(report.ReportMarkdown) != "" ||
		len(report.Errors) > 0 ||
		len(report.Risks) > 0 ||
		len(report.RecommendedActions) > 0

	if !hasSignal {
		errors = append(errors, "structured report is empty: require score, scoreStatus, toolRuns, evidence, summary, or errors")
	}

	if report.Score != nil {
		if *report.Score < 0 || *report.Score > 100 {
			errors = append(errors, fmt.Sprintf("score out of range: %d (expected 0-100)", *report.Score))
		}
	}

	if status := normalizeScoreStatus(report.ScoreStatus); status != "" {
		if _, ok := allowedScoreStatuses[status]; !ok {
			errors = append(errors, fmt.Sprintf("invalid scoreStatus: %q", report.ScoreStatus))
		}
	}

	if conf := strings.TrimSpace(strings.ToLower(report.Confidence)); conf != "" {
		if _, ok := allowedConfidenceLevels[conf]; !ok {
			errors = append(errors, fmt.Sprintf("invalid confidence: %q", report.Confidence))
		}
	}

	for i, run := range report.ToolRuns {
		if strings.TrimSpace(run.ToolName) == "" {
			errors = append(errors, fmt.Sprintf("toolRuns[%d].toolName is required", i))
		}
	}

	return len(errors) == 0 && hasSignal, errors
}

func normalizeScoreStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "healthy":
		return ScoreStatusOK
	default:
		return status
	}
}

type structuredParseResult struct {
	found  bool
	valid  bool
	report InspectionResult
	errors []string
}

func parseStructuredInspectionReportStrict(text string) structuredParseResult {
	var lastInvalid structuredParseResult
	for _, candidate := range structuredJSONCandidates(text) {
		var parsed InspectionResult
		if err := jsonUnmarshalInspectionCandidate(candidate, &parsed); err != nil {
			if looksLikeInspectionJSON(candidate) {
				lastInvalid = structuredParseResult{
					found:  true,
					valid:  false,
					errors: []string{fmt.Sprintf("invalid JSON: %v", err)},
				}
			}
			continue
		}
		if !looksLikeInspectionPayload(parsed) {
			continue
		}
		ok, validationErrors := ValidateInspectionReportPayload(parsed)
		if ok {
			parsed.ScoreStatus = normalizeScoreStatus(parsed.ScoreStatus)
			if parsed.ScoreStatus == "" && parsed.Score != nil {
				parsed.ScoreStatus = scoreStatusFromScore(*parsed.Score)
			}
			return structuredParseResult{found: true, valid: true, report: parsed}
		}
		lastInvalid = structuredParseResult{
			found:  true,
			valid:  false,
			report: parsed,
			errors: validationErrors,
		}
	}
	return lastInvalid
}

func looksLikeInspectionJSON(candidate string) bool {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if !strings.HasPrefix(candidate, "{") {
		return false
	}
	for _, key := range []string{`"score"`, `"scorestatus"`, `"toolruns"`, `"metricsevidence"`, `"domain"`, `"errors"`} {
		if strings.Contains(candidate, key) {
			return true
		}
	}
	return false
}

func looksLikeInspectionPayload(parsed InspectionResult) bool {
	return parsed.Score != nil ||
		strings.TrimSpace(parsed.ScoreStatus) != "" ||
		len(parsed.ToolRuns) > 0 ||
		parsed.MetricsEvidence != nil ||
		strings.TrimSpace(parsed.Domain) != "" ||
		len(parsed.Errors) > 0
}

func jsonUnmarshalInspectionCandidate(candidate string, dst *InspectionResult) error {
	return json.Unmarshal([]byte(candidate), dst)
}
