package employees

// EmployeeTask represents a single work task performed by a digital employee.
type EmployeeTask struct {
	ID            string   `json:"id"`
	EmployeeID    string   `json:"employeeId"`
	DomainKey     string   `json:"domainKey"`
	CapabilityKey string   `json:"capabilityKey"`
	ScenarioKey   string   `json:"scenarioKey"`
	ObjectRef     string   `json:"objectRef"`
	TriggerType   string   `json:"triggerType"` // e.g. "manual", "cron", "alert"
	Status        string   `json:"status"`      // e.g. "pending", "running", "success", "failed"
	Input         string   `json:"input"`
	Output        string   `json:"output"`
	Conclusion    string   `json:"conclusion"`
	Artifacts     []string `json:"artifacts,omitempty"`
	StartedAt     int64    `json:"startedAt"` // Unix timestamp in milliseconds
	FinishedAt    int64    `json:"finishedAt"`
	Operator      string   `json:"operator"` // "system", "cron", "username", etc.
	Evaluation    string   `json:"evaluation"` // e.g. "unrated", "accepted", "rejected"
}
