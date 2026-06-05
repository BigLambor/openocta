package ops

import (
	"path/filepath"
	"testing"
)

func TestClusterCRUD(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}

	created, err := CreateCluster(ClusterCreate{
		Name:          "北京生产",
		Domain:        DomainHadoop,
		NodeCount:     10,
		Components:    []string{"HDFS", "YARN"},
		Status:        "healthy",
		Owner:         "ops-a",
		MonitorLabels: `job="hadoop-prod"`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected id")
	}

	list, err := ListClusters(DomainHadoop)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}

	patched, err := PatchCluster(created.ID, ClusterPatch{
		Status: strPtr("warning"),
	})
	if err != nil || patched.Status != "warning" {
		t.Fatalf("patch: %v status=%s", err, patched.Status)
	}

	if err := DeleteCluster(created.ID); err != nil {
		t.Fatal(err)
	}
	left, _ := ListClusters("")
	if len(left) != 0 {
		t.Fatalf("expected empty store, got %d", len(left))
	}

	// persistence round-trip
	storePath = filepath.Join(dir, "ops", "clusters.json")
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateCluster(ClusterCreate{Name: "x", Domain: DomainFI, Status: "unknown", MonitorLabels: `job="fi-prod"`}); err != nil {
		t.Fatal(err)
	}
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	list2, _ := ListClusters("")
	if len(list2) != 1 {
		t.Fatalf("reload expected 1 cluster, got %d", len(list2))
	}
}

func TestBuildDashboardSummary(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	_, _ = CreateCluster(ClusterCreate{Name: "a", Domain: DomainHadoop, Status: "healthy", MonitorLabels: `job="hadoop-prod"`})
	_, _ = CreateCluster(ClusterCreate{Name: "b", Domain: DomainHadoop, Status: "warning", MonitorLabels: `job="hadoop-prod"`})

	s := BuildDashboardSummary()
	if s.TotalClusters != 2 || s.HealthyClusters != 1 || s.WarningClusters != 1 {
		t.Fatalf("unexpected summary: %+v", s)
	}
	if len(s.Domains) != 5 {
		t.Fatalf("expected 5 domain slots, got %d", len(s.Domains))
	}
}
