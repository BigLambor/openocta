package runtime

import "strings"

var defaultHighRiskAskPatterns = []string{
	"bash", "sh", "shell", "powershell", "pwsh", "cmd", "windows_exec_cmd",
	"write", "edit", "file_write", "file_edit",
	"restart", "reboot", "shutdown", "kill", "rollback", "scale", "deploy",
	"kubectl", "systemctl", "service", "docker", "helm", "terraform", "ansible-playbook",
	"rm", "mv", "chmod", "chown", "drop", "truncate", "delete",
}

var defaultHighRiskDenyFragments = []string{
	"rm -rf /",
	"mkfs",
	"dd if=",
	":(){ :|:& };:",
}

// ApplyCommercialHighRiskDefaults merges commercial P0 ask/deny rules onto a resolved policy.
func ApplyCommercialHighRiskDefaults(base *ResolvedCommandPolicy) *ResolvedCommandPolicy {
	if base == nil {
		base = &ResolvedCommandPolicy{
			Enabled:       true,
			DefaultPolicy: "ask",
			MaxLength:     4096,
		}
	}
	if !base.Enabled {
		return base
	}
	if base.DefaultPolicy == "" {
		base.DefaultPolicy = "ask"
	}

	existingAsk := map[string]struct{}{}
	for _, rule := range base.AskRules {
		existingAsk[strings.ToLower(rule.Pattern)] = struct{}{}
	}
	for _, pattern := range defaultHighRiskAskPatterns {
		p := strings.ToLower(strings.TrimSpace(pattern))
		if p == "" {
			continue
		}
		if _, ok := existingAsk[p]; ok {
			continue
		}
		typ := "command"
		if strings.Contains(p, " ") {
			typ = "fragment"
		}
		base.AskRules = append(base.AskRules, CommandRule{Pattern: p, Type: typ})
		existingAsk[p] = struct{}{}
	}

	existingDeny := map[string]struct{}{}
	for _, rule := range base.DenyRules {
		if rule.Type == "fragment" {
			existingDeny[strings.ToLower(rule.Pattern)] = struct{}{}
		}
	}
	for _, fragment := range defaultHighRiskDenyFragments {
		f := strings.ToLower(strings.TrimSpace(fragment))
		if f == "" {
			continue
		}
		if _, ok := existingDeny[f]; ok {
			continue
		}
		base.DenyRules = append(base.DenyRules, CommandRule{Pattern: f, Type: "fragment"})
	}
	return base
}
