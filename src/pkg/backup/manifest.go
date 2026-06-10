package backup

import (
	"encoding/json"
	"time"
)

const FormatVersion = 1

// Manifest describes a backup archive and its integrity checksums.
type Manifest struct {
	FormatVersion    int               `json:"formatVersion"`
	OpenOctaVersion  string            `json:"openoctaVersion"`
	CreatedAt        time.Time         `json:"createdAt"`
	StateDir         string            `json:"stateDir"`
	SchemaVersion    int64             `json:"schemaVersion"`
	SchemaVersionMax int64             `json:"schemaVersionMax"`
	Files            []ManifestFile    `json:"files"`
	Notes            map[string]string `json:"notes,omitempty"`
}

// ManifestFile records one file in the archive with a SHA-256 checksum.
type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

func encodeManifest(m Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

func decodeManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}
