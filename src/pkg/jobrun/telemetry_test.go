package jobrun

import "testing"

func TestRecordToolExecutionWritesStepAndInvocation(t *testing.T) {
	svc := initTestJobRunService(t)
	run, err := svc.Start(StartInput{
		JobID:       "alert-diagnosis",
		TriggerType: TriggerAlert,
		TriggerRef:  "alert-group-1",
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	RecordToolExecution(ToolExecutionInput{
		RunID:      run.ID,
		SessionID:  "session-1",
		ToolName:   "query_vm_metrics",
		Input:      `{"password":"secret","query":"up"}`,
		Output:     `{"status":"ok"}`,
		Status:     StatusSucceeded,
		DurationMs: 120,
	})

	steps, err := svc.ListSteps(run.ID)
	if err != nil || len(steps) != 1 {
		t.Fatalf("steps: %+v err=%v", steps, err)
	}
	if steps[0].InputSummary == "" || steps[0].Name != "query_vm_metrics" {
		t.Fatalf("unexpected step: %+v", steps[0])
	}
	if contains(steps[0].InputSummary, "secret") {
		t.Fatalf("expected redacted input, got %q", steps[0].InputSummary)
	}

	detail, err := svc.GetDetail(run.ID)
	if err != nil || len(detail.ToolInvocations) != 1 {
		t.Fatalf("detail invocations: %+v err=%v", detail.ToolInvocations, err)
	}
}

func TestRecordToolExecutionSkipsUnknownRun(t *testing.T) {
	svc := initTestJobRunService(t)
	if svc == nil {
		t.Fatal("expected service")
	}
	RecordToolExecution(ToolExecutionInput{
		RunID:    "missing-run",
		ToolName: "noop",
		Output:   "ok",
	})
}

func TestRecordModelUsage(t *testing.T) {
	svc := initTestJobRunService(t)
	run, err := svc.Start(StartInput{JobID: "chat", TriggerType: TriggerManual})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	RecordModelUsage(ModelUsageInput{
		RunID:        run.ID,
		SessionID:    "s1",
		Provider:     "openai",
		Model:        "gpt-4",
		InputTokens:  10,
		OutputTokens: 20,
		LatencyMs:    800,
	})
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexString(s, sub) >= 0)
}

func indexString(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
