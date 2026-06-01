package ops

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type alertsStoreFile struct {
	Version int          `json:"version"`
	Groups  []AlertGroup `json:"groups"`
}

func loadAlertsStore(path string) (*alertsStoreFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &alertsStoreFile{Version: 1, Groups: []AlertGroup{}}, nil
		}
		return nil, err
	}
	var store alertsStoreFile
	if len(data) == 0 {
		store.Groups = []AlertGroup{}
	} else if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	if store.Groups == nil {
		store.Groups = []AlertGroup{}
	}
	if store.Version == 0 {
		store.Version = 1
	}
	return &store, nil
}

func saveAlertsStore(path string, store *alertsStoreFile) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
