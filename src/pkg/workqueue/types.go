// Package workqueue implements trigger-agnostic work planning and execution.
package workqueue

const (
	TierL0 = "L0"
	TierL1 = "L1"
	TierL2 = "L2"

	ActionCollectAndScore = "collect_and_score"
	ActionScenarioInspect = "scenario_inspect"
	ActionAIDiagnose      = "ai_diagnose"
	ActionDomainReduce    = "domain_reduce"

	PlanStatusQueued    = "queued"
	PlanStatusRunning   = "running"
	PlanStatusSucceeded = "succeeded"
	PlanStatusPartial   = "partial"
	PlanStatusFailed    = "failed"

	TaskStatusQueued    = "queued"
	TaskStatusRunning   = "running"
	TaskStatusSucceeded = "succeeded"
	TaskStatusFailed    = "failed"
	TaskStatusTimeout   = "timeout"
	TaskStatusCancelled = "cancelled"

	PriorityLow    = 0
	PriorityNormal = 10
	PriorityHigh   = 100
)

// CronJobSnapshot is the serializable subset of a cron job needed for execution.
type CronJobSnapshot struct {
	ID                string `json:"id"`
	AgentID           string `json:"agentId,omitempty"`
	DigitalEmployeeID string `json:"digitalEmployeeId,omitempty"`
	SessionTarget     string `json:"sessionTarget,omitempty"`
	SessionKey        string `json:"sessionKey,omitempty"`
	PayloadKind       string `json:"payloadKind,omitempty"`
	PayloadMessage    string `json:"payloadMessage,omitempty"`
}

// TriggerScope identifies objects in a run.
type TriggerScope struct {
	ObjectType string   `json:"objectType,omitempty"`
	ObjectIDs  []string `json:"objectIds,omitempty"`
	ClusterID  string   `json:"clusterId,omitempty"`
	Domain     string   `json:"domain,omitempty"`
	Component  string   `json:"component,omitempty"`
	TenantID   string   `json:"tenantId,omitempty"`
}

// TriggerEnvelope is the unified trigger payload.
type TriggerEnvelope struct {
	TriggerType     string       `json:"triggerType"`
	TriggerRef      string       `json:"triggerRef"`
	ScenarioKey     string       `json:"scenarioKey"`
	Scope           TriggerScope `json:"scope"`
	Priority        int          `json:"priority"`
	IdempotencyKey  string       `json:"idempotencyKey"`
	ScheduledAtMs   int64        `json:"scheduledAtMs"`
	CronJob         CronJobSnapshot `json:"cronJob,omitempty"`
	CronMode        string       `json:"cronMode,omitempty"`
	Message         string       `json:"message,omitempty"`
	Domain          string       `json:"domain,omitempty"`
	ClusterID       string       `json:"clusterId,omitempty"`
	Component       string       `json:"component,omitempty"`
	ParentRunID     string       `json:"parentRunId,omitempty"`
}

// PlanStep is one tier action inside a work plan.
type PlanStep struct {
	Tier           string   `json:"tier"`
	Action         string   `json:"action"`
	ObjectIDs      []string `json:"objectIds,omitempty"`
	MaxConcurrency int      `json:"maxConcurrency,omitempty"`
}

// WorkPlan is the planner output persisted and executed by the queue.
type WorkPlan struct {
	ID            string         `json:"id"`
	ParentRunID   string         `json:"parentRunId"`
	TriggerType   string         `json:"triggerType"`
	TriggerRef    string         `json:"triggerRef"`
	ScenarioKey   string         `json:"scenarioKey"`
	Status        string         `json:"status"`
	Priority      int            `json:"priority"`
	IdempotencyKey string        `json:"idempotencyKey"`
	ScheduledAtMs int64          `json:"scheduledAtMs"`
	Steps         []PlanStep     `json:"steps"`
	Envelope      TriggerEnvelope `json:"-"`
}

// WorkTask is one executable unit in the queue.
type WorkTask struct {
	ID             string
	PlanID         string
	TenantID       string
	Tier           string
	Action         string
	ObjectType     string
	ObjectID       string
	ParentRunID    string
	ChildRunID     string
	Status         string
	Priority       int
	IdempotencyKey string
	LeaseUntil     int64
	WorkerID       string
	Input          map[string]interface{}
	Output         map[string]interface{}
	Error          string
	CreatedAt      int64
	UpdatedAt      int64
}

// SubmitResult is returned after a plan is submitted and executed.
type SubmitResult struct {
	PlanID      string
	ParentRunID string
	Status      string
	Error       string
	Summary     string
}
