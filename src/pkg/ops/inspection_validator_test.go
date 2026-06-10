package ops

import "testing"

func TestValidateInspectionReportPayload(t *testing.T) {
	score := 88
	ok, errs := ValidateInspectionReportPayload(InspectionResult{
		Score:       &score,
		ScoreStatus: ScoreStatusWarning,
	})
	if !ok || len(errs) != 0 {
		t.Fatalf("expected valid report, got ok=%v errs=%v", ok, errs)
	}

	badScore := 150
	ok, errs = ValidateInspectionReportPayload(InspectionResult{Score: &badScore})
	if ok || len(errs) == 0 {
		t.Fatalf("expected invalid score, got ok=%v errs=%v", ok, errs)
	}

	ok, errs = ValidateInspectionReportPayload(InspectionResult{})
	if ok {
		t.Fatalf("expected empty report to be invalid")
	}
}

func TestParseInspectionResultRejectsLegacyTextByDefault(t *testing.T) {
	res := ParseInspectionResultWithOptions(
		"session-legacy",
		"job-inspect-hadoop-deep",
		"健康得分: 82\n存在 YARN 队列压力。",
		"ok",
		1000,
		250,
		InspectionContext{Domain: DomainHadoop, ClusterID: "cluster-1"},
		ParseInspectionOptions{AllowLegacyTextScore: false},
	)

	if res.Score != nil {
		t.Fatalf("expected no regex score on commercial path, got %+v", res.Score)
	}
	if res.ScoreSource != ScoreSourceNone {
		t.Fatalf("expected scoreSource none, got %s", res.ScoreSource)
	}
	if res.ValidationStatus != ValidationStatusMissing {
		t.Fatalf("expected validationStatus missing, got %s", res.ValidationStatus)
	}
	if res.ScoreStatus != ScoreStatusUnknown {
		t.Fatalf("expected unknown status, got %s", res.ScoreStatus)
	}
}

func TestParseInspectionResultLegacyTextOptIn(t *testing.T) {
	res := ParseInspectionResultWithOptions(
		"session-legacy",
		"job-inspect-hadoop-deep",
		"健康得分: 82\n存在 YARN 队列压力。",
		"ok",
		1000,
		250,
		InspectionContext{Domain: DomainHadoop, ClusterID: "cluster-1"},
		ParseInspectionOptions{AllowLegacyTextScore: true},
	)

	if res.Score == nil || *res.Score != 82 {
		t.Fatalf("expected legacy score 82, got %+v", res.Score)
	}
	if res.ScoreSource != ScoreSourceLegacyText {
		t.Fatalf("expected legacy_text score source, got %s", res.ScoreSource)
	}
}

func TestParseInspectionResultInvalidStructuredJSON(t *testing.T) {
	res := ParseInspectionResultWithContext(
		"session-invalid",
		"job-inspect-gbase",
		"```json\n{\"domain\":\"gbase\",\"score\":150,\"scoreStatus\":\"ok\"}\n```",
		"ok",
		1000,
		250,
		InspectionContext{},
	)

	if res.Score != nil {
		t.Fatalf("invalid structured output must not keep score, got %+v", res.Score)
	}
	if res.ScoreStatus != ScoreStatusDegraded {
		t.Fatalf("expected degraded, got %s", res.ScoreStatus)
	}
	if res.ValidationStatus != ValidationStatusInvalid {
		t.Fatalf("expected invalid validation status, got %s", res.ValidationStatus)
	}
	if res.ScoreSource != ScoreSourceInvalidStructured {
		t.Fatalf("expected invalid_structured source, got %s", res.ScoreSource)
	}
	if len(res.ValidationErrors) == 0 {
		t.Fatal("expected validation errors")
	}
}

func TestParseInspectionResultValidStructuredWithMetadata(t *testing.T) {
	res := ParseInspectionResultWithContext(
		"session-valid",
		"job-inspect-gbase",
		"```json\n{\"domain\":\"gbase\",\"clusterId\":\"c1\",\"score\":91,\"scoreStatus\":\"ok\",\"confidence\":\"high\",\"summary\":\"all good\",\"risks\":[\"slow sql\"],\"recommendedActions\":[\"review indexes\"],\"requiresApproval\":true,\"toolRuns\":[{\"toolName\":\"query_gbase_slow_sql\",\"success\":true,\"output\":\"[]\"}]}\n```",
		"ok",
		1000,
		250,
		InspectionContext{},
	)

	if res.ValidationStatus != ValidationStatusValid {
		t.Fatalf("expected valid, got %s", res.ValidationStatus)
	}
	if res.Confidence != "high" || res.Summary != "all good" {
		t.Fatalf("metadata not applied: %+v", res)
	}
	if len(res.Risks) != 1 || len(res.RecommendedActions) != 1 {
		t.Fatalf("risks/actions not applied: %+v", res)
	}
	if res.RequiresApproval == nil || !*res.RequiresApproval {
		t.Fatalf("requiresApproval not applied")
	}
}
