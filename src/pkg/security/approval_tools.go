package security

import "strings"

// ToolsRequiringApproval lists built-in tools that must pass the approval queue by default.
var ToolsRequiringApproval = map[string]struct{}{
	"bash":             {},
	"sh":               {},
	"shell":            {},
	"powershell":       {},
	"pwsh":             {},
	"cmd":              {},
	"windows_exec_cmd": {},
	"write":            {},
	"edit":             {},
	"file_write":       {},
	"file_edit":        {},
}

// ToolRequiresApproval reports whether a tool invocation must go through approval.
func ToolRequiresApproval(toolName string) bool {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if name == "" {
		return false
	}
	if _, ok := ToolsRequiringApproval[name]; ok {
		return true
	}
	return false
}
