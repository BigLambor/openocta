package jobrun

import "time"

const (
	StatusQueued           = "queued"
	StatusRunning          = "running"
	StatusWaitingApproval  = "waiting_approval"
	StatusSucceeded        = "succeeded"
	StatusPartial          = "partial"
	StatusFailed           = "failed"
	StatusCancelled        = "cancelled"
	StatusTimeout          = "timeout"
	TriggerCron            = "cron"
	TriggerManual          = "manual"
	TriggerInspection      = "inspection"
	TriggerAlert           = "alert"
	TriggerEscalation      = "escalation"
)

// JobRun is a persisted execution record for cron, inspection, or other jobs.
type JobRun struct {
	ID          string                 `json:"id"`
	JobID       string                 `json:"jobId"`
	TaskID      string                 `json:"taskId,omitempty"`
	ParentRunID string                 `json:"parentRunId,omitempty"`
	TriggerType string                 `json:"triggerType"`
	TriggerRef  string                 `json:"triggerRef,omitempty"`
	Status      string                 `json:"status"`
	StartedAt   int64                  `json:"startedAt"`
	FinishedAt  int64                  `json:"finishedAt,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Input       map[string]interface{} `json:"input,omitempty"`
	Output      map[string]interface{} `json:"output,omitempty"`
	CreatedAt   int64                  `json:"createdAt"`
	UpdatedAt   int64                  `json:"updatedAt"`
}

// RunStep is one step inside a job run.
type RunStep struct {
	ID            string `json:"id"`
	RunID         string `json:"runId"`
	StepOrder     int    `json:"stepOrder"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Status        string `json:"status"`
	StartedAt     int64  `json:"startedAt"`
	FinishedAt    int64  `json:"finishedAt,omitempty"`
	Error         string `json:"error,omitempty"`
	InputSummary  string `json:"inputSummary,omitempty"`
	OutputSummary string `json:"outputSummary,omitempty"`
}

// StartInput describes a new run.
type StartInput struct {
	RunID       string
	JobID       string
	TaskID      string
	ParentRunID string
	TriggerType string
	TriggerRef  string
	Input       map[string]interface{}
}

// FinishInput describes run completion payload.
type FinishInput struct {
	Output map[string]interface{}
}

// ListFilter narrows job run queries.
type ListFilter struct {
	JobID       string
	ParentRunID string
	TriggerType string
	TriggerRef  string
	Limit       int
}

// RunDetail is a job run with ordered steps and tool invocations.
type RunDetail struct {
	Run             JobRun           `json:"run"`
	Steps           []RunStep        `json:"steps"`
	ToolInvocations []ToolInvocation `json:"toolInvocations,omitempty"`
}

// ToolInvocation is a persisted tool/MCP call audit record.
type ToolInvocation struct {
	ID            string `json:"id"`
	RunID         string `json:"runId"`
	SessionID     string `json:"sessionId,omitempty"`
	StepID        string `json:"stepId,omitempty"`
	ToolName      string `json:"toolName"`
	Provider      string `json:"provider,omitempty"`
	InputSummary  string `json:"inputSummary,omitempty"`
	OutputSummary string `json:"outputSummary,omitempty"`
	Status        string `json:"status"`
	DurationMs    int64  `json:"durationMs,omitempty"`
	Error         string `json:"error,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
}

// ToolExecutionInput describes one tool call to record.
type ToolExecutionInput struct {
	RunID      string
	SessionID  string
	ToolName   string
	Provider   string
	Input      string
	Output     string
	Status     string
	Error      string
	DurationMs int64
}

// ModelUsageInput describes one LLM call to record.
type ModelUsageInput struct {
	RunID        string
	SessionID    string
	Provider     string
	Model        string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	LatencyMs    int64
	Status       string
}

// StepInput describes a run step.
type StepInput struct {
	Kind          string
	Name          string
	Status        string
	InputSummary  string
	OutputSummary string
	Error         string
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
