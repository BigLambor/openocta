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

// DefaultRules defines the pre-configured rules.
var DefaultRules = []TriggerRule{
	{
		ID:         "rule-bch-alert-oncall",
		Name:       "BCH 告警自动分析规则",
		Enabled:    true,
		Domain:     "hadoop",
		Severity:   "critical",
		EmployeeID: "emp_bch_duty",
	},
	{
		ID:         "rule-fi-alert-oncall",
		Name:       "FusionInsight 告警自动分析规则",
		Enabled:    true,
		Domain:     "fi",
		Severity:   "critical",
		EmployeeID: "emp_bch_duty",
	},
	{
		ID:         "rule-gbase-alert-oncall",
		Name:       "GBase 告警自动分析规则",
		Enabled:    true,
		Domain:     "gbase",
		EmployeeID: "emp_bch_duty",
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
