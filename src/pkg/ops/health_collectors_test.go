package ops

import "testing"

func TestAggregateGBaseMissingRequiredSourceIsDegraded(t *testing.T) {
	cluster := Cluster{
		ID:     "cluster-gbase-test",
		Name:   "GBase Test",
		Domain: DomainGBase,
		Status: "healthy",
	}
	policy := defaultDomainHealthPolicy(DomainGBase)
	signals := []HealthSignal{
		collectAssetStatusSignal(cluster),
	}

	snapshot := AggregateHealthSnapshot(cluster, policy, signals)
	if snapshot.Score != nil {
		t.Fatalf("expected no composite score, got %d", *snapshot.Score)
	}
	if snapshot.ScoreStatus != ScoreStatusDegraded {
		t.Fatalf("expected degraded, got %s", snapshot.ScoreStatus)
	}
	if !containsString(snapshot.MissingSources, SignalTypeGBaseSQL) {
		t.Fatalf("expected missing gbase_sql, got %v", snapshot.MissingSources)
	}
}

func TestCollectAlertSignalDeduplicatesAndScores(t *testing.T) {
	initTestAlertsStore(t)
	cluster := Cluster{ID: "cluster-gbase-alert", Domain: DomainGBase, Status: "healthy"}
	_, err := RecordMergedAlertGroup("gbase-monitor", "session", "run", []MergedAlertInput{
		{Title: "pool exhausted", Severity: "critical", Alertname: "PoolFull", Service: "GBase", Instance: "db-1", ClusterID: cluster.ID, Component: "pool"},
		{Title: "pool exhausted again", Severity: "warning", Alertname: "PoolFull", Service: "GBase", Instance: "db-1", ClusterID: cluster.ID, Component: "pool"},
	})
	if err != nil {
		t.Fatal(err)
	}

	signal, ok := collectAlertSignal(cluster)
	if !ok {
		t.Fatal("expected alert signal")
	}
	if signal.Score == nil || *signal.Score != 85 {
		t.Fatalf("expected score 85 after one critical fingerprint, got %v", signal.Score)
	}
	if signal.Status != HealthStatusCritical {
		t.Fatalf("expected critical status, got %s", signal.Status)
	}
	if signal.SourceKind != SourceKindCollector || signal.Source != "collector:alerts" {
		t.Fatalf("unexpected source: %s %s", signal.SourceKind, signal.Source)
	}
}

func TestNormalizeMetricsBaseURL(t *testing.T) {
	tests := map[string]string{
		"http://vm.example.com:8428/api/v1/query": "http://vm.example.com:8428",
		"http://vm.example.com:8428/api/v1":       "http://vm.example.com:8428",
		"http://vm.example.com:8428/":             "http://vm.example.com:8428",
	}
	for input, want := range tests {
		if got := normalizeMetricsBaseURL(input); got != want {
			t.Fatalf("normalizeMetricsBaseURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveClusterMetricsBaseURLFromEnvRef(t *testing.T) {
	t.Setenv("VM_TEST_URL", "http://vm.example.com:8428/api/v1/query")
	got := resolveClusterMetricsBaseURL(Cluster{VMUrlRef: "VM_TEST_URL"})
	if got != "http://vm.example.com:8428" {
		t.Fatalf("expected env ref to resolve and normalize, got %q", got)
	}
}

func containsString(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
