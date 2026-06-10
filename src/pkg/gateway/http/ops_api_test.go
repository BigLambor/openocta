package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
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
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("openocta InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
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
		Name:          "Hadoop Prod",
		Domain:        "hadoop",
		Region:        "region-1",
		NodeCount:     100,
		Status:        "healthy",
		MonitorLabels: `job="hadoop-prod"`,
	})
	if err != nil {
		t.Fatalf("Create Hadoop cluster: %v", err)
	}
	c2, err := ops.CreateCluster(ops.ClusterCreate{
		Name:          "GBase Prod",
		Domain:        "gbase",
		Region:        "region-1",
		NodeCount:     10,
		Status:        "healthy",
		MonitorLabels: `job="gbase-prod"`,
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

	// Seed admin and operator users for permission tests
	if _, err := rbac.SetupInitialAdmin("admin", "admin888!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}
	if err := rbac.CreateUser("gbase_op", "op123456", 4); err != nil {
		t.Fatalf("CreateUser gbase_op: %v", err)
	}

	// Authenticate users to get tokens
	adminToken, err := rbac.AuthenticateUser("admin", "admin888!")
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

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}

	if _, err := rbac.SetupInitialAdmin("admin", "admin888!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	// Seed hadoop operator
	if err := rbac.EnsureRolePermission(2, "menu:hadoop"); err != nil {
		t.Fatalf("EnsureRolePermission: %v", err)
	}
	if err := rbac.CreateUser("hadoop_op", "op123456", 2); err != nil {
		t.Fatalf("CreateUser hadoop_op: %v", err)
	}
	if err := rbac.CreateUser("gbase_op", "op123456", 4); err != nil {
		t.Fatalf("CreateUser gbase_op: %v", err)
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

	// Test 3: List YARN queues (should succeed, contains root.test)
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/yarn/queues", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Hadoop operator listing YARN queues should succeed, got %d", resp.StatusCode)
	}
	var queues []ops.YarnQueueEvaluation
	err = json.NewDecoder(resp.Body).Decode(&queues)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("Failed to decode YARN queues list: %v", err)
	}
	hasTestQueue := false
	var testQueue ops.YarnQueueEvaluation
	for _, q := range queues {
		if q.ID == "root.test" {
			hasTestQueue = true
			testQueue = q
			break
		}
	}
	if !hasTestQueue {
		t.Errorf("Expected YARN queues to contain root.test")
	}
	if testQueue.Status != "idle" {
		t.Errorf("Expected root.test status to initially be idle, got %s", testQueue.Status)
	}

	originalCapacity := testQueue.CurrentCapacity

	// Test 4: Execute root.test queue capacity change (should succeed)
	req, _ = http.NewRequest("POST", ts.URL+"/api/ops/bch/yarn/queues/root.test/execute", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	var executeRes map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&executeRes)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Hadoop operator executing YARN queue change should succeed, got %d, body: %v", resp.StatusCode, executeRes)
	}

	// Test 5: List YARN queues again, check root.test state is now healthy/reclaimed
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/yarn/queues", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(resp.Body).Decode(&queues)
	resp.Body.Close()
	for _, q := range queues {
		if q.ID == "root.test" {
			testQueue = q
			break
		}
	}
	if testQueue.Status != "healthy" {
		t.Errorf("Expected root.test status to be healthy after reclaim, got %s", testQueue.Status)
	}
	if testQueue.CurrentCapacity != testQueue.TargetCapacity {
		t.Errorf("Expected root.test currentCapacity to be updated to targetCapacity %v, got %v", testQueue.TargetCapacity, testQueue.CurrentCapacity)
	}

	// Test 6: Rollback root.test queue and verify baseline state is restored
	req, _ = http.NewRequest("POST", ts.URL+"/api/ops/bch/yarn/queues/root.test/rollback", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Hadoop operator rolling back YARN queue should succeed, got %d", resp.StatusCode)
	}
	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/bch/yarn/queues", nil)
	req.Header.Set("Authorization", "Bearer "+hadoopToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = json.NewDecoder(resp.Body).Decode(&queues)
	resp.Body.Close()
	for _, q := range queues {
		if q.ID == "root.test" {
			testQueue = q
			break
		}
	}
	if testQueue.Status != "idle" {
		t.Errorf("Expected root.test status to be idle after rollback, got %s", testQueue.Status)
	}
	if testQueue.CurrentCapacity != originalCapacity {
		t.Errorf("Expected root.test currentCapacity to rollback to %v, got %v", originalCapacity, testQueue.CurrentCapacity)
	}
}

func TestOpsJobRunsAPI(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("OPENOCTA_STATE_DIR", tempDir)
	t.Setenv("OPENOCTA_RUN_MODE", "service")
	t.Setenv("OPENOCTA_SKIP_CHANNELS", "1")
	t.Setenv("OPENOCTA_SKIP_CRON", "1")

	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })
	if err := rbac.InitDB(tempDir); err != nil {
		t.Fatalf("rbac InitDB: %v", err)
	}
	if err := jobrun.Init(); err != nil {
		t.Fatalf("jobrun.Init: %v", err)
	}

	if _, err := rbac.SetupInitialAdmin("admin", "admin888!"); err != nil {
		t.Fatalf("SetupInitialAdmin: %v", err)
	}

	svc := jobrun.Default()
	run, err := svc.Start(jobrun.StartInput{
		JobID:       "job-inspect-hadoop",
		TriggerType: jobrun.TriggerInspection,
		TriggerRef:  "ops-bch-health",
		Input:       map[string]interface{}{"domain": "hadoop"},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if _, err := svc.AddStep(run.ID, jobrun.StepInput{
		Kind:          "tool",
		Name:          "query_vm_metrics",
		Status:        jobrun.StatusSucceeded,
		OutputSummary: "ok",
	}); err != nil {
		t.Fatalf("AddStep: %v", err)
	}
	if err := svc.Succeed(run.ID, jobrun.FinishInput{Output: map[string]interface{}{"score": 90}}); err != nil {
		t.Fatalf("Succeed: %v", err)
	}

	adminToken, err := rbac.AuthenticateUser("admin", "admin888!")
	if err != nil {
		t.Fatalf("Auth admin: %v", err)
	}

	srv := NewServer(":0", "test-1.0.0")
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()
	client := &http.Client{}

	req, _ := http.NewRequest("GET", ts.URL+"/api/ops/job-runs?jobId=job-inspect-hadoop", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}
	var listResp struct {
		Runs  []jobrun.JobRun `json:"runs"`
		Total int             `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatal(err)
	}
	if listResp.Total != 1 || len(listResp.Runs) != 1 || listResp.Runs[0].ID != run.ID {
		t.Fatalf("unexpected list response: %+v", listResp)
	}

	req, _ = http.NewRequest("GET", ts.URL+"/api/ops/job-runs/"+run.ID, nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", resp.StatusCode)
	}
	var detail jobrun.RunDetail
	if err := json.NewDecoder(resp.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.Run.ID != run.ID || len(detail.Steps) != 1 || detail.Steps[0].Name != "query_vm_metrics" {
		t.Fatalf("unexpected detail: %+v", detail)
	}
}
