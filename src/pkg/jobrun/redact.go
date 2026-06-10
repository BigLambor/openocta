package jobrun

import (
	"encoding/json"
	"regexp"
	"strings"
)

const maxSummaryLen = 500

var (
	bearerTokenRe = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-+/=]+`)
	jsonSensitiveKeyRe = regexp.MustCompile(`(?i)"((?:password|passwd|secret|token|api[_-]?key|authorization|bearer|dsn|credential|private[_-]?key|access[_-]?key)[^"]*)"\s*:\s*"([^"]*)"`)
	querySensitiveKeyRe = regexp.MustCompile(`(?i)(password|passwd|secret|token|api[_-]?key|dsn|credential)\s*=\s*[^&\s]+`)
)

// RedactText removes sensitive values from audit summaries.
func RedactText(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	out := bearerTokenRe.ReplaceAllString(raw, "Bearer [REDACTED]")
	out = jsonSensitiveKeyRe.ReplaceAllString(out, `"$1":"[REDACTED]"`)
	out = querySensitiveKeyRe.ReplaceAllString(out, "$1=[REDACTED]")
	return truncateSummary(out)
}

// SummarizePayload converts arbitrary tool input/output to a redacted summary string.
func SummarizePayload(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "{") || strings.HasPrefix(raw, "[") {
		var parsed interface{}
		if err := json.Unmarshal([]byte(raw), &parsed); err == nil {
			if b, err := json.Marshal(parsed); err == nil {
				return RedactText(string(b))
			}
		}
	}
	return RedactText(raw)
}

func truncateSummary(s string) string {
	if len(s) <= maxSummaryLen {
		return s
	}
	return s[:maxSummaryLen] + "..."
}

func inferToolProvider(toolName string) string {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if strings.HasPrefix(name, "mcp") || strings.Contains(name, "mcp_") || strings.Contains(name, "/") {
		return "mcp"
	}
	return "agent"
}
