package ops

import (
	"os"
	"strings"
	"time"
)

// AlertTimelineEvent represents an audit timeline event in an alert group's lifecycle.
type AlertTimelineEvent struct {
	Type        string `json:"type"`        // status_change, assignee_change, ack_note, resolved_reason, etc.
	Operator    string `json:"operator"`    // operator username or system
	TimestampMs int64  `json:"timestampMs"`
	Message     string `json:"message"`
}

// FingerprintFields determines which fields are used to build alert fingerprint keys.
var FingerprintFields = []string{"alertname", "service", "instance", "clusterId", "component"}

func init() {
	if envFields := os.Getenv("OPENOCTA_ALERT_FINGERPRINT_FIELDS"); envFields != "" {
		parts := strings.Split(envFields, ",")
		for i, p := range parts {
			parts[i] = strings.TrimSpace(p)
		}
		FingerprintFields = parts
	}
}

// Alert group lifecycle statuses.
const (
	AlertStatusActive    = "active"
	AlertStatusAnalyzing = "analyzing"
	AlertStatusResolved  = "resolved"
)

// AlertEvent is one raw alert ingested before merge.
type AlertEvent struct {
	AlertID    string `json:"alertId,omitempty"`
	Title      string `json:"title,omitempty"`
	Message    string `json:"message,omitempty"`
	Severity   string `json:"severity,omitempty"`
	ReceivedAt int64  `json:"receivedAtMs"`
	Alertname  string `json:"alertname,omitempty"`
	Service    string `json:"service,omitempty"`
	Instance   string `json:"instance,omitempty"`
	ClusterID  string `json:"clusterId,omitempty"`
	Component  string `json:"component,omitempty"`
}

// AlertGroup is a merged alert batch with optional Agent analysis.
type AlertGroup struct {
	ID                  string                 `json:"id"`
	Source              string                 `json:"source"`
	Domain              string                 `json:"domain,omitempty"`
	Title               string                 `json:"title"`
	Severity            string                 `json:"severity"`
	Status              string                 `json:"status"`
	OriginalCount       int                    `json:"originalCount"`
	ReducedTo           int                    `json:"reducedTo"`
	SessionKey          string                 `json:"sessionKey,omitempty"`
	RunID               string                 `json:"runId,omitempty"`
	RootCauseMarkdown   string                 `json:"rootCauseMarkdown,omitempty"`
	ImpactMarkdown      string                 `json:"impactMarkdown,omitempty"`
	Events              []AlertEvent           `json:"events,omitempty"`
	CreatedAtMs         int64                  `json:"createdAtMs"`
	UpdatedAtMs         int64                  `json:"updatedAtMs"`
	Alertname           string                 `json:"alertname,omitempty"`
	Service             string                 `json:"service,omitempty"`
	Instance            string                 `json:"instance,omitempty"`
	ClusterID           string                 `json:"clusterId,omitempty"`
	Component           string                 `json:"component,omitempty"`
	Assignee            string                 `json:"assignee,omitempty"`
	AckNote             string                 `json:"ackNote,omitempty"`
	ResolvedReason      string                 `json:"resolvedReason,omitempty"`
	Timeline            []AlertTimelineEvent   `json:"timeline,omitempty"`
	DiagnosticStatus    string                 `json:"diagnosticStatus,omitempty"`
	RootCauseSummary    string                 `json:"rootCauseSummary,omitempty"`
	ImpactAnalysis      string                 `json:"impactAnalysis,omitempty"`
	SuggestedActions    string                 `json:"suggestedActions,omitempty"`
	Evidence            map[string]interface{} `json:"evidence,omitempty"`
	SuppressionCategory string                 `json:"suppressionCategory,omitempty"`
	SuppressionDetail   string                 `json:"suppressionDetail,omitempty"`
	ReviewStatus        string                 `json:"reviewStatus,omitempty"`
	ReviewNote          string                 `json:"reviewNote,omitempty"`
}

// AlertGroupsListResponse is returned by GET /api/ops/alerts/groups.
type AlertGroupsListResponse struct {
	Groups         []AlertGroup `json:"groups"`
	Total          int          `json:"total"`
	OriginalTotal  int          `json:"originalTotal"`
	MergedTotal    int          `json:"mergedTotal"`
	ReductionRate  float64      `json:"reductionRate"`
	PendingActive  int          `json:"pendingActive"`
}

// AlertGroupPatch supports resolving a group from the UI.
type AlertGroupPatch struct {
	Status         *string `json:"status"`
	Assignee       *string `json:"assignee"`
	AckNote        *string `json:"ackNote"`
	ResolvedReason *string `json:"resolvedReason"`
	ReviewStatus   *string `json:"reviewStatus"`
	ReviewNote     *string `json:"reviewNote"`
}

func normalizeSeverity(sev string) string {
	s := strings.TrimSpace(strings.ToLower(sev))
	switch s {
	case "critical", "crit", "fatal", "error":
		return "critical"
	case "warn", "warning":
		return "warning"
	default:
		return "info"
	}
}

func inferDomainFromSource(source string) string {
	s := strings.ToLower(strings.TrimSpace(source))
	switch {
	case strings.Contains(s, "hadoop"), strings.Contains(s, "yarn"), strings.Contains(s, "hdfs"), strings.Contains(s, "bch"):
		return DomainHadoop
	case strings.Contains(s, "fusion"), strings.Contains(s, "fi-"):
		return DomainFI
	case strings.Contains(s, "gbase"):
		return DomainGBase
	case strings.Contains(s, "governance"), strings.Contains(s, "metadata"), strings.Contains(s, "lineage"):
		return DomainGovernance
	case strings.Contains(s, "dataapp"), strings.Contains(s, "scheduler"), strings.Contains(s, "pipeline"):
		return DomainDataApps
	default:
		return ""
	}
}

func pickGroupTitle(alerts []AlertEvent) string {
	for _, a := range alerts {
		if t := strings.TrimSpace(a.Title); t != "" {
			return t
		}
	}
	for _, a := range alerts {
		if m := strings.TrimSpace(a.Message); m != "" {
			if len(m) > 120 {
				return m[:120] + "…"
			}
			return m
		}
	}
	return "合并告警组"
}

func pickGroupSeverity(alerts []AlertEvent) string {
	worst := "info"
	rank := map[string]int{"info": 0, "warning": 1, "critical": 2}
	for _, a := range alerts {
		s := normalizeSeverity(a.Severity)
		if rank[s] > rank[worst] {
			worst = s
		}
	}
	return worst
}

func formatGroupTimestamp(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}
