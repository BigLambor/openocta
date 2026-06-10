package ops

import "testing"

func TestParseChatOpsCommand(t *testing.T) {
	cmd, args, ok := ParseChatOpsCommand("/ack alert-group-abc")
	if !ok || cmd != "ack" || args != "alert-group-abc" {
		t.Fatalf("parse ack: %q %q %v", cmd, args, ok)
	}
	_, _, ok2 := ParseChatOpsCommand("hello")
	if ok2 {
		t.Fatal("expected non-command")
	}
}

func TestChatOpsAck(t *testing.T) {
	initTestAlertsStore(t)
	_, err := RecordMergedAlertGroup("test", "agent:main:alert:x", "run-1", []MergedAlertInput{
		{Title: "t", Severity: "warning", Message: "m"},
	})
	if err != nil {
		t.Fatal(err)
	}
	list := ListAlertGroups("", "")
	if list.Total != 1 {
		t.Fatalf("expected 1 group")
	}
	id := list.Groups[0].ID

	res := HandleChatOpsCommand("ack", id)
	if !res.Handled || res.Reply == "" {
		t.Fatalf("ack failed: %+v", res)
	}
	g, err := GetAlertGroup(id)
	if err != nil || g.Status != AlertStatusResolved {
		t.Fatalf("status: %v err=%v", g.Status, err)
	}
}
