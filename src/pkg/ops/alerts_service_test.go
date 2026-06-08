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

func TestInitAlertsStoreMigratesExistingGroupsWithoutReplacing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "ops", "alerts.json")
	existing := AlertGroup{
		ID:            "existing-alert",
		Domain:        DomainHadoop,
		Title:         "Existing production alert",
		Severity:      "critical",
		Status:        AlertStatusActive,
		OriginalCount: 4,
		ReducedTo:     1,
	}
	if err := saveAlertsStore(path, &alertsStoreFile{Version: 1, Groups: []AlertGroup{existing}}); err != nil {
		t.Fatal(err)
	}

	if err := InitAlertsStore(dir); err != nil {
		t.Fatal(err)
	}

	list := ListAlertGroups(DomainHadoop, "")
	if list.Total != 1 {
		t.Fatalf("expected existing group to be preserved, got %d groups", list.Total)
	}
	got := list.Groups[0]
	if got.ID != existing.ID {
		t.Fatalf("expected existing group %q, got %q", existing.ID, got.ID)
	}
	if got.SuppressionCategory != "none" || got.ReviewStatus != "pending" {
		t.Fatalf("expected migrated defaults, got suppression=%q review=%q", got.SuppressionCategory, got.ReviewStatus)
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

func TestExtractSection(t *testing.T) {
	headings := []string{"根因候选", "判断结论", "影响范围"}

	// Case 1: Markdown header line with extra text
	md1 := "### 根因候选（基于技能规范与 BCH 运维知识库）**：\n当前环境不存在实际运行的 Hadoop 集群。\n## 影响范围\n部分任务失败。"
	res1 := extractSection(md1, headings)
	expected1 := "当前环境不存在实际运行的 Hadoop 集群。"
	if res1 != expected1 {
		t.Fatalf("expected %q, got %q", expected1, res1)
	}

	// Case 2: Inline header bold
	md2 := "**根因候选：** 当前环境不存在实际运行的 Hadoop 集群。\n## 影响范围\n部分任务失败。"
	res2 := extractSection(md2, headings)
	expected2 := "当前环境不存在实际运行的 Hadoop 集群。"
	if res2 != expected2 {
		t.Fatalf("expected %q, got %q", expected2, res2)
	}

	// Case 3: Empty content
	md3 := "### 根因候选"
	res3 := extractSection(md3, headings)
	if res3 != "" {
		t.Fatalf("expected empty, got %q", res3)
	}

	// Case 4: False positive matching heading in sentence followed by real markdown heading
	md4 := "根据当前告警组进行根因候选分析工作。\n\n## 根因候选\n当前环境不存在实际运行的 Hadoop 集群。"
	res4 := extractSection(md4, headings)
	expected4 := "当前环境不存在实际运行的 Hadoop 集群。"
	if res4 != expected4 {
		t.Fatalf("expected %q, got %q", expected4, res4)
	}

	// Case 5: Bold inline header next marker
	md5 := "**根因候选：** 内存溢出。\n\n**可验证依据：** 虚拟机健康指标正常。"
	res5 := extractSection(md5, headings)
	expected5 := "内存溢出。"
	if res5 != expected5 {
		t.Fatalf("expected %q, got %q", expected5, res5)
	}
}

func TestParseReportMarkdownFromText(t *testing.T) {
	// Case 1: Standard markdown with ```json code block containing reportMarkdown
	text1 := "Here is the result:\n```json\n{\n  \"reportMarkdown\": \"### 根因\\n应用连接池占满。\"\n}\n```\nFollow up questions?"
	res1 := parseReportMarkdownFromText(text1)
	expected1 := "### 根因\n应用连接池占满。"
	if res1 != expected1 {
		t.Fatalf("expected %q, got %q", expected1, res1)
	}

	// Case 2: Markdown with standard ``` block containing JSON object with reportMarkdown
	text2 := "Result:\n```\n{\n  \"report_markdown\": \"### 根因\\n应用连接池占满。\"\n}\n```"
	res2 := parseReportMarkdownFromText(text2)
	expected2 := "### 根因\n应用连接池占满。"
	if res2 != expected2 {
		t.Fatalf("expected %q, got %q", expected2, res2)
	}

	// Case 3: Non-JSON standard text
	text3 := "### 根因\n应用连接池占满。"
	res3 := parseReportMarkdownFromText(text3)
	if res3 != text3 {
		t.Fatalf("expected %q, got %q", text3, res3)
	}
}

func TestIsCompleteDiagnosisReport(t *testing.T) {
	// Case 1: Short greeting/intro - should be false
	text1 := "我将作为数据血缘治理数字员工，分析该告警。"
	if isCompleteDiagnosisReport(text1) {
		t.Errorf("expected false for short greeting, got true")
	}

	// Case 2: Structured report - should be true
	text2 := "分析结束。\n## 根因候选\n底层组件故障。"
	if !isCompleteDiagnosisReport(text2) {
		t.Errorf("expected true for structured report, got false")
	}

	// Case 3: Paragraph with bold title but no specific header - should check length
	text3 := "**判断结论：** 这是一个比较详细的诊断建议，应该包含了分析。"
	if !isCompleteDiagnosisReport(text3) {
		t.Errorf("expected true for report containing '判断结论', got false")
	}
}
