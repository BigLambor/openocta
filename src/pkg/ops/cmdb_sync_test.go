package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestSyncClustersFromCMDBBody(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}

	rows := []map[string]interface{}{
		{
			"name":       "CMDB-A",
			"domain":     DomainHadoop,
			"nodeCount":  5,
			"components": json.RawMessage(`"HDFS,YARN"`),
			"status":        "healthy",
			"monitorLabels": `job="hadoop-prod"`,
		},
	}
	res, err := SyncClustersFromCMDB(context.Background(), rows, "upsert", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 1 || res.Updated != 0 {
		t.Fatalf("first sync: %+v", res)
	}

	rows[0]["nodeCount"] = 9
	rows[0]["status"] = "warning"
	res2, err := SyncClustersFromCMDB(context.Background(), rows, "upsert", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Updated != 1 {
		t.Fatalf("second sync: %+v", res2)
	}
	got, err := GetCluster(listFirstClusterID(t))
	if err != nil || got.NodeCount != 9 || got.Status != "warning" {
		t.Fatalf("cluster: %+v err=%v", got, err)
	}
}

func TestSyncClustersFromCMDBWebhook(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"clusters": []map[string]interface{}{
				{"name": "Webhook-B", "domain": DomainFI, "nodeCount": 3, "status": "unknown", "monitorLabels": `job="fi-prod"`},
			},
		})
	}))
	defer srv.Close()

	t.Setenv("OPS_CMDB_SYNC_URL", srv.URL)
	res, err := SyncClustersFromCMDB(context.Background(), nil, "upsert", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Source != "webhook" || res.Created != 1 {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestSyncClustersFromCMDBMissingConfig(t *testing.T) {
	os.Unsetenv("OPS_CMDB_SYNC_URL")
	_, err := SyncClustersFromCMDB(context.Background(), nil, "upsert", nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestSyncClustersMappingAndStrategy(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}

	// 1. Test Field Mapping
	customMapping := CMDBMapping{
		Name:        "ext_cluster_name",
		Domain:      "ext_domain",
		Region:      "ext_region",
		NodeCount:   "ext_node_count",
		Components:  "ext_components",
		Owner:       "ext_owner",
		Status:      "ext_status",
		Description: "ext_desc",
	}

	rows := []map[string]interface{}{
		{
			"ext_cluster_name": "Mapped-Cluster-1",
			"ext_domain":       DomainGBase,
			"ext_region":       "shenzhen",
			"ext_node_count":   12,
			"ext_components":   []string{"GBase-Server"},
			"ext_owner":        "Bob",
			"ext_status":       "healthy",
			"ext_desc":         "mapped correctly",
			"monitorLabels":    `job="gbase-prod"`,
		},
	}

	res, err := SyncClustersFromCMDB(context.Background(), rows, "upsert", &customMapping)
	if err != nil {
		t.Fatal(err)
	}
	if res.Created != 1 || len(res.Errors) > 0 {
		t.Fatalf("mapped sync failed: %+v", res)
	}

	// Verify Mapped Fields
	list, err := ListClusters(DomainGBase)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 GBase cluster, got %v", list)
	}
	c := list[0]
	if c.Name != "Mapped-Cluster-1" || c.Region != "shenzhen" || c.NodeCount != 12 || c.Owner != "Bob" {
		t.Fatalf("fields not correctly mapped in cluster: %+v", c)
	}

	// 2. Test "mark-inactive" Strategy
	// We import a new feed which doesn't contain "Mapped-Cluster-1" but contains "Mapped-Cluster-2"
	rows2 := []map[string]interface{}{
		{
			"ext_cluster_name": "Mapped-Cluster-2",
			"ext_domain":       DomainGBase,
			"ext_status":       "healthy",
			"monitorLabels":    `job="gbase-prod"`,
		},
	}

	res2, err := SyncClustersFromCMDB(context.Background(), rows2, "mark-inactive", &customMapping)
	if err != nil {
		t.Fatal(err)
	}
	if res2.Created != 1 {
		t.Fatalf("expected 1 created: %+v", res2)
	}

	// Mapped-Cluster-1 should now be "inactive"
	c1, err := GetCluster(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if c1.Status != "inactive" {
		t.Fatalf("expected Mapped-Cluster-1 to be marked inactive, got status: %s", c1.Status)
	}

	// 3. Test "delete" Strategy
	// We import an empty feed with "delete" strategy. It should delete both Mapped-Cluster-1 and Mapped-Cluster-2
	// Wait, empty rawClusters calls URL sync, so we pass a slice with 1 invalid item to test error reporting, or just mock it.
	rows3 := []map[string]interface{}{}
	t.Setenv("OPS_CMDB_SYNC_URL", "http://disabled") // just to bypass check if we call it with empty rawClusters, but we can pass 1 valid item
	rows3 = append(rows3, map[string]interface{}{
		"ext_cluster_name": "Mapped-Cluster-3",
		"ext_domain":       DomainGBase,
		"ext_status":       "healthy",
		"monitorLabels":    `job="gbase-prod"`,
	})
	res3, err := SyncClustersFromCMDB(context.Background(), rows3, "delete", &customMapping)
	if err != nil {
		t.Fatal(err)
	}
	if res3.Created != 1 {
		t.Fatal("expected Mapped-Cluster-3 to be created")
	}

	// Mapped-Cluster-2 should be deleted now
	allGBase, _ := ListClusters(DomainGBase)
	for _, cls := range allGBase {
		if cls.Name == "Mapped-Cluster-2" {
			t.Fatalf("expected Mapped-Cluster-2 to be deleted")
		}
	}

	// 4. Test Sync Errors reporting
	invalidRows := []map[string]interface{}{
		{
			"ext_cluster_name": "", // empty name - should fail
			"ext_domain":       DomainGBase,
		},
		{
			"ext_cluster_name": "Cluster-With-Invalid-Domain",
			"ext_domain":       "nonexistent-domain", // invalid domain
		},
	}
	res4, err := SyncClustersFromCMDB(context.Background(), invalidRows, "dry-run", &customMapping)
	if err != nil {
		t.Fatal(err)
	}
	if res4.Skipped != 2 || len(res4.Errors) != 2 {
		t.Fatalf("expected 2 skipped errors, got %+v", res4)
	}
	if res4.Errors[0].Name != "" || res4.Errors[1].Name != "Cluster-With-Invalid-Domain" {
		t.Fatalf("error report structure is incorrect: %+v", res4.Errors)
	}
}

func TestSyncClustersFromCMDBDryRunDoesNotMutateStore(t *testing.T) {
	dir := t.TempDir()
	if err := InitStore(dir); err != nil {
		t.Fatal(err)
	}

	rows := []map[string]interface{}{
		{
			"name":      "Preview-Only",
			"domain":    DomainHadoop,
			"nodeCount": 5,
			"status":        "healthy",
			"monitorLabels": `job="hadoop-prod"`,
		},
	}
	res, err := SyncClustersFromCMDB(context.Background(), rows, "dry-run", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.DryRun || res.Strategy != "dry-run" || res.Created != 1 {
		t.Fatalf("unexpected dry-run result: %+v", res)
	}
	list, err := ListClusters("")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("dry-run must not write clusters, got %+v", list)
	}
}

func listFirstClusterID(t *testing.T) string {
	t.Helper()
	list, err := ListClusters("")
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	return list[0].ID
}
