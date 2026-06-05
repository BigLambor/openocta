package tools

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGBaseSlowSqlToolReturnsStructuredEvidenceWhenDSNMissing(t *testing.T) {
	t.Setenv("GBASE_DSN", "")
	res, err := GBaseSlowSqlTool{}.Execute(context.Background(), map[string]interface{}{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Success {
		t.Fatal("expected failure without DSN")
	}
	var out gbaseSlowSQLEvidence
	if err := json.Unmarshal([]byte(res.Output), &out); err != nil {
		t.Fatalf("expected structured json output, got %q: %v", res.Output, err)
	}
	if out.Type != "gbase_sql" || out.Status != "critical" || out.Error == "" {
		t.Fatalf("unexpected evidence: %+v", out)
	}
	if _, ok := res.Data.(gbaseSlowSQLEvidence); !ok {
		t.Fatalf("expected structured Data, got %T", res.Data)
	}
}
