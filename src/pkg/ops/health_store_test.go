package ops

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openocta/openocta/pkg/db"
)

func TestHealthJSONStore(t *testing.T) {
	// Ensure DB is closed to use JSON mode
	_ = db.CloseDB()

	tempDir := t.TempDir()

	err := InitHealthStore(tempDir)
	if err != nil {
		t.Fatalf("InitHealthStore failed: %v", err)
	}

	// 1. Add some signals
	scoreVal := 85
	signal := HealthSignal{
		ObjectType: "cluster",
		ObjectID:   "hadoop-1",
		Domain:     "hadoop",
		Type:       "cpu",
		Source:     "vm",
		Status:     "ok",
		Score:      &scoreVal,
	}

	err = healthStore.UpsertSignals([]HealthSignal{signal})
	if err != nil {
		t.Fatalf("UpsertSignals failed: %v", err)
	}

	// 2. List signals
	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatalf("ListHealthSignals failed: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].ObjectID != "hadoop-1" {
		t.Errorf("expected ObjectID 'hadoop-1', got %q", signals[0].ObjectID)
	}

	// 3. Upsert snapshot
	snapshot := HealthSnapshot{
		ObjectType:  "cluster",
		ObjectID:    "hadoop-1",
		Domain:      "hadoop",
		Score:       &scoreVal,
		ScoreStatus: ScoreStatusOK,
	}

	err = healthStore.UpsertSnapshots([]HealthSnapshot{snapshot})
	if err != nil {
		t.Fatalf("UpsertSnapshots failed: %v", err)
	}

	// 4. Get snapshot
	snap, ok := GetHealthSnapshot("hadoop-1")
	if !ok {
		t.Fatalf("GetHealthSnapshot failed")
	}
	if *snap.Score != 85 {
		t.Errorf("expected score 85, got %d", *snap.Score)
	}

	// 5. Aggregate domain snapshot
	domSnap, err := AggregateDomainSnapshot("hadoop")
	if err != nil {
		t.Fatalf("AggregateDomainSnapshot failed: %v", err)
	}
	if domSnap.TotalClusters != 1 || domSnap.HealthyClusters != 1 {
		t.Errorf("unexpected domain aggregation: %+v", domSnap)
	}
}

func TestHealthSQLiteStore(t *testing.T) {
	// Initialize DB
	tempDir := t.TempDir()
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		_ = db.CloseDB()
	}()

	err := InitHealthStore(tempDir)
	if err != nil {
		t.Fatalf("InitHealthStore failed: %v", err)
	}

	// 1. Add some signals
	scoreVal := 90
	signal := HealthSignal{
		ObjectType: "cluster",
		ObjectID:   "fi-1",
		Domain:     "fi",
		Type:       "hbase",
		Source:     "vm",
		Status:     "ok",
		Score:      &scoreVal,
	}

	err = healthStore.UpsertSignals([]HealthSignal{signal})
	if err != nil {
		t.Fatalf("UpsertSignals failed: %v", err)
	}

	// 2. List signals
	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatalf("ListHealthSignals failed: %v", err)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 signal, got %d", len(signals))
	}
	if signals[0].ObjectID != "fi-1" {
		t.Errorf("expected ObjectID 'fi-1', got %q", signals[0].ObjectID)
	}

	// 3. Upsert snapshot
	snapshot := HealthSnapshot{
		ObjectType:  "cluster",
		ObjectID:    "fi-1",
		Domain:      "fi",
		Score:       &scoreVal,
		ScoreStatus: ScoreStatusOK,
	}

	err = healthStore.UpsertSnapshots([]HealthSnapshot{snapshot})
	if err != nil {
		t.Fatalf("UpsertSnapshots failed: %v", err)
	}

	// 4. Get snapshot
	snap, ok := GetHealthSnapshot("fi-1")
	if !ok {
		t.Fatalf("GetHealthSnapshot failed")
	}
	if *snap.Score != 90 {
		t.Errorf("expected score 90, got %d", *snap.Score)
	}

	// 5. Aggregate domain snapshot
	domSnap, err := AggregateDomainSnapshot("fi")
	if err != nil {
		t.Fatalf("AggregateDomainSnapshot failed: %v", err)
	}
	if domSnap.TotalClusters != 1 || domSnap.HealthyClusters != 1 {
		t.Errorf("unexpected domain aggregation: %+v", domSnap)
	}
}

func TestHealthMigration(t *testing.T) {
	// First run in JSON mode
	_ = db.CloseDB()

	tempDir := t.TempDir()
	signalsPath := filepath.Join(tempDir, "ops", "health_signals.json")
	snapshotsPath := filepath.Join(tempDir, "ops", "health_snapshots.json")

	err := InitHealthStore(tempDir)
	if err != nil {
		t.Fatalf("InitHealthStore failed: %v", err)
	}

	// Add data
	scoreVal := 75
	signal := HealthSignal{
		ObjectType: "cluster",
		ObjectID:   "gbase-1",
		Domain:     "gbase",
		Type:       "connections",
		Source:     "vm",
		Status:     "ok",
		Score:      &scoreVal,
	}
	err = healthStore.UpsertSignals([]HealthSignal{signal})
	if err != nil {
		t.Fatalf("UpsertSignals: %v", err)
	}

	snapshot := HealthSnapshot{
		ObjectType:  "cluster",
		ObjectID:    "gbase-1",
		Domain:      "gbase",
		Score:       &scoreVal,
		ScoreStatus: ScoreStatusOK,
	}
	err = healthStore.UpsertSnapshots([]HealthSnapshot{snapshot})
	if err != nil {
		t.Fatalf("UpsertSnapshots: %v", err)
	}

	// Verify JSON files exist
	if _, err := os.Stat(signalsPath); err != nil {
		t.Fatalf("signals JSON not found: %v", err)
	}
	if _, err := os.Stat(snapshotsPath); err != nil {
		t.Fatalf("snapshots JSON not found: %v", err)
	}

	// Now switch to SQLite mode by initializing DB
	if err := db.InitDB(tempDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer func() {
		_ = db.CloseDB()
	}()

	err = InitHealthStore(tempDir)
	if err != nil {
		t.Fatalf("InitHealthStore SQLite: %v", err)
	}

	// Verify data migrated to SQLite
	signals, err := ListHealthSignals()
	if err != nil {
		t.Fatalf("ListHealthSignals: %v", err)
	}
	if len(signals) != 1 || signals[0].ObjectID != "gbase-1" {
		t.Errorf("migrated signals mismatch: %+v", signals)
	}

	snap, ok := GetHealthSnapshot("gbase-1")
	if !ok {
		t.Fatalf("migrated snapshot not found")
	}
	if *snap.Score != 75 {
		t.Errorf("migrated snapshot score mismatch: %d", *snap.Score)
	}

	// Verify old files renamed to .bak
	if _, err := os.Stat(signalsPath); err == nil {
		t.Errorf("old signals JSON should have been renamed/deleted")
	}
	if _, err := os.Stat(signalsPath + ".bak"); err != nil {
		t.Errorf("signals bak file not found: %v", err)
	}
	if _, err := os.Stat(snapshotsPath); err == nil {
		t.Errorf("old snapshots JSON should have been renamed/deleted")
	}
	if _, err := os.Stat(snapshotsPath + ".bak"); err != nil {
		t.Errorf("snapshots bak file not found: %v", err)
	}
}
