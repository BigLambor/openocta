package ops

import (
	"testing"

	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
)

func initTestOpsStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = db.CloseDB()
	if err := db.InitDB(dir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })
	if err := InitStore(dir); err != nil {
		t.Fatalf("InitStore: %v", err)
	}
	return dir
}

func initTestAlertsStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	_ = db.CloseDB()
	if err := db.InitDB(dir); err != nil {
		t.Fatalf("db.InitDB: %v", err)
	}
	t.Cleanup(func() { _ = db.CloseDB() })
	if err := InitAlertsStore(dir); err != nil {
		t.Fatalf("InitAlertsStore: %v", err)
	}
	if err := jobrun.Init(); err != nil {
		t.Fatalf("jobrun.Init: %v", err)
	}
	return dir
}

func initTestAlertJobRunStore(t *testing.T) string {
	t.Helper()
	return initTestAlertsStore(t)
}
