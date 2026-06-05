package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	HealthObjectCluster = "cluster"

	SignalTypeGBaseSQL      = "gbase_sql"
	SignalTypeMetrics       = "metrics"
	SignalTypeAlerts        = "alerts"
	SignalTypeInspection    = "inspection"
	SignalTypeAssetStatus   = "asset_status"
	SignalTypeJMX           = "jmx"
	SignalTypeBCHWorkload   = "bch_workload"
	SignalTypeFIManager     = "fi_manager"
	SignalTypeGovernanceAPI = "governance_api"
	SignalTypeSchedulerAPI  = "scheduler_api"

	SourceKindPlatformTool = "platform_tool"
	SourceKindMCP          = "mcp"
	SourceKindCollector    = "collector"

	HealthStatusHealthy  = "healthy"
	HealthStatusWarning  = "warning"
	HealthStatusCritical = "critical"
	HealthStatusUnknown  = "unknown"

	ScoreStatusOK       = "ok"
	ScoreStatusWarning  = "warning"
	ScoreStatusCritical = "critical"
	ScoreStatusPartial  = "partial"
	ScoreStatusDegraded = "degraded"
	ScoreStatusUnknown  = "unknown"

	FreshnessOK      = "ok"
	FreshnessExpired = "expired"
)

// HealthSignal is one structured L3 observation from a collector, tool, or MCP.
type HealthSignal struct {
	SchemaVersion string                 `json:"schemaVersion"`
	ID            string                 `json:"id"`
	RunID         string                 `json:"runId"`
	ScenarioKey   string                 `json:"scenarioKey"`
	ObjectType    string                 `json:"objectType"`
	ObjectID      string                 `json:"objectId"`
	ClusterID     string                 `json:"clusterId,omitempty"`
	Domain        string                 `json:"domain"`
	Tenant        string                 `json:"tenant,omitempty"`
	Env           string                 `json:"env,omitempty"`
	Region        string                 `json:"region,omitempty"`
	Type          string                 `json:"type"`
	Status        string                 `json:"status"`
	Score         *int                   `json:"score,omitempty"`
	Confidence    string                 `json:"confidence"`
	Source        string                 `json:"source"`
	SourceKind    string                 `json:"sourceKind"`
	Evidence      map[string]interface{} `json:"evidence,omitempty"`
	Error         string                 `json:"error,omitempty"`
	ObservedAt    string                 `json:"observedAt"`
	TTLSec        int                    `json:"ttlSec"`
	Freshness     string                 `json:"freshness"`
}

// HealthSnapshot is the aggregate L3 health view for a cluster/job/db object.
type HealthSnapshot struct {
	SchemaVersion            string         `json:"schemaVersion"`
	AggregationPolicyVersion string         `json:"aggregationPolicyVersion"`
	ObjectType               string         `json:"objectType"`
	ObjectID                 string         `json:"objectId"`
	ClusterID                string         `json:"clusterId,omitempty"`
	Domain                   string         `json:"domain"`
	Score                    *int           `json:"score,omitempty"`
	ScoreStatus              string         `json:"scoreStatus"`
	Source                   string         `json:"source"`
	Coverage                 float64        `json:"coverage"`
	MissingSources           []string       `json:"missingSources,omitempty"`
	PresentSources           []string       `json:"presentSources,omitempty"`
	Signals                  []HealthSignal `json:"signals,omitempty"`
	ObservedAt               string         `json:"observedAt"`
}

// DomainHealthSnapshot is the aggregate health view for an entire domain.
type DomainHealthSnapshot struct {
	Domain                  string         `json:"domain"`
	AverageScore            *int           `json:"averageScore,omitempty"`
	TotalClusters           int            `json:"totalClusters"`
	HealthyClusters         int            `json:"healthyClusters"`
	WarningClusters         int            `json:"warningClusters"`
	CriticalClusters        int            `json:"criticalClusters"`
	PartialClusters         int            `json:"partialClusters"`
	DegradedClusters        int            `json:"degradedClusters"`
	UnknownClusters         int            `json:"unknownClusters"`
	MissingSourcesBreakdown map[string]int `json:"missingSourcesBreakdown,omitempty"`
	ObservedAt              string         `json:"observedAt"`
}

// DomainHealthPolicy controls composite-score eligibility and weighting.
type DomainHealthPolicy struct {
	PolicyVersion       string
	RequiredAnyOf       []string
	OptionalSources     []string
	Weights             map[string]float64
	MinCoverageForScore float64
	CoverageScope       string
	DefaultTTLSec       int
}

func defaultDomainHealthPolicy(domain string) *DomainHealthPolicy {
	switch domain {
	case DomainGBase:
		return &DomainHealthPolicy{
			PolicyVersion:       "1",
			RequiredAnyOf:       []string{SignalTypeGBaseSQL},
			OptionalSources:     []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection, SignalTypeAssetStatus},
			Weights:             map[string]float64{SignalTypeGBaseSQL: 0.45, SignalTypeMetrics: 0.25, SignalTypeAlerts: 0.20, SignalTypeInspection: 0.10, SignalTypeAssetStatus: 0.05},
			MinCoverageForScore: 0.5,
			CoverageScope:       "configured",
			DefaultTTLSec:       300,
		}
	case DomainHadoop: // BCH
		return &DomainHealthPolicy{
			PolicyVersion:       "1",
			RequiredAnyOf:       []string{SignalTypeMetrics, SignalTypeJMX, SignalTypeBCHWorkload},
			OptionalSources:     []string{SignalTypeAlerts, SignalTypeInspection, SignalTypeAssetStatus},
			Weights:             map[string]float64{SignalTypeMetrics: 0.3, SignalTypeJMX: 0.3, SignalTypeBCHWorkload: 0.2, SignalTypeAlerts: 0.1, SignalTypeInspection: 0.05, SignalTypeAssetStatus: 0.05},
			MinCoverageForScore: 0.5,
			CoverageScope:       "configured",
			DefaultTTLSec:       300,
		}
	case DomainFI:
		return &DomainHealthPolicy{
			PolicyVersion:       "1",
			RequiredAnyOf:       []string{SignalTypeFIManager},
			OptionalSources:     []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection, SignalTypeAssetStatus},
			Weights:             map[string]float64{SignalTypeFIManager: 0.6, SignalTypeMetrics: 0.2, SignalTypeAlerts: 0.1, SignalTypeInspection: 0.05, SignalTypeAssetStatus: 0.05},
			MinCoverageForScore: 0.5,
			CoverageScope:       "configured",
			DefaultTTLSec:       300,
		}
	case DomainGovernance:
		return &DomainHealthPolicy{
			PolicyVersion:       "1",
			RequiredAnyOf:       []string{SignalTypeGovernanceAPI, SignalTypeMetrics},
			OptionalSources:     []string{SignalTypeAlerts, SignalTypeInspection},
			Weights:             map[string]float64{SignalTypeGovernanceAPI: 0.5, SignalTypeMetrics: 0.3, SignalTypeAlerts: 0.1, SignalTypeInspection: 0.1},
			MinCoverageForScore: 0.5,
			CoverageScope:       "configured",
			DefaultTTLSec:       300,
		}
	case DomainDataApps:
		return &DomainHealthPolicy{
			PolicyVersion:       "1",
			RequiredAnyOf:       []string{SignalTypeSchedulerAPI},
			OptionalSources:     []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
			Weights:             map[string]float64{SignalTypeSchedulerAPI: 0.6, SignalTypeMetrics: 0.2, SignalTypeAlerts: 0.1, SignalTypeInspection: 0.1},
			MinCoverageForScore: 0.5,
			CoverageScope:       "configured",
			DefaultTTLSec:       300,
		}
	default:
		return nil
	}
}

func newCollectorSignal(cluster Cluster, typ, status string, score *int, evidence map[string]interface{}, errText string) HealthSignal {
	now := time.Now().UTC().Format(time.RFC3339)
	ttl := 300
	if p := defaultDomainHealthPolicy(cluster.Domain); p != nil && p.DefaultTTLSec > 0 {
		ttl = p.DefaultTTLSec
	}
	return HealthSignal{
		SchemaVersion: "1",
		ID:            "sig-" + uuid.New().String(),
		RunID:         "collector-" + typ + "-" + fmt.Sprint(time.Now().UnixMilli()),
		ScenarioKey:   "system:collector",
		ObjectType:    HealthObjectCluster,
		ObjectID:      cluster.ID,
		ClusterID:     cluster.ID,
		Domain:        cluster.Domain,
		Region:        cluster.Region,
		Type:          typ,
		Status:        status,
		Score:         score,
		Confidence:    "high",
		Source:        "collector:" + typ,
		SourceKind:    SourceKindCollector,
		Evidence:      evidence,
		Error:         errText,
		ObservedAt:    now,
		TTLSec:        ttl,
		Freshness:     FreshnessOK,
	}
}

func signalObservedTime(s HealthSignal) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(s.ObservedAt))
	if err != nil {
		return time.Time{}
	}
	return t
}

func signalExpired(s HealthSignal, now time.Time) bool {
	if s.TTLSec <= 0 {
		return false
	}
	t := signalObservedTime(s)
	return !t.IsZero() && now.After(t.Add(time.Duration(s.TTLSec)*time.Second))
}

func scoreStatusFromScore(score int) string {
	switch {
	case score >= 90:
		return ScoreStatusOK
	case score >= 75:
		return ScoreStatusWarning
	default:
		return ScoreStatusCritical
	}
}
