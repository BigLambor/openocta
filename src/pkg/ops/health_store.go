package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
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
