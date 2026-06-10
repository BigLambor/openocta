package ops

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestClusterCRUD(t *testing.T) {
	dir := initTestOpsStore(t)

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
	initTestOpsStore(t)
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

func TestSeedDemoDataDefaultDisabled(t *testing.T) {
	t.Setenv(envSeedDemoData, "")
	dir := initTestOpsStore(t)
	clusters, err := ListClusters("")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 0 {
		t.Fatalf("expected no demo clusters by default, got %d", len(clusters))
	}

	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}
	alerts := ListAlertGroups("", "")
	if len(alerts.Groups) != 0 {
		t.Fatalf("expected no demo alerts by default, got %d", len(alerts.Groups))
	}
}

func TestSeedDemoDataEnabledFlag(t *testing.T) {
	t.Setenv(envSeedDemoData, "1")
	if !seedDemoDataEnabled() {
		t.Fatal("expected demo seed to be enabled")
	}
	t.Setenv(envSeedDemoData, "true")
	if !seedDemoDataEnabled() {
		t.Fatal("expected true to enable demo seed")
	}
	t.Setenv(envSeedDemoData, "0")
	if seedDemoDataEnabled() {
		t.Fatal("expected 0 to disable demo seed")
	}
	t.Setenv(envSeedDemoData, "")
	if seedDemoDataEnabled() {
		t.Fatal("expected unset env to disable demo seed")
	}
}

func TestSeedDemoDataExplicitlyEnabledWritesFixtures(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envSeedDemoData, "1")

	_ = db.CloseDB()
	if err := db.InitDB(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	clusters, err := ListClusters("")
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) == 0 {
		t.Fatal("expected demo clusters when seed env is enabled")
	}

	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}
	alerts := ListAlertGroups("", "")
	if len(alerts.Groups) == 0 {
		t.Fatal("expected demo alerts when seed env is enabled")
	}
}

func TestInitStoreImportsClustersJSONAndBacksUp(t *testing.T) {
	dir := t.TempDir()
	_ = db.CloseDB()
	if err := db.InitDB(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })

	jsonPath := filepath.Join(dir, "ops", "clusters.json")
	legacy := Cluster{
		ID:            "cluster-legacy",
		Name:          "Legacy Hadoop",
		Domain:        DomainHadoop,
		Status:        "healthy",
		NodeCount:     3,
		Components:    []string{"HDFS"},
		MonitorLabels: `job="hadoop-prod"`,
		CreatedAtMs:   nowMs(),
		UpdatedAtMs:   nowMs(),
	}
	if err := SaveStore(jsonPath, &storeFile{Version: 1, Clusters: []Cluster{legacy}}); err != nil {
		t.Fatal(err)
	}
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(jsonPath); !os.IsNotExist(err) {
		t.Fatalf("expected legacy clusters.json to be moved to backup, stat err=%v", err)
	}
	backups, err := filepath.Glob(jsonPath + ".bak.*")
	if err != nil || len(backups) != 1 {
		t.Fatalf("expected one backup, got %v err=%v", backups, err)
	}

	list, err := ListClusters("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != legacy.ID {
		t.Fatalf("expected imported legacy cluster, got %+v", list)
	}
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}
	list2, err := ListClusters("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list2) != 1 {
		t.Fatalf("expected repeated InitStore to remain idempotent, got %d", len(list2))
	}
}

func TestClusterRepositoryConcurrentCreates(t *testing.T) {
	initTestOpsStore(t)

	const total = 12
	var wg sync.WaitGroup
	errs := make(chan error, total)
	for i := 0; i < total; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := CreateCluster(ClusterCreate{
				Name:          "concurrent-" + string(rune('a'+i)),
				Domain:        DomainHadoop,
				Status:        "healthy",
				MonitorLabels: `job="hadoop-prod"`,
			})
			errs <- err
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent create failed: %v", err)
		}
	}
	list, err := ListClusters(DomainHadoop)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != total {
		t.Fatalf("expected %d clusters, got %d", total, len(list))
	}
}
