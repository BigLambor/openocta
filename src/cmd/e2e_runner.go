package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/openocta/openocta/pkg/ops"
)

func main() {
	// Initialize L3 Health Store in a temporary directory
	tmpDir, err := os.MkdirTemp("", "openocta-e2e-")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Initialized Temp State Dir: %s\n", tmpDir)
	if err := ops.InitHealthStore(tmpDir); err != nil {
		panic(err)
	}

	// 1. Run the Scenario
	fmt.Println("\n--- 1. Running ops-gbase-health Scenario ---")
	ctx := context.Background()
	os.Setenv("GBASE_DSN", "mock://test") // Trigger mock success
	opts := ops.RunOpts{
		SessionID:  "e2e-session-123",
		RunID:      "e2e-run-456",
		EmployeeID: "emp_e2e_test",
		Params: map[string]interface{}{
			"query": "sum(gbase_active_connections)",
		},
	}

	// For the cluster to exist in DB, we mock it first.
	// Cluster facts doesn't have a CreateCluster, but we can use an existing mock ID.
	// Wait, is there a mock cluster ID in GetCluster? 
	// The cluster list in `cluster_facts.go` has "cluster-prod-a" and "cluster-prod-b"
	clusterID := "cluster-prod-a"
	
	// Pre-register a mock cluster in ops if necessary, or just run. 
	// Wait, PersistInspectionFacts looks up the cluster by ID: `cluster, err := GetCluster(report.ClusterID)`.
	// If it doesn't exist, it might skip writing signals. Let's see if we can register a mock cluster.
	// We'll just run it and observe the InspectionResult for now.
	res, err := ops.RunScenario(ctx, "ops-gbase-health", clusterID, opts)
	if err != nil {
		fmt.Printf("RunScenario Error: %v\n", err)
	} else {
		b, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("RunScenario Result:\n%s\n", string(b))
	}

	// 2. Read back from L3 Facts
	fmt.Println("\n--- 2. L3 Facts Verification ---")
	snapshots, _ := ops.ListHealthSnapshots()
	if len(snapshots) > 0 {
		b, _ := json.MarshalIndent(snapshots, "", "  ")
		fmt.Printf("HealthSnapshots:\n%s\n", string(b))
	} else {
		fmt.Println("No HealthSnapshots found. (Note: this is expected if cluster-gbase-demo is not in the Cluster Store)")
	}

	signals, _ := ops.ListHealthSignals()
	if len(signals) > 0 {
		b, _ := json.MarshalIndent(signals, "", "  ")
		fmt.Printf("HealthSignals:\n%s\n", string(b))
	} else {
		fmt.Println("No HealthSignals found.")
	}

	fmt.Println("\n--- 3. E2E Complete ---")
}
