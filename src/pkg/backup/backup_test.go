package backup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/openocta/openocta/pkg/backup"
	openoctadb "github.com/openocta/openocta/pkg/db"
)

func TestBackupRestoreRoundTrip(t *testing.T) {
	srcDir := t.TempDir()
	restoreDir := t.TempDir()
	archivePath := filepath.Join(t.TempDir(), "openocta-backup.tar.gz")

	if err := openoctadb.InitDB(srcDir); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "openocta.json"), []byte(`{"gateway":{"port":18900}}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "sessions"), 0o750); err != nil {
		t.Fatalf("mkdir sessions: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sessions", "note.txt"), []byte("transcript-archive"), 0o600); err != nil {
		t.Fatalf("write session file: %v", err)
	}
	_ = openoctadb.CloseDB()

	manifest, err := backup.Create(backup.Options{
		StateDir:   srcDir,
		OutputPath: archivePath,
	})
	if err != nil {
		t.Fatalf("Create backup: %v", err)
	}
	if manifest.SchemaVersion <= 0 {
		t.Fatalf("expected schema version in manifest, got %+v", manifest)
	}
	if _, err := backup.VerifyArchive(archivePath); err != nil {
		t.Fatalf("VerifyArchive: %v", err)
	}

	restored, err := backup.Restore(backup.RestoreOptions{
		ArchivePath: archivePath,
		StateDir:    restoreDir,
		Force:       true,
	})
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if restored.SchemaVersion != manifest.SchemaVersion {
		t.Fatalf("schema version mismatch: backup=%d restore=%d", manifest.SchemaVersion, restored.SchemaVersion)
	}
	if _, err := os.Stat(filepath.Join(restoreDir, "openocta.db")); err != nil {
		t.Fatalf("restored db missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(restoreDir, "sessions", "note.txt")); err != nil {
		t.Fatalf("restored attachment missing: %v", err)
	}
	if err := openoctadb.InitDB(restoreDir); err != nil {
		t.Fatalf("post-restore InitDB: %v", err)
	}
	_ = openoctadb.CloseDB()
}
