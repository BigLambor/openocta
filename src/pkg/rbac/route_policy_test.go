package rbac

import "testing"

func TestLookupHTTPRouteOpsAck(t *testing.T) {
	spec, ok := LookupHTTPRoute("PATCH", "/api/ops/alerts/groups/g1")
	if !ok {
		t.Fatal("expected route match")
	}
	if spec.Permission != "ops:ack" {
		t.Fatalf("permission = %q, want ops:ack", spec.Permission)
	}
}

func TestAuthorizeMethodChatSend(t *testing.T) {
	viewer := &UserSession{RoleName: "viewer", Permissions: []string{PermSessionRead}}
	if err := AuthorizeMethod(viewer, "chat.send", false); err == nil {
		t.Fatal("viewer should not invoke chat.send")
	}
	operator := &UserSession{RoleName: "gbase_operator", Permissions: []string{PermToolExecute}}
	if err := AuthorizeMethod(operator, "chat.send", false); err != nil {
		t.Fatalf("operator chat.send: %v", err)
	}
}

func TestAuthorizeMethodLegacyGatewayToken(t *testing.T) {
	if err := AuthorizeMethod(nil, "config.get", true); err != nil {
		t.Fatalf("legacy gateway token should bypass: %v", err)
	}
	if err := AuthorizeMethod(nil, "health", true); err != nil {
		t.Fatalf("legacy gateway token health: %v", err)
	}
}
