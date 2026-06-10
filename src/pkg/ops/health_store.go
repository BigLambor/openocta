package ops

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

type healthSignalStoreFile struct {
	Version int            `json:"version"`
	Items   []HealthSignal `json:"items"`
}

type healthSnapshotStoreFile struct {
	Version int              `json:"version"`
	Items   []HealthSnapshot `json:"items"`
}

// HealthSignalStore abstracts the Phase 1 JSON backend so SQLite can replace it later.
type HealthSignalStore interface {
	ListSignals() ([]HealthSignal, error)
	UpsertSignals([]HealthSignal) error
	ListSnapshots() ([]HealthSnapshot, error)
	GetHealthSnapshot(clusterID string) (HealthSnapshot, bool)
	UpsertSnapshots([]HealthSnapshot) error
	AggregateDomainSnapshot(domain string) (DomainHealthSnapshot, error)
}

type jsonHealthSignalStore struct {
	mu            sync.RWMutex
	signalsPath   string
	snapshotsPath string
	signals       []HealthSignal
	snapshots     []HealthSnapshot
}

var healthStore HealthSignalStore

func InitHealthStore(stateDir string) error {
	sqliteDB := db.GetDB()
	if sqliteDB != nil {
		signalsPath := filepath.Join(stateDir, "ops", "health_signals.json")
		snapshotsPath := filepath.Join(stateDir, "ops", "health_snapshots.json")

		// Run migration if JSON exists
		if err := migrateJSONToSQLite(sqliteDB, signalsPath, snapshotsPath); err != nil {
			// Log warning but don't fail startup
			fmt.Printf("warning: health JSON to SQLite migration failed: %v\n", err)
		}

		healthStore = &sqliteHealthSignalStore{db: sqliteDB}
		return nil
	}

	store, err := newJSONHealthSignalStore(
		filepath.Join(stateDir, "ops", "health_signals.json"),
		filepath.Join(stateDir, "ops", "health_snapshots.json"),
	)
	if err != nil {
		return err
	}
	healthStore = store
	return nil
}

func ListHealthSignals() ([]HealthSignal, error) {
	if healthStore == nil {
		return []HealthSignal{}, nil
	}
	return healthStore.ListSignals()
}

func ListHealthSnapshots() ([]HealthSnapshot, error) {
	if healthStore == nil {
		return []HealthSnapshot{}, nil
	}
	return healthStore.ListSnapshots()
}

func GetHealthSnapshot(clusterID string) (HealthSnapshot, bool) {
	if healthStore == nil {
		return HealthSnapshot{}, false
	}
	return healthStore.GetHealthSnapshot(clusterID)
}

func newJSONHealthSignalStore(signalsPath, snapshotsPath string) (*jsonHealthSignalStore, error) {
	s := &jsonHealthSignalStore{
		signalsPath:   signalsPath,
		snapshotsPath: snapshotsPath,
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *jsonHealthSignalStore) load() error {
	signals, err := loadHealthSignals(s.signalsPath)
	if err != nil {
		return err
	}
	snapshots, err := loadHealthSnapshots(s.snapshotsPath)
	if err != nil {
		return err
	}
	s.signals = signals
	s.snapshots = snapshots
	return nil
}

func loadHealthSignals(path string) ([]HealthSignal, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HealthSignal{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []HealthSignal{}, nil
	}
	var f healthSignalStoreFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Items == nil {
		f.Items = []HealthSignal{}
	}
	return f.Items, nil
}

func loadHealthSnapshots(path string) ([]HealthSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []HealthSnapshot{}, nil
		}
		return nil, err
	}
	if len(data) == 0 {
		return []HealthSnapshot{}, nil
	}
	var f healthSnapshotStoreFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	if f.Items == nil {
		f.Items = []HealthSnapshot{}
	}
	return f.Items, nil
}

func (s *jsonHealthSignalStore) ListSignals() ([]HealthSignal, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HealthSignal, len(s.signals))
	copy(out, s.signals)
	return out, nil
}

func (s *jsonHealthSignalStore) UpsertSignals(items []HealthSignal) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	byKey := make(map[string]int, len(s.signals))
	for i, item := range s.signals {
		byKey[healthSignalKey(item)] = i
	}
	for _, item := range items {
		key := healthSignalKey(item)
		if idx, ok := byKey[key]; ok {
			s.signals[idx] = item
		} else {
			byKey[key] = len(s.signals)
			s.signals = append(s.signals, item)
		}
	}
	return s.persistSignalsLocked()
}

func (s *jsonHealthSignalStore) ListSnapshots() ([]HealthSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]HealthSnapshot, len(s.snapshots))
	copy(out, s.snapshots)
	return out, nil
}

func (s *jsonHealthSignalStore) GetHealthSnapshot(clusterID string) (HealthSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	// Find the most recent snapshot for the given clusterID
	// Currently assumes one per objectId=clusterId, so we search backwards
	for i := len(s.snapshots) - 1; i >= 0; i-- {
		if s.snapshots[i].ObjectID == clusterID {
			return s.snapshots[i], true
		}
	}
	return HealthSnapshot{}, false
}

func (s *jsonHealthSignalStore) UpsertSnapshots(items []HealthSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	byKey := make(map[string]int, len(s.snapshots))
	for i, item := range s.snapshots {
		byKey[healthSnapshotKey(item)] = i
	}
	for _, item := range items {
		key := healthSnapshotKey(item)
		if idx, ok := byKey[key]; ok {
			s.snapshots[idx] = item
		} else {
			byKey[key] = len(s.snapshots)
			s.snapshots = append(s.snapshots, item)
		}
	}
	return s.persistSnapshotsLocked()
}

func (s *jsonHealthSignalStore) persistSignalsLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.signalsPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(healthSignalStoreFile{Version: 1, Items: s.signals}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.signalsPath, data, 0o644)
}

func (s *jsonHealthSignalStore) persistSnapshotsLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.snapshotsPath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(healthSnapshotStoreFile{Version: 1, Items: s.snapshots}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.snapshotsPath, data, 0o644)
}

func (s *jsonHealthSignalStore) AggregateDomainSnapshot(domain string) (DomainHealthSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var (
		totalClusters    int
		healthyClusters  int
		warningClusters  int
		criticalClusters int
		partialClusters  int
		degradedClusters int
		unknownClusters  int
		totalScore       int
		scoredClusters   int
		missingBreakdown = make(map[string]int)
	)

	now := time.Now().UTC().Format(time.RFC3339)

	for _, snap := range s.snapshots {
		if snap.Domain != domain {
			continue
		}
		totalClusters++
		switch snap.ScoreStatus {
		case ScoreStatusOK:
			healthyClusters++
		case ScoreStatusWarning:
			warningClusters++
		case ScoreStatusCritical:
			criticalClusters++
		case ScoreStatusPartial:
			partialClusters++
		case ScoreStatusDegraded:
			degradedClusters++
		default:
			unknownClusters++
		}

		if snap.Score != nil {
			totalScore += *snap.Score
			scoredClusters++
		}

		for _, ms := range snap.MissingSources {
			missingBreakdown[ms]++
		}
	}

	var avgScore *int
	if scoredClusters > 0 {
		val := totalScore / scoredClusters
		avgScore = &val
	}

	return DomainHealthSnapshot{
		Domain:                  domain,
		AverageScore:            avgScore,
		TotalClusters:           totalClusters,
		HealthyClusters:         healthyClusters,
		WarningClusters:         warningClusters,
		CriticalClusters:        criticalClusters,
		PartialClusters:         partialClusters,
		DegradedClusters:        degradedClusters,
		UnknownClusters:         unknownClusters,
		MissingSourcesBreakdown: missingBreakdown,
		ObservedAt:              now,
	}, nil
}

func healthSignalKey(s HealthSignal) string {
	return fmt.Sprintf("%s|%s|%s|%s", s.ObjectType, s.ObjectID, s.Type, s.Source)
}

func healthSnapshotKey(s HealthSnapshot) string {
	return fmt.Sprintf("%s|%s", s.ObjectType, s.ObjectID)
}

func AggregateDomainSnapshot(domain string) (DomainHealthSnapshot, error) {
	if healthStore == nil {
		return DomainHealthSnapshot{Domain: domain}, nil
	}
	return healthStore.AggregateDomainSnapshot(domain)
}

type sqliteHealthSignalStore struct {
	db *sql.DB
}

func migrateJSONToSQLite(db *sql.DB, signalsPath, snapshotsPath string) error {
	if _, err := os.Stat(signalsPath); err == nil {
		signals, err := loadHealthSignals(signalsPath)
		if err == nil && len(signals) > 0 {
			store := &sqliteHealthSignalStore{db: db}
			if err := store.UpsertSignals(signals); err == nil {
				_ = os.Rename(signalsPath, signalsPath+".bak")
			}
		}
	}
	if _, err := os.Stat(snapshotsPath); err == nil {
		snapshots, err := loadHealthSnapshots(snapshotsPath)
		if err == nil && len(snapshots) > 0 {
			store := &sqliteHealthSignalStore{db: db}
			if err := store.UpsertSnapshots(snapshots); err == nil {
				_ = os.Rename(snapshotsPath, snapshotsPath+".bak")
			}
		}
	}
	return nil
}

func (s *sqliteHealthSignalStore) ListSignals() ([]HealthSignal, error) {
	rows, err := s.db.Query(`SELECT detail_json FROM health_signals`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []HealthSignal
	for rows.Next() {
		var detailStr string
		if err := rows.Scan(&detailStr); err != nil {
			return nil, err
		}
		var item HealthSignal
		if err := json.Unmarshal([]byte(detailStr), &item); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}

func (s *sqliteHealthSignalStore) UpsertSignals(items []HealthSignal) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO health_signals (object_type, object_id, type, source, detail_json)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(object_type, object_id, type, source) DO UPDATE SET
			detail_json = excluded.detail_json
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(item.ObjectType, item.ObjectID, item.Type, item.Source, string(b))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteHealthSignalStore) ListSnapshots() ([]HealthSnapshot, error) {
	rows, err := s.db.Query(`SELECT detail_json FROM health_snapshots`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []HealthSnapshot
	for rows.Next() {
		var detailStr string
		if err := rows.Scan(&detailStr); err != nil {
			return nil, err
		}
		var item HealthSnapshot
		if err := json.Unmarshal([]byte(detailStr), &item); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, nil
}

func (s *sqliteHealthSignalStore) GetHealthSnapshot(clusterID string) (HealthSnapshot, bool) {
	var detailStr string
	err := s.db.QueryRow(`
		SELECT detail_json FROM health_snapshots 
		WHERE object_type = 'cluster' AND object_id = ?
	`, clusterID).Scan(&detailStr)
	if err != nil {
		return HealthSnapshot{}, false
	}
	var item HealthSnapshot
	if err := json.Unmarshal([]byte(detailStr), &item); err != nil {
		return HealthSnapshot{}, false
	}
	return item, true
}

func (s *sqliteHealthSignalStore) UpsertSnapshots(items []HealthSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO health_snapshots (object_type, object_id, domain, detail_json)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(object_type, object_id) DO UPDATE SET
			domain = excluded.domain,
			detail_json = excluded.detail_json
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, item := range items {
		b, err := json.Marshal(item)
		if err != nil {
			return err
		}
		_, err = stmt.Exec(item.ObjectType, item.ObjectID, item.Domain, string(b))
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *sqliteHealthSignalStore) AggregateDomainSnapshot(domain string) (DomainHealthSnapshot, error) {
	rows, err := s.db.Query(`
		SELECT detail_json FROM health_snapshots WHERE domain = ?
	`, domain)
	if err != nil {
		return DomainHealthSnapshot{Domain: domain}, err
	}
	defer rows.Close()

	var snapshots []HealthSnapshot
	for rows.Next() {
		var detailStr string
		if err := rows.Scan(&detailStr); err != nil {
			return DomainHealthSnapshot{Domain: domain}, err
		}
		var item HealthSnapshot
		if err := json.Unmarshal([]byte(detailStr), &item); err != nil {
			return DomainHealthSnapshot{Domain: domain}, err
		}
		snapshots = append(snapshots, item)
	}

	var (
		totalClusters    int
		healthyClusters  int
		warningClusters  int
		criticalClusters int
		partialClusters  int
		degradedClusters int
		unknownClusters  int
		totalScore       int
		scoredClusters   int
		missingBreakdown = make(map[string]int)
	)

	now := time.Now().UTC().Format(time.RFC3339)

	for _, snap := range snapshots {
		totalClusters++
		switch snap.ScoreStatus {
		case ScoreStatusOK:
			healthyClusters++
		case ScoreStatusWarning:
			warningClusters++
		case ScoreStatusCritical:
			criticalClusters++
		case ScoreStatusPartial:
			partialClusters++
		case ScoreStatusDegraded:
			degradedClusters++
		default:
			unknownClusters++
		}

		if snap.Score != nil {
			totalScore += *snap.Score
			scoredClusters++
		}

		for _, ms := range snap.MissingSources {
			missingBreakdown[ms]++
		}
	}

	var avgScore *int
	if scoredClusters > 0 {
		val := totalScore / scoredClusters
		avgScore = &val
	}

	return DomainHealthSnapshot{
		Domain:                  domain,
		AverageScore:            avgScore,
		TotalClusters:           totalClusters,
		HealthyClusters:         healthyClusters,
		WarningClusters:         warningClusters,
		CriticalClusters:        criticalClusters,
		PartialClusters:         partialClusters,
		DegradedClusters:        degradedClusters,
		UnknownClusters:         unknownClusters,
		MissingSourcesBreakdown: missingBreakdown,
		ObservedAt:              now,
	}, nil
}
