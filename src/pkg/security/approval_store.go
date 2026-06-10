package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type approvalStore interface {
	Load() (map[string]*ApprovalRecord, map[string]time.Time, error)
	Persist(records map[string]*ApprovalRecord, whitelist map[string]time.Time) error
	SaveRecord(rec *ApprovalRecord) error
	SaveWhitelist(whitelist map[string]time.Time) error
}

type jsonApprovalStore struct {
	path string
}

func (s *jsonApprovalStore) Load() (map[string]*ApprovalRecord, map[string]time.Time, error) {
	records := make(map[string]*ApprovalRecord)
	whitelist := make(map[string]time.Time)
	if s.path == "" {
		return records, whitelist, nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return nil, nil, fmt.Errorf("security: create approval dir: %w", err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return records, whitelist, nil
		}
		return nil, nil, fmt.Errorf("security: load approvals: %w", err)
	}
	var snapshot approvalSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, nil, fmt.Errorf("security: parse approvals: %w", err)
	}
	for _, rec := range snapshot.Records {
		if rec != nil {
			records[rec.ID] = rec
		}
	}
	for session, expiry := range snapshot.Whitelist {
		whitelist[session] = expiry
	}
	return records, whitelist, nil
}

func (s *jsonApprovalStore) Persist(records map[string]*ApprovalRecord, whitelist map[string]time.Time) error {
	if s.path == "" {
		return nil
	}
	snapshot := approvalSnapshot{
		Records:   make([]*ApprovalRecord, 0, len(records)),
		Whitelist: make(map[string]time.Time, len(whitelist)),
	}
	for _, rec := range records {
		snapshot.Records = append(snapshot.Records, rec)
	}
	for session, expiry := range whitelist {
		snapshot.Whitelist[session] = expiry
	}
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("security: encode approvals: %w", err)
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("security: write approvals: %w", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return fmt.Errorf("security: atomically replace approvals: %w", err)
	}
	return nil
}

func (s *jsonApprovalStore) SaveRecord(rec *ApprovalRecord) error {
	records, whitelist, err := s.Load()
	if err != nil {
		return err
	}
	if rec != nil {
		records[rec.ID] = rec
	}
	return s.Persist(records, whitelist)
}

func (s *jsonApprovalStore) SaveWhitelist(whitelist map[string]time.Time) error {
	records, _, err := s.Load()
	if err != nil {
		return err
	}
	return s.Persist(records, whitelist)
}

func resolveApprovalStore(storePath string) approvalStore {
	if useJSONApprovalStore() {
		return &jsonApprovalStore{path: storePath}
	}
	if repo := defaultApprovalRepository(); repo != nil {
		_ = repo.ImportFromJSONIfEmpty(storePath)
		return repo
	}
	return &jsonApprovalStore{path: storePath}
}

func useJSONApprovalStore() bool {
	return os.Getenv("OPENOCTA_APPROVAL_JSON_STORE") == "1"
}

func approvalStoreKey(storePath string) string {
	if useJSONApprovalStore() || defaultApprovalRepository() == nil {
		return storePath
	}
	return "@openocta:approvals:db"
}
