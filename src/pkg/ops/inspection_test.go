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
		InspectionContext{Domain: DomainHadoop, ClusterID: "cluster-1", Component: "YARN"},
	)

	if res.Domain != DomainHadoop || res.ClusterID != "cluster-1" || res.Component != "YARN" {
		t.Fatalf("context not preserved: %+v", res)
	}
	if res.Score == nil || *res.Score != 82 || res.ScoreStatus != "warning" {
		t.Fatalf("score not parsed correctly: %+v", res)
	}
	if res.StartedAt != 1000 || res.FinishedAt != 1250 {
		t.Fatalf("timestamps not preserved: %+v", res)
	}
}

func TestParseInspectionContextLine(t *testing.T) {
	ctx := parseInspectionContextLine("[运维上下文] 业务域: BCH生态 | 目标: YARN ResourceManager | prod-a | cluster=cluster-1 | component=YARN+ResourceManager")

	if ctx.Domain != DomainHadoop || ctx.ClusterID != "cluster-1" || ctx.Component != "YARN ResourceManager" {
		t.Fatalf("unexpected context: %+v", ctx)
	}
}
