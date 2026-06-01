package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type storeFile struct {
	Version  int       `json:"version"`
	Clusters []Cluster `json:"clusters"`
}

// LoadStore reads clusters from path; missing file yields an empty store.
func LoadStore(storePath string) (*storeFile, error) {
	data, err := os.ReadFile(storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &storeFile{Version: 1, Clusters: []Cluster{}}, nil
		}
		return nil, err
	}
	var store storeFile
	if len(data) == 0 {
		store.Clusters = []Cluster{}
	} else if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Clusters == nil {
		store.Clusters = []Cluster{}
	}
	if store.Version == 0 {
		store.Version = 1
	}
	return &store, nil
}

// SaveStore persists clusters to path.
func SaveStore(storePath string, store *storeFile) error {
	if err := os.MkdirAll(filepath.Dir(storePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(storePath, data, 0o644)
}
