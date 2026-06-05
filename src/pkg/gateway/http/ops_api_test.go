package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/rbac"
)

func TestOpsAPIPermissionIsolation(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	// Ensure DB & Store Init
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := ops.InitStore(tempDir); err != nil {
		t.Fatalf("InitStore: %v", err)
	}
	if err := ops.InitAlertsStore(tempDir); err != nil {
		t.Fatalf("InitAlertsStore: %v", err)
	}

	// Clean any preseeded alerts / clusters for test cleanliness
	_ = os.Remove(filepath.Join(tempDir, "ops", "clusters.json"))
	_ = os.Remove(filepath.Join(tempDir, "ops", "alerts.json"))
	if err := ops.InitStore(tempDir); err != nil {
		t.Fatalf("Re-init InitStore: %v", err)
	}
	if err := ops.InitAlertsStore(tempDir); err != nil {
		t.Fatalf("Re-init InitAlertsStore: %v", err)
	}

	// Seed test clusters
	c1, err := ops.CreateCluster(ops.ClusterCreate{
		Name:      "Hadoop Prod",
		Domain:    "hadoop",
		Region:    "region-1",
		NodeCount: 100,
		Status:    "healthy",
	})
	if err != nil {
		t.Fatalf("Create Hadoop cluster: %v", err)
	}
	c2, err := ops.CreateCluster(ops.ClusterCreate{
		Name:      "GBase Prod",
		Domain:    "gbase",
		Region:    "region-1",
		NodeCount: 10,
		Status:    "healthy",
	})
	if err != nil {
		t.Fatalf("Create GBase cluster: %v", err)
	}

	// Seed test alert groups
	_, err = ops.RecordMergedAlertGroup("hadoop-prod", "session-hadoop", "run-hadoop", []ops.MergedAlertInput{
		{AlertID: "a1", Title: "Hadoop Alert", Severity: "critical", Alertname: "HadoopDown", Service: "hdfs", ClusterID: c1.ID},
	})
	if err != nil {
		t.Fatalf("Record Hadoop alert: %v", err)
	}
	_, err = ops.RecordMergedAlertGroup("gbase-prod", "session-gbase", "run-gbase", []ops.MergedAlertInput{
		{AlertID: "a2", Title: "GBase Alert", Severity: "critical", Alertname: "GBaseDown", Service: "gbase", ClusterID: c2.ID},
	})
	if err != nil {
		t.Fatalf("Record GBase alert: %v", err)
	}

	// Authenticate users to get tokens
	adminToken, err := rbac.AuthenticateUser("admin", "admin888")
	if err != nil {
		t.Fatalf("Auth admin: %v", err)
	}
	gbaseToken, err := rbac.AuthenticateUser("gbase_op", "op123456")
	if err != nil {
		t.Fatalf("Auth gbase_op: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := &http.Client{}

	// Test 1: GET /api/ops/clusters as Admin
	req, _ := http.NewRequest("GET", ts.URL+"/api/ops/clusters", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Admin list clusters status: %d", resp.StatusCode)
	}
	var resClusters struct {
		Clusters []ops.Cluster `json:"clusters"`
		Total    int           `json:"total"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&resClusters)
	resp.Body.Close()
	if resClusters.Total != 2 {
		t.Errorf("Admin should see 2 clusters, got %d", resClusters.Total)
	}

	// Test 2: GET /api/ops/clusters as GBase Operator
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/clusters", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GBase op list clusters status: %d", resp.StatusCode)
	}
	_ = json.NewDecoder(resp.Body).Decode(&resClusters)
	resp.Body.Close()
	if resClusters.Total != 1 || resClusters.Clusters[0].Domain != "gbase" {
		t.Errorf("GBase op should only see 1 GBase cluster, got %v", resClusters.Clusters)
	}

	// Test 3: GET /api/ops/clusters?domain=hadoop as GBase Operator (should be 403)
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/clusters?domain=hadoop", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("GBase op requesting hadoop domain should be 403, got %d", resp.StatusCode)
	}

	// Test 4: GET /api/ops/alerts/groups as GBase Operator
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/alerts/groups", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GBase op list alerts status: %d", resp.StatusCode)
	}
	var resAlerts ops.AlertGroupsListResponse
	_ = json.NewDecoder(resp.Body).Decode(&resAlerts)
	resp.Body.Close()
	if resAlerts.Total != 1 || resAlerts.Groups[0].Domain != "gbase" {
		t.Errorf("GBase op should only see 1 GBase alert, got %d groups: %v", resAlerts.Total, resAlerts.Groups)
	}

	// Test 5: GET /api/ops/dashboard/summary as GBase Operator
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/dashboard/summary", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GBase op dashboard status: %d", resp.StatusCode)
	}
	var resSummary ops.DashboardSummary
	_ = json.NewDecoder(resp.Body).Decode(&resSummary)
	resp.Body.Close()

	if resSummary.TotalClusters != 1 {
		t.Errorf("GBase op dashboard total clusters should be 1, got %d", resSummary.TotalClusters)
	}
	if len(resSummary.Domains) != 1 || resSummary.Domains[0].Domain != "gbase" {
		t.Errorf("GBase op dashboard should only roll up gbase, got %v", resSummary.Domains)
	}

	// Test 6: Direct access to BCH clusters health as GBase Operator (should be 403)
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/clusters/health", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("GBase op accessing BCH API should be 403, got %d", resp.StatusCode)
	}

	// Test 7: Direct access to BCH clusters health as Admin (should be 200)
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/clusters/health", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Admin accessing BCH API should be 200, got %d", resp.StatusCode)
	}
}

func TestBCHAPIPermissionEnforcement(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}

	// Seed hadoop operator
	// Role 2 is hadoop_operator, seed hadoop user
	salt := "test_salt"
	opHash := rbac.HashPassword("op123456", salt)
	db := rbac.GetDB()
	if db != nil {
		_, _ = db.Exec(`INSERT INTO users (username, password_hash, salt, role_id) VALUES (?, ?, ?, ?)`, "hadoop_op", opHash, salt, 2)
		// Bind permissions
		_, _ = db.Exec(`INSERT OR IGNORE INTO role_permissions (role_id, permission_code) VALUES (?, ?)`, 2, "menu:hadoop")
	}

	hadoopToken, err := rbac.AuthenticateUser("hadoop_op", "op123456")
	if err != nil {
		t.Fatalf("Auth hadoop_op: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	client := &http.Client{}

	// Test 1: Hadoop operator gets BCH clusters health (should succeed)
	req, _ := http.NewRequest("GET", ts.URL+"/api/ops/bch/clusters/health", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Hadoop operator accessing BCH API should succeed, got %d", resp.StatusCode)
	}

	// Test 2: GBase operator gets BCH clusters health (should be 403)
	gbaseToken, err := rbac.AuthenticateUser("gbase_op", "op123456")
	if err != nil {
		t.Fatalf("Auth gbase_op: %v", err)
	}

	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/clusters/health", nil)
	req.Header.Set("Authorization", "Bearer "+gbaseToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("GBase operator accessing BCH API should fail with 403, got %d", resp.StatusCode)
	}
}
