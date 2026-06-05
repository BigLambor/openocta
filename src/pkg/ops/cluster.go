package ops

import (
	"fmt"
	"strings"
	"time"
)

// Domain keys align with UI navigation tabs.
const (
	DomainHadoop     = "hadoop"
	DomainFI         = "fi"
	DomainGBase      = "gbase"
	DomainGovernance = "governance"
	DomainDataApps   = "dataapps"
)

var validDomains = map[string]struct{}{
	DomainHadoop:     {},
	DomainFI:         {},
	DomainGBase:      {},
	DomainGovernance: {},
	DomainDataApps:   {},
}

var validStatuses = map[string]struct{}{
	"healthy":  {},
	"warning":  {},
	"critical": {},
	"unknown":  {},
	"inactive": {},
}

// Cluster is a managed ops asset (CMDB row).
type Cluster struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Domain         string   `json:"domain"`
	Region         string   `json:"region,omitempty"`
	NodeCount      int      `json:"nodeCount"`
	Components     []string `json:"components"`
	Owner          string   `json:"owner,omitempty"`
	Status         string   `json:"status"`
	Description    string   `json:"description,omitempty"`
	CreatedAtMs    int64    `json:"createdAtMs"`
	UpdatedAtMs    int64    `json:"updatedAtMs"`
	MonitorLabels  string   `json:"monitorLabels,omitempty"`
	VMUrlRef       string   `json:"vmUrlRef,omitempty"`
	MetricsBaseUrl string   `json:"metricsBaseUrl,omitempty"`
	JMXUrl         string   `json:"jmxUrl,omitempty"`
	FIManagerUrl   string   `json:"fiManagerUrl,omitempty"`
	GBaseDsnRef    string   `json:"gbaseDsnRef,omitempty"`
	CredentialsRef string   `json:"credentialsRef,omitempty"`
}

// ClusterCreate is the POST body for registering a cluster.
type ClusterCreate struct {
	Name           string   `json:"name"`
	Domain         string   `json:"domain"`
	Region         string   `json:"region"`
	NodeCount      int      `json:"nodeCount"`
	Components     []string `json:"components"`
	Owner          string   `json:"owner"`
	Status         string   `json:"status"`
	Description    string   `json:"description"`
	MonitorLabels  string   `json:"monitorLabels"`
	VMUrlRef       string   `json:"vmUrlRef"`
	MetricsBaseUrl string   `json:"metricsBaseUrl"`
	JMXUrl         string   `json:"jmxUrl"`
	FIManagerUrl   string   `json:"fiManagerUrl"`
	GBaseDsnRef    string   `json:"gbaseDsnRef"`
	CredentialsRef string   `json:"credentialsRef"`
}

// ClusterPatch is a partial update (PATCH).
type ClusterPatch struct {
	Name           *string   `json:"name"`
	Domain         *string   `json:"domain"`
	Region         *string   `json:"region"`
	NodeCount      *int      `json:"nodeCount"`
	Components     *[]string `json:"components"`
	Owner          *string   `json:"owner"`
	Status         *string   `json:"status"`
	Description    *string   `json:"description"`
	MonitorLabels  *string   `json:"monitorLabels"`
	VMUrlRef       *string   `json:"vmUrlRef"`
	MetricsBaseUrl *string   `json:"metricsBaseUrl"`
	JMXUrl         *string   `json:"jmxUrl"`
	FIManagerUrl   *string   `json:"fiManagerUrl"`
	GBaseDsnRef    *string   `json:"gbaseDsnRef"`
	CredentialsRef *string   `json:"credentialsRef"`
}

// DashboardSummary aggregates cluster assets for the ops overview (P1-5).
type DashboardSummary struct {
	TotalClusters    int                   `json:"totalClusters"`
	HealthyClusters  int                   `json:"healthyClusters"`
	WarningClusters  int                   `json:"warningClusters"`
	CriticalClusters int                   `json:"criticalClusters"`
	PendingAlerts    int                   `json:"pendingAlerts"`
	VMConfigured     bool                  `json:"vmConfigured"`
	Domains          []DomainHealthSummary `json:"domains"`
}

// DomainHealthSummary is per-domain rollup on the dashboard.
type DomainHealthSummary struct {
	Domain            string   `json:"domain"`
	ClusterCount      int      `json:"clusterCount"`
	HealthyCount      int      `json:"healthyCount"`
	WarningCount      int      `json:"warningCount"`
	CriticalCount     int      `json:"criticalCount"`
	HealthScore       *int     `json:"healthScore,omitempty"`
	HealthScoreSource string   `json:"healthScoreSource,omitempty"`
	HealthScoreNote   string   `json:"healthScoreNote,omitempty"`
	ScoreStatus       string   `json:"scoreStatus,omitempty"`
	Coverage          *float64 `json:"coverage,omitempty"`
	MissingSources    []string `json:"missingSources,omitempty"`
	PresentSources    []string `json:"presentSources,omitempty"`
	Note              string   `json:"note,omitempty"`
}

func NormalizeDomain(domain string) (string, error) {
	d := strings.TrimSpace(strings.ToLower(domain))
	if d == "" {
		return "", fmt.Errorf("domain 不能为空")
	}
	if _, ok := validDomains[d]; !ok {
		return "", fmt.Errorf("无效的业务域: %s", domain)
	}
	return d, nil
}

func NormalizeStatus(status string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(status))
	if s == "" {
		return "unknown", nil
	}
	if _, ok := validStatuses[s]; !ok {
		return "", fmt.Errorf("无效的状态: %s", status)
	}
	return s, nil
}

func domainDisplayName(domain string) string {
	switch domain {
	case DomainHadoop:
		return "BCH生态"
	case DomainFI:
		return "FI商业生态"
	case DomainGBase:
		return "GBase数据库"
	case DomainGovernance:
		return "开发治理平台"
	case DomainDataApps:
		return "数据App运维"
	default:
		return domain
	}
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
