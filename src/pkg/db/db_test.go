package db

import "testing"

func TestInitDBRunsMigrations(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = CloseDB() })

	if err := InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	sqliteDB := GetDB()
	if sqliteDB == nil {
		t.Fatal("expected db handle")
	}

	for _, table := range []string{
		"schema_migrations",
		"sessions",
		"sessions_v1",
		"cron_jobs",
		"health_signals",
		"health_snapshots",
		"assets",
		"clusters",
		"asset_relations",
		"alert_groups",
		"alert_events",
		"incident_timeline",
		"tasks",
		"jobs",
		"job_runs",
		"run_steps",
		"tool_invocations",
		"model_usage",
		"approvals",
		"approval_steps",
		"approval_session_whitelist",
		"audit_logs",
		"secrets",
		"config_versions",
		"users",
		"roles",
		"permissions",
		"role_permissions",
		"user_tokens",
		"job_schedules",
	} {
		var count int
		if err := sqliteDB.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count); err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected table %s to exist", table)
		}
	}

	var applied int
	if err := sqliteDB.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version IN (1, 2, 3, 4, 5)`).Scan(&applied); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if applied != 5 {
		t.Fatalf("expected migration versions 1 through 5 to be recorded, got %d", applied)
	}
}

func TestInitDBIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = CloseDB() })

	if err := InitDB(dir); err != nil {
		t.Fatalf("first InitDB: %v", err)
	}
	if _, err := GetDB().Exec(`INSERT INTO cron_jobs (id, detail_json) VALUES (?, ?)`, "job-test", "{}"); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	if err := InitDB(dir); err != nil {
		t.Fatalf("second InitDB: %v", err)
	}

	var detail string
	if err := GetDB().QueryRow(`SELECT detail_json FROM cron_jobs WHERE id = ?`, "job-test").Scan(&detail); err != nil {
		t.Fatalf("query fixture after second init: %v", err)
	}
	if detail != "{}" {
		t.Fatalf("unexpected detail after second init: %q", detail)
	}
}

func TestRunMigrationsRejectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	t.Cleanup(func() { _ = CloseDB() })

	if err := InitDB(dir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if _, err := GetDB().Exec(`UPDATE schema_migrations SET checksum = ? WHERE version = 1`, "bad-checksum"); err != nil {
		t.Fatalf("tamper checksum: %v", err)
	}
	if err := InitDB(dir); err == nil {
		t.Fatal("expected InitDB to reject checksum mismatch")
	}
	if GetDB() != nil {
		t.Fatal("expected db handle to be reset after failed migration")
	}
}
