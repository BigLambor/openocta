package automation

import (
	"strings"
)

// TriggerRule defines a rule to link alert events to an AI digital employee.
type TriggerRule struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Enabled    bool   `json:"enabled"`
	Domain     string `json:"domain"`     // hadoop, fi, gbase, governance, dataapps
	Severity   string `json:"severity"`   // critical, warning
	Alertname  string `json:"alertname"`  // substring match in title/alertname
	EmployeeID string `json:"employeeId"` // target digital employee ID
}

// domainEmployeeIDs maps a technical domain to its default seeded digital-employee ID.
// Single source of truth shared by alert auto-routing (MatchAlert) and the
// pre-configured rules below. Keep in sync with the seed manifests in
// pkg/init/employee.go and the frontend opsAssistantForDomain mapping.
var domainEmployeeIDs = map[string]string{
	"hadoop":     "emp_bch_duty",
	"fi":         "emp_fi_inspect",
	"gbase":      "emp_gbase_diagnose",
	"governance": "emp_governance_remediate",
	"dataapps":   "emp_dataapps_ops",
}

// DefaultEmployeeForDomain returns the seeded employee ID for a technical domain.
func DefaultEmployeeForDomain(domain string) (string, bool) {
	id, ok := domainEmployeeIDs[strings.ToLower(strings.TrimSpace(domain))]
	return id, ok
}

// DefaultRules defines the pre-configured rules. Each domain routes to its own
// seeded expert digital employee rather than a single shared one.
var DefaultRules = []TriggerRule{
	{
		ID:         "rule-bch-alert-oncall",
		Name:       "BCH 告警自动分析规则",
		Enabled:    true,
		Domain:     "hadoop",
		Severity:   "critical",
		EmployeeID: domainEmployeeIDs["hadoop"],
	},
	{
		ID:         "rule-fi-alert-oncall",
		Name:       "FusionInsight 告警自动分析规则",
		Enabled:    true,
		Domain:     "fi",
		Severity:   "critical",
		EmployeeID: domainEmployeeIDs["fi"],
	},
	{
		ID:         "rule-gbase-alert-oncall",
		Name:       "GBase 告警自动分析规则",
		Enabled:    true,
		Domain:     "gbase",
		EmployeeID: domainEmployeeIDs["gbase"],
	},
	{
		ID:         "rule-governance-alert",
		Name:       "开发治理告警自动分析规则",
		Enabled:    true,
		Domain:     "governance",
		EmployeeID: domainEmployeeIDs["governance"],
	},
	{
		ID:         "rule-dataapps-alert",
		Name:       "数据 App 告警自动分析规则",
		Enabled:    true,
		Domain:     "dataapps",
		EmployeeID: domainEmployeeIDs["dataapps"],
	},
}

// MatchAlert matches an alert group against the rules and returns the target EmployeeID.
func MatchAlert(domain, severity, title string) (string, bool) {
	for _, rule := range DefaultRules {
		if !rule.Enabled {
			continue
		}
		if rule.Domain != "" && !strings.EqualFold(domain, rule.Domain) {
			continue
		}
		if rule.Severity != "" && !strings.EqualFold(severity, rule.Severity) {
			continue
		}
		if rule.Alertname != "" && !strings.Contains(strings.ToLower(title), strings.ToLower(rule.Alertname)) {
			continue
		}
		return rule.EmployeeID, true
	}
	return "", false
}
