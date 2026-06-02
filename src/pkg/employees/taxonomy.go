package employees

import "strings"

const (
	DomainHadoop     = "hadoop"
	DomainFI         = "fi"
	DomainGBase      = "gbase"
	DomainGovernance = "governance"
	DomainDataApps   = "dataapps"
)

const (
	CapabilityObservabilityAlert      = "observability-alert"
	CapabilityHealthInspection        = "health-inspection"
	CapabilityDiagnosisIncident       = "diagnosis-incident"
	CapabilityGovernanceOptimization  = "governance-optimization"
	CapabilityCapacityPerformanceCost = "capacity-performance-cost"
	CapabilityChangeConfigCompliance  = "change-config-compliance"
)

const (
	ExecutionPending   = "pending"
	ExecutionRunning   = "running"
	ExecutionSucceeded = "succeeded"
	ExecutionFailed    = "failed"
)

const (
	WorkflowOpen            = "open"
	WorkflowWaitingApproval = "waiting_approval"
	WorkflowClosed          = "closed"
	WorkflowRejected        = "rejected"
)

const (
	EvaluationUnrated  = "unrated"
	EvaluationAccepted = "accepted"
	EvaluationRejected = "rejected"
)

var CanonicalDomainKeys = []string{
	DomainHadoop,
	DomainFI,
	DomainGBase,
	DomainGovernance,
	DomainDataApps,
}

var CanonicalCapabilityKeys = []string{
	CapabilityObservabilityAlert,
	CapabilityHealthInspection,
	CapabilityDiagnosisIncident,
	CapabilityGovernanceOptimization,
	CapabilityCapacityPerformanceCost,
	CapabilityChangeConfigCompliance,
}

func NormalizeDomainKey(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "bch", "bigdata", "hadoop-ecosystem":
		return DomainHadoop
	case DomainHadoop:
		return DomainHadoop
	case DomainFI, "fusioninsight":
		return DomainFI
	case DomainGBase:
		return DomainGBase
	case DomainGovernance, "dev-governance", "development-governance":
		return DomainGovernance
	case DomainDataApps, "data-apps", "data_apps", "dataapp":
		return DomainDataApps
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func NormalizeCapabilityKey(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "observability", "alerts", "alert":
		return CapabilityObservabilityAlert
	case CapabilityObservabilityAlert:
		return CapabilityObservabilityAlert
	case "inspection", "health", "巡检":
		return CapabilityHealthInspection
	case CapabilityHealthInspection:
		return CapabilityHealthInspection
	case "diagnosis", "incident":
		return CapabilityDiagnosisIncident
	case CapabilityDiagnosisIncident:
		return CapabilityDiagnosisIncident
	case "governance", "optimization":
		return CapabilityGovernanceOptimization
	case CapabilityGovernanceOptimization:
		return CapabilityGovernanceOptimization
	case "capacity", "performance", "cost":
		return CapabilityCapacityPerformanceCost
	case CapabilityCapacityPerformanceCost:
		return CapabilityCapacityPerformanceCost
	case "change", "config", "compliance":
		return CapabilityChangeConfigCompliance
	case CapabilityChangeConfigCompliance:
		return CapabilityChangeConfigCompliance
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func NormalizeExecutionStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "pending":
		return ExecutionPending
	case "running":
		return ExecutionRunning
	case "success", "succeeded", "ok", "closed":
		return ExecutionSucceeded
	case "failed", "failure", "error":
		return ExecutionFailed
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func NormalizeWorkflowStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "open":
		return WorkflowOpen
	case "waiting", "waitingapproval", "waiting_approval", "pending_approval":
		return WorkflowWaitingApproval
	case "closed", "completed", "success":
		return WorkflowClosed
	case "rejected", "dismissed":
		return WorkflowRejected
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func LegacyStatusFromExecution(status string) string {
	switch NormalizeExecutionStatus(status) {
	case ExecutionSucceeded:
		return "success"
	case ExecutionFailed:
		return "failed"
	case ExecutionRunning:
		return "running"
	default:
		return "pending"
	}
}

func ExecutionFromLegacyStatus(status string) string {
	return NormalizeExecutionStatus(status)
}
