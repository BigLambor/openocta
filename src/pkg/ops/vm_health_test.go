package ops

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestScoreFromSamples(t *testing.T) {
	tests := []struct {
		in   []float64
		want int
		ok   bool
	}{
		{[]float64{0.92}, 92, true},
		{[]float64{0.5, 1}, 75, true},
		{[]float64{85}, 85, true},
		{nil, 0, false},
	}
	for _, tc := range tests {
		got, ok := scoreFromSamples(tc.in)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Fatalf("scoreFromSamples(%v) = %d, %v; want %d, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestDomainHealthScoreFromVM(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/query" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"0.88"]}]}}`))
	}))
	defer srv.Close()

	t.Setenv(envSeedDemoData, "")
	initTestOpsStore(t)
	if _, err := CreateCluster(ClusterCreate{
		Name:           "hadoop-vm-test",
		Domain:         DomainHadoop,
		Status:         "healthy",
		MonitorLabels:  `job="hadoop-prod"`,
		MetricsBaseUrl: srv.URL,
	}); err != nil {
		t.Fatal(err)
	}

	t.Setenv("VICTORIAMETRICS_URL", srv.URL)
	client := newVMClient()
	score, note := domainHealthScore(context.Background(), client, DomainHadoop)
	if score == nil || *score != 88 {
		t.Fatalf("expected score 88, got %v note=%q", score, note)
	}
	if note != "" {
		t.Fatalf("unexpected note: %s", note)
	}
}

func TestEnrichDashboardVMHealthUnconfigured(t *testing.T) {
	os.Unsetenv("VICTORIAMETRICS_URL")
	os.Unsetenv("PROMETHEUS_URL")

	summary := DashboardSummary{
		Domains: []DomainHealthSummary{
			{Domain: DomainHadoop, ClusterCount: 1},
		},
	}
	enrichDashboardVMHealth(context.Background(), &summary)
	if summary.VMConfigured {
		t.Fatal("expected vmConfigured false")
	}
	if summary.Domains[0].HealthScoreNote == "" {
		t.Fatal("expected health score note when VM not configured")
	}
}
