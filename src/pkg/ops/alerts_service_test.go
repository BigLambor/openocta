package ops

import (
	"path/filepath"
	"testing"
)

func TestRecordAndListAlertGroups(t *testing.T) {
	dir := t.TempDir()
	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}

	g, err := RecordMergedAlertGroup("hadoop-prod", "agent:main:alert:abc", "run-1", []MergedAlertInput{
		{
			Title:     "YARN 队列满",
			Severity:  "critical",
			Message:   "queue full",
			Alertname: "YarnQueueFull",
			Service:   "yarn-rm",
			Instance:  "rm-1",
			ClusterID: "hadoop-cluster-1",
			Component: "yarn",
		},
		{
			Title:     "YARN 队列满",
			Severity:  "warning",
			Message:   "retry",
			Alertname: "YarnQueueFull",
			Service:   "yarn-rm",
			Instance:  "rm-1",
			ClusterID: "hadoop-cluster-1",
			Component: "yarn",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if g.Alertname != "YarnQueueFull" || g.Service != "yarn-rm" || g.ClusterID != "hadoop-cluster-1" {
		t.Fatalf("fields not parsed correctly: %+v", g)
	}

	if len(g.Timeline) == 0 {
		t.Fatalf("expected timeline event on creation")
	}

	list := ListAlertGroups(DomainHadoop, "")
	if list.Total != 1 || list.OriginalTotal != 2 {
		t.Fatalf("unexpected list: %+v", list)
	}
	if list.PendingActive != 1 {
		t.Fatalf("expected pending 1, got %d", list.PendingActive)
	}

	alertsPath = filepath.Join(dir, "ops", "alerts.json")
	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}
	list2 := ListAlertGroups(DomainHadoop, "")
	if list2.Total != 1 {
		t.Fatalf("reload expected 1 group, got %d", list2.Total)
	}
}

func TestPatchAlertGroupValidationAndTimeline(t *testing.T) {
	dir := t.TempDir()
	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}

	g, err := RecordMergedAlertGroup("hadoop-prod", "agent:main:alert:abc", "run-1", []MergedAlertInput{
		{Title: "Alert", Severity: "critical", Message: "msg"},
	})
	if err != nil {
		t.Fatal(err)
	}

	// 1. Enforce validation: status = resolved requires AckNote or ResolvedReason
	resolvedStatus := AlertStatusResolved
	_, err = PatchAlertGroup(g.ID, AlertGroupPatch{Status: &resolvedStatus}, "admin")
	if err == nil {
		t.Fatalf("expected error when patching status to resolved without notes or reason")
	}

	// 2. Successful patch with note
	note := "fixed RM config"
	gPatched, err := PatchAlertGroup(g.ID, AlertGroupPatch{
		Status:  &resolvedStatus,
		AckNote: &note,
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}

	if gPatched.Status != AlertStatusResolved || gPatched.AckNote != "fixed RM config" {
		t.Fatalf("patch failed: %+v", gPatched)
	}

	// 3. Verify timeline entries
	if len(gPatched.Timeline) < 3 { // creation + status_change + ack_note
		t.Fatalf("expected timeline entries, got %d", len(gPatched.Timeline))
	}
	lastEvent := gPatched.Timeline[len(gPatched.Timeline)-1]
	if lastEvent.Operator != "admin" || lastEvent.Type != "ack_note" {
		t.Fatalf("unexpected last event: %+v", lastEvent)
	}
}
