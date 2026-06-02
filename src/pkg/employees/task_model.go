package employees

// EmployeeTask represents a single work task performed by a digital employee.
type EmployeeTask struct {
	ID              string              `json:"id"`
	SessionID       string              `json:"sessionId,omitempty"`
	RunID           string              `json:"runId,omitempty"`
	EmployeeID      string              `json:"employeeId"`
	DomainKey       string              `json:"domainKey"`
	CapabilityKey   string              `json:"capabilityKey"`
	ScenarioKey     string              `json:"scenarioKey"`
	ObjectRef       string              `json:"objectRef"`
	TriggerType     string              `json:"triggerType"`      // e.g. "manual", "cron", "alert"
	ExecutionStatus string              `json:"executionStatus"`  // pending, running, succeeded, failed
	WorkflowStatus  string              `json:"workflowStatus"`   // open, waiting_approval, closed, rejected
	Status          string              `json:"status,omitempty"` // legacy projection for older UI callers
	Input           string              `json:"input"`
	Output          string              `json:"output"`
	Conclusion      string              `json:"conclusion"`
	Artifacts       []string            `json:"artifacts,omitempty"`
	Metrics         EmployeeTaskMetrics `json:"metrics,omitempty"`
	StartedAt       int64               `json:"startedAt"` // Unix timestamp in milliseconds
	FinishedAt      int64               `json:"finishedAt"`
	Operator        string              `json:"operator"`   // "system", "cron", "username", etc.
	Evaluation      string              `json:"evaluation"` // unrated, accepted, rejected
}

type EmployeeTaskMetrics struct {
	RawAlertCount     int     `json:"rawAlertCount,omitempty"`
	ReducedAlertCount int     `json:"reducedAlertCount,omitempty"`
	SavedHours        float64 `json:"savedHours,omitempty"`
	CostUSD           float64 `json:"costUsd,omitempty"`
	MTTAMs            int64   `json:"mttaMs,omitempty"`
	MTTRMs            int64   `json:"mttrMs,omitempty"`
}
