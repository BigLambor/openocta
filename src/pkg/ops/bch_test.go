package ops

import (
	"testing"
)

func TestMockBchService(t *testing.T) {
	svc := NewMockBchService()

	// 1. Clusters Health
	clusters, err := svc.GetClustersHealth()
	if err != nil {
		t.Fatal(err)
	}
	if len(clusters) != 2 {
		t.Fatalf("expected 2 clusters, got %d", len(clusters))
	}

	// 2. Flink Jobs
	flinkJobs, err := svc.ListFlinkJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(flinkJobs) != 6 {
		t.Fatalf("expected 6 flink jobs, got %d", len(flinkJobs))
	}

	// 3. Flink Job config & diagnose
	config, err := svc.GetFlinkJobConfig("job_tx_core")
	if err != nil {
		t.Fatal(err)
	}
	if config == "" {
		t.Fatal("expected non-empty config")
	}

	diagnose, err := svc.DiagnoseFlinkJob("job_tx_core")
	if err != nil {
		t.Fatal(err)
	}
	if diagnose.ID != "job_tx_core" {
		t.Fatalf("expected job_tx_core, got %s", diagnose.ID)
	}

	// 4. Spark Jobs
	sparkJobs, err := svc.ListSparkJobs()
	if err != nil {
		t.Fatal(err)
	}
	if len(sparkJobs) != 3 {
		t.Fatalf("expected 3 spark jobs, got %d", len(sparkJobs))
	}

	// 5. HDFS FSImage
	hdfsStats, err := svc.GetHdfsFsImage("NS1")
	if err != nil {
		t.Fatal(err)
	}
	if hdfsStats.Namespace != "NS1" {
		t.Fatalf("expected NS1, got %s", hdfsStats.Namespace)
	}

	// 6. Employees
	employees, err := svc.ListEmployees()
	if err != nil {
		t.Fatal(err)
	}
	if len(employees) != 3 {
		t.Fatalf("expected 3 employees, got %d", len(employees))
	}
}
