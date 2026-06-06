package ops

import (
	"strings"
	"testing"
)

func TestEnrichInspectionWithMockReport(t *testing.T) {
	res := InspectionResult{Domain: DomainHadoop}
	EnrichInspectionWithMockReport(&res, "ops-bch-health", "")

	if res.Score == nil || *res.Score != 86 {
		t.Fatalf("expected mock score 86, got %+v", res.Score)
	}
	if res.ReportMarkdown == "" {
		t.Fatal("expected structured markdown report")
	}
	for _, part := range []string{"执行摘要", "风险项", "后续动作"} {
		if !strings.Contains(res.ReportMarkdown, part) {
			t.Fatalf("report missing section %q", part)
		}
	}
}

func TestEnrichInspectionWithMockReportReplacesToolDump(t *testing.T) {
	res := InspectionResult{
		Domain:          DomainHadoop,
		TriggerType:     "scenario_runner",
		SourceKind:      "cron",
		ReportMarkdown:  "#### [query_vm_metrics] Failed\nconnection refused",
		ScoreSource:     "none",
	}
	EnrichInspectionWithMockReport(&res, "ops-bch-health", "")

	if strings.HasPrefix(res.ReportMarkdown, "#### [") {
		t.Fatalf("expected polished report, got tool dump: %s", res.ReportMarkdown[:80])
	}
	if res.Score == nil || *res.Score != 86 {
		t.Fatalf("expected mock score 86, got %+v", res.Score)
	}
}
