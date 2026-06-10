package backup

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

// RestoreOptions controls archive extraction and validation.
type RestoreOptions struct {
	ArchivePath string
	StateDir    string
	Force       bool
}

// Restore extracts a backup archive into stateDir after compatibility and checksum checks.
func Restore(opts RestoreOptions) (Manifest, error) {
	archivePath := strings.TrimSpace(opts.ArchivePath)
	if archivePath == "" {
		return Manifest{}, fmt.Errorf("archive path is required")
	}
	stateDir := strings.TrimSpace(opts.StateDir)
	if stateDir == "" {
		return Manifest{}, fmt.Errorf("stateDir is required")
	}

	tmpDir, err := os.MkdirTemp("", "openocta-restore-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(archivePath, tmpDir); err != nil {
		return Manifest{}, fmt.Errorf("extract archive: %w", err)
	}

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return Manifest{}, fmt.Errorf("manifest.json missing from archive: %w", err)
	}
	manifest, err := decodeManifest(manifestData)
	if err != nil {
		return Manifest{}, fmt.Errorf("invalid manifest: %w", err)
	}
	if manifest.FormatVersion != FormatVersion {
		return Manifest{}, fmt.Errorf("unsupported backup format version %d", manifest.FormatVersion)
	}

	maxEmbedded, err := openoctadb.MaxEmbeddedMigrationVersion()
	if err != nil {
		return Manifest{}, err
	}
	if manifest.SchemaVersion > maxEmbedded {
		return Manifest{}, fmt.Errorf(
			"backup schema version %d is newer than this binary supports (%d); upgrade OpenOcta first",
			manifest.SchemaVersion, maxEmbedded,
		)
	}

	for _, file := range manifest.Files {
		path := filepath.Join(tmpDir, filepath.FromSlash(file.Path))
		sum, size, err := fileSHA256(path)
		if err != nil {
			return Manifest{}, fmt.Errorf("verify %q: %w", file.Path, err)
		}
		if sum != file.SHA256 || size != file.Bytes {
			return Manifest{}, fmt.Errorf("checksum mismatch for %q", file.Path)
		}
	}

	dbPath := filepath.Join(stateDir, "openocta.db")
	if _, err := os.Stat(dbPath); err == nil && !opts.Force {
		return Manifest{}, fmt.Errorf("target database already exists at %s; pass --force to overwrite", dbPath)
	}

	if err := os.MkdirAll(stateDir, 0o750); err != nil {
		return Manifest{}, err
	}

	for _, file := range manifest.Files {
		src := filepath.Join(tmpDir, filepath.FromSlash(file.Path))
		dst := filepath.Join(stateDir, filepath.FromSlash(file.Path))
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return Manifest{}, err
		}
		if err := copyFile(src, dst); err != nil {
			return Manifest{}, fmt.Errorf("restore %q: %w", file.Path, err)
		}
	}

	// Re-open restored DB and ensure migrations can apply forward from backup schema version.
	if err := openoctadb.InitDB(stateDir); err != nil {
		return Manifest{}, fmt.Errorf("post-restore migration: %w", err)
	}
	_ = openoctadb.CloseDB()
	return manifest, nil
}

func extractTarGz(archivePath, destDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target := filepath.Join(destDir, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) &&
			filepath.Clean(target) != filepath.Clean(destDir) {
			return fmt.Errorf("invalid tar path: %q", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o750); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				_ = out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		default:
			continue
		}
	}
}

// VerifyArchive validates manifest checksums without restoring.
func VerifyArchive(archivePath string) (Manifest, error) {
	tmpDir, err := os.MkdirTemp("", "openocta-verify-*")
	if err != nil {
		return Manifest{}, err
	}
	defer os.RemoveAll(tmpDir)
	if err := extractTarGz(archivePath, tmpDir); err != nil {
		return Manifest{}, err
	}
	data, err := os.ReadFile(filepath.Join(tmpDir, "manifest.json"))
	if err != nil {
		return Manifest{}, err
	}
	manifest, err := decodeManifest(data)
	if err != nil {
		return Manifest{}, err
	}
	for _, file := range manifest.Files {
		path := filepath.Join(tmpDir, filepath.FromSlash(file.Path))
		sum, size, err := fileSHA256(path)
		if err != nil {
			return Manifest{}, err
		}
		if sum != file.SHA256 || size != file.Bytes {
			return Manifest{}, fmt.Errorf("checksum mismatch for %q", file.Path)
		}
	}
	return manifest, nil
}

// HashFile returns hex SHA-256 for a file (exported for tests).
func HashFile(path string) (string, error) {
	sum, _, err := fileSHA256(path)
	return sum, err
}

// HashBytes returns hex SHA-256 for raw bytes.
func HashBytes(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
