package rbac

import "testing"

func TestCheckToolExecution_ViewerDeniedShell(t *testing.T) {
	session := &UserSession{
		RoleName:    "viewer",
		Permissions: []string{"menu:overview"},
	}
	if err := CheckToolExecution(session, "bash"); err == nil {
		t.Fatal("expected bash to be denied for viewer")
	}
}

func TestCheckToolExecution_OperatorAllowedShell(t *testing.T) {
	session := &UserSession{
		RoleName: "gbase_operator",
		Permissions: []string{
			PermToolExecute,
			PermToolExecuteShell,
			PermToolExecuteWrite,
			PermToolExecuteMCP,
		},
	}
	if err := CheckToolExecution(session, "bash"); err != nil {
		t.Fatalf("expected bash allowed: %v", err)
	}
}

func TestCheckToolExecution_ViewerReadOnlyTool(t *testing.T) {
	session := &UserSession{
		RoleName:    "viewer",
		Permissions: []string{"menu:overview"},
	}
	if err := CheckToolExecution(session, "read"); err == nil {
		t.Fatal("expected read to require tool:execute")
	}

	session.Permissions = append(session.Permissions, PermToolExecute)
	if err := CheckToolExecution(session, "read"); err != nil {
		t.Fatalf("expected read allowed with tool:execute: %v", err)
	}
}

func TestCheckToolExecution_NilSessionBypass(t *testing.T) {
	if err := CheckToolExecution(nil, "bash"); err != nil {
		t.Fatalf("nil session should bypass RBAC: %v", err)
	}
}

func TestCheckToolExecution_AdminBypass(t *testing.T) {
	session := &UserSession{RoleName: "admin", Permissions: []string{}}
	if err := CheckToolExecution(session, "bash"); err != nil {
		t.Fatalf("admin should bypass: %v", err)
	}
}

func TestCheckToolExecution_MCPTool(t *testing.T) {
	session := &UserSession{
		RoleName:    "viewer",
		Permissions: []string{PermToolExecute},
	}
	if err := CheckToolExecution(session, "custom_mcp_tool"); err == nil {
		t.Fatal("expected MCP tool to require tool:execute:mcp")
	}
}
