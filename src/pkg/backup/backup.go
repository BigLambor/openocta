package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	openoctadb "github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/version"

	_ "modernc.org/sqlite"
)

// DefaultAttachmentDirs are copied into the backup archive alongside openocta.db.
var DefaultAttachmentDirs = []string{
	"openocta.json",
	"sessions",
	"ops",
	"credentials",
	"workspace",
	"employees",
}

// Options controls backup creation.
type Options struct {
	StateDir        string
	OutputPath      string
	AttachmentPaths []string
	OpenOctaVersion string
}

// Create writes a gzip-compressed tar backup archive.
func Create(opts Options) (Manifest, error) {
	stateDir := strings.TrimSpace(opts.StateDir)
	if stateDir == "" {
		return Manifest{}, fmt.Errorf("stateDir is required")
	}
	outputPath := strings.TrimSpace(opts.OutputPath)
	if outputPath == "" {
		return Manifest{}, fmt.Errorf("output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outputPath), 0o750); err != nil {
		return Manifest{}, err
	}

	dbPath := filepath.Join(stateDir, "openocta.db")
	if _, err := os.Stat(dbPath); err != nil {
		return Manifest{}, fmt.Errorf("database not found at %s: %w", dbPath, err)
	}

	tmpDir, err := os.MkdirTemp("", "openocta-backup-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tmpDir)

	dbBackupPath := filepath.Join(tmpDir, "openocta.db")
	if err := backupSQLite(dbPath, dbBackupPath); err != nil {
		return Manifest{}, fmt.Errorf("sqlite backup: %w", err)
	}

	schemaVersion, err := readSchemaVersion(dbBackupPath)
	if err != nil {
		return Manifest{}, err
	}
	maxSchema, err := openoctadb.MaxEmbeddedMigrationVersion()
	if err != nil {
		return Manifest{}, err
	}

	ver := strings.TrimSpace(opts.OpenOctaVersion)
	if ver == "" {
		ver = version.Version
	}

	manifest := Manifest{
		FormatVersion:    FormatVersion,
		OpenOctaVersion:  ver,
		CreatedAt:        time.Now().UTC(),
		StateDir:         stateDir,
		SchemaVersion:    schemaVersion,
		SchemaVersionMax: maxSchema,
		Files:            []ManifestFile{},
	}

	stagingRoot := filepath.Join(tmpDir, "payload")
	if err := os.MkdirAll(stagingRoot, 0o750); err != nil {
		return Manifest{}, err
	}
	if err := copyFile(dbBackupPath, filepath.Join(stagingRoot, "openocta.db")); err != nil {
		return Manifest{}, err
	}

	attachments := opts.AttachmentPaths
	if len(attachments) == 0 {
		attachments = DefaultAttachmentDirs
	}
	for _, rel := range attachments {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		src := filepath.Join(stateDir, rel)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Manifest{}, err
		}
		dst := filepath.Join(stagingRoot, rel)
		if err := copyPath(src, dst); err != nil {
			return Manifest{}, fmt.Errorf("copy attachment %q: %w", rel, err)
		}
	}

	if err := filepath.Walk(stagingRoot, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(stagingRoot, path)
		if err != nil {
			return err
		}
		sum, size, err := fileSHA256(path)
		if err != nil {
			return err
		}
		manifest.Files = append(manifest.Files, ManifestFile{
			Path:   filepath.ToSlash(rel),
			SHA256: sum,
			Bytes:  size,
		})
		return nil
	}); err != nil {
		return Manifest{}, err
	}

	manifestBytes, err := encodeManifest(manifest)
	if err != nil {
		return Manifest{}, err
	}
	if err := os.WriteFile(filepath.Join(stagingRoot, "manifest.json"), manifestBytes, 0o600); err != nil {
		return Manifest{}, err
	}

	if err := writeTarGz(outputPath, stagingRoot); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func backupSQLite(srcPath, destPath string) error {
	if err := os.RemoveAll(destPath); err != nil {
		return err
	}
	dsn := srcPath + "?_pragma=busy_timeout(10000)&_txlock=immediate"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		return err
	}
	escaped := strings.ReplaceAll(filepath.Clean(destPath), `'`, `''`)
	_, err = db.Exec(`VACUUM INTO '` + escaped + `'`)
	return err
}

func readSchemaVersion(dbPath string) (int64, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(5000)&mode=ro")
	if err != nil {
		return 0, err
	}
	defer db.Close()
	var version int64
	err = db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&version)
	return version, err
}

func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o750)
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func fileSHA256(path string) (string, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), n, nil
}

func writeTarGz(outputPath, root string) error {
	out, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	return filepath.Walk(root, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
