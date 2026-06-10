package rbac

import (
	"fmt"
	"strings"
)

const (
	PermToolExecute      = "tool:execute"
	PermToolExecuteShell = "tool:execute:shell"
	PermToolExecuteWrite = "tool:execute:write"
	PermToolExecuteMCP   = "tool:execute:mcp"
)

// HasPermission reports whether session holds the given permission code.
// Nil session or admin role bypasses checks (legacy/internal callers).
func HasPermission(session *UserSession, code string) bool {
	if session == nil {
		return true
	}
	if session.RoleName == "admin" {
		return true
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return true
	}
	for _, p := range session.Permissions {
		if p == code {
			return true
		}
	}
	return false
}

// CheckToolExecutePermission verifies the base permission to run agent tools (e.g. chat.send dispatch).
func CheckToolExecutePermission(session *UserSession) error {
	if session == nil || HasPermission(session, PermToolExecute) {
		return nil
	}
	return fmt.Errorf("forbidden: requires permission %s", PermToolExecute)
}

// RequiredToolPermissions returns permission codes required for the given tool name.
func RequiredToolPermissions(toolName string) []string {
	name := strings.ToLower(strings.TrimSpace(toolName))
	if name == "" {
		return []string{PermToolExecute}
	}
	perms := []string{PermToolExecute}
	switch {
	case isShellTool(name):
		perms = append(perms, PermToolExecuteShell)
	case isWriteTool(name):
		perms = append(perms, PermToolExecuteWrite)
	case isReadTool(name):
		// base tool:execute only
	case isBuiltinTool(name):
		// unknown builtin category: require MCP permission as safe default
		perms = append(perms, PermToolExecuteMCP)
	default:
		perms = append(perms, PermToolExecuteMCP)
	}
	return perms
}

// CheckToolExecution verifies RBAC for a single tool invocation.
func CheckToolExecution(session *UserSession, toolName string) error {
	if session == nil {
		return nil
	}
	for _, code := range RequiredToolPermissions(toolName) {
		if !HasPermission(session, code) {
			return fmt.Errorf("forbidden: tool %q requires permission %s", strings.TrimSpace(toolName), code)
		}
	}
	return nil
}

func isShellTool(name string) bool {
	switch name {
	case "bash", "sh", "shell", "powershell", "pwsh", "cmd", "windows_exec_cmd":
		return true
	default:
		return false
	}
}

func isWriteTool(name string) bool {
	switch name {
	case "write", "edit", "file_write", "file_edit":
		return true
	default:
		return false
	}
}

func isReadTool(name string) bool {
	switch name {
	case "read", "grep", "glob", "file_read":
		return true
	default:
		return false
	}
}

func isBuiltinTool(name string) bool {
	return isShellTool(name) || isWriteTool(name) || isReadTool(name)
}
