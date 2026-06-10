package jobrun

import "strings"

// RecordToolExecution writes run_steps + tool_invocations when runID maps to an active JobRun.
func RecordToolExecution(input ToolExecutionInput) {
	svc := Default()
	if svc == nil || svc.repo == nil {
		return
	}
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		return
	}
	if _, err := svc.Get(runID); err != nil {
		return
	}

	toolName := strings.TrimSpace(input.ToolName)
	if toolName == "" {
		toolName = "unknown_tool"
	}
	provider := strings.TrimSpace(input.Provider)
	if provider == "" {
		provider = inferToolProvider(toolName)
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = StatusSucceeded
	}
	inputSummary := SummarizePayload(input.Input)
	outputSummary := SummarizePayload(input.Output)
	errText := RedactText(strings.TrimSpace(input.Error))

	step, err := svc.AddStep(runID, StepInput{
		Kind:          "tool",
		Name:          toolName,
		Status:        status,
		InputSummary:  inputSummary,
		OutputSummary: outputSummary,
		Error:         errText,
	})
	if err != nil {
		return
	}

	inv := ToolInvocation{
		ID:            step.ID,
		RunID:         runID,
		SessionID:     strings.TrimSpace(input.SessionID),
		StepID:        step.ID,
		ToolName:      toolName,
		Provider:      provider,
		InputSummary:  inputSummary,
		OutputSummary: outputSummary,
		Status:        status,
		DurationMs:    input.DurationMs,
		Error:         errText,
		CreatedAt:     step.StartedAt,
	}
	_ = svc.repo.InsertToolInvocation(inv)
}

// RecordModelUsage writes model_usage when runID maps to a JobRun.
func RecordModelUsage(input ModelUsageInput) {
	svc := Default()
	if svc == nil || svc.repo == nil {
		return
	}
	runID := strings.TrimSpace(input.RunID)
	if runID == "" {
		return
	}
	if _, err := svc.Get(runID); err != nil {
		return
	}
	if input.InputTokens == 0 && input.OutputTokens == 0 && input.TotalTokens == 0 {
		return
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = StatusSucceeded
	}
	_ = svc.repo.InsertModelUsage(input, status)
}
