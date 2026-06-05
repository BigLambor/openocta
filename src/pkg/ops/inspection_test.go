package ops

import "testing"

func TestParseInspectionResultWithExplicitContext(t *testing.T) {
	res := ParseInspectionResultWithContext(
		"session-1",
		"job-inspect-hadoop-deep",
		"健康得分: 82\n存在 YARN 队列压力。",
		"ok",
		1000,
		250,
		InspectionContext{Domain: DomainHadoop, ClusterID: "cluster-1", Component: "YARN", ScenarioKey: "ops-hadoop-health"},
	)

	if res.Domain != DomainHadoop || res.ClusterID != "cluster-1" || res.Component != "YARN" || res.ScenarioKey != "ops-hadoop-health" {
		t.Fatalf("context not preserved: %+v", res)
	}
	if res.Score == nil || *res.Score != 82 || res.ScoreStatus != "warning" {
		t.Fatalf("score not parsed correctly: %+v", res)
	}
	if res.ScoreSource != "legacy_text" {
		t.Fatalf("expected legacy_text score source, got %s", res.ScoreSource)
	}
	if res.StartedAt != 1000 || res.FinishedAt != 1250 {
		t.Fatalf("timestamps not preserved: %+v", res)
	}
}

func TestParseInspectionResultPrefersStructuredJSON(t *testing.T) {
	res := ParseInspectionResultWithContext(
		"session-structured",
		"job-inspect-gbase",
		"```json\n{\"domain\":\"gbase\",\"clusterId\":\"cluster-gbase-1\",\"score\":91,\"scoreStatus\":\"ok\",\"errors\":[],\"toolRuns\":[{\"toolName\":\"query_gbase_slow_sql\",\"success\":true,\"output\":\"[]\"}]}\n```",
		"ok",
		1000,
		250,
		InspectionContext{},
	)

	if res.Domain != DomainGBase || res.ClusterID != "cluster-gbase-1" {
		t.Fatalf("structured context not applied: %+v", res)
	}
	if res.Score == nil || *res.Score != 91 || res.ScoreStatus != "ok" {
		t.Fatalf("structured score not applied: %+v", res)
	}
	if res.ScoreSource != "structured" {
		t.Fatalf("expected structured score source, got %s", res.ScoreSource)
	}
	if res.ScenarioKey != "ops-gbase-health" {
		t.Fatalf("expected inferred GBase scenario key, got %s", res.ScenarioKey)
	}
	if len(res.ToolRuns) != 1 || res.ToolRuns[0].ToolName != "query_gbase_slow_sql" {
		t.Fatalf("structured tool runs not applied: %+v", res.ToolRuns)
	}
}

func TestParseInspectionContextLine(t *testing.T) {
	ctx := parseInspectionContextLine("[运维上下文] 业务域: BCH生态 | 目标: YARN ResourceManager | prod-a | cluster=cluster-1 | component=YARN+ResourceManager")

	if ctx.Domain != DomainHadoop || ctx.ClusterID != "cluster-1" || ctx.Component != "YARN ResourceManager" {
		t.Fatalf("unexpected context: %+v", ctx)
	}
}
