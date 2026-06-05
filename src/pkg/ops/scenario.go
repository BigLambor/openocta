package ops

// OpsScenario declares the executable assembly of skills, tools, and facts.
type OpsScenario struct {
	Key              string   `json:"key"`
	DomainKey        string   `json:"domainKey"`
	CapabilityKey    string   `json:"capabilityKey"`
	ObjectType       string   `json:"objectType"`
	SkillIDs         []string `json:"skillIds"`
	PlatformToolKeys []string `json:"platformToolKeys"`
	MCPServerKeys    []string `json:"mcpServerKeys"`
	RequiredSources  []string `json:"requiredSources"`
	OptionalSources  []string `json:"optionalSources"`
	InputSources     []string `json:"inputSources"`
	OutputSchema     string   `json:"outputSchema"`
	TriggerTypes     []string `json:"triggerTypes"`
	EmployeeIDs      []string `json:"employeeIds"`
}

var builtinOpsScenarios = []OpsScenario{
	{
		Key:              "ops-gbase-health",
		DomainKey:        DomainGBase,
		CapabilityKey:    "health-inspection",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-gbase-health"},
		PlatformToolKeys: []string{"query_gbase_slow_sql", "query_vm_metrics"},
		RequiredSources:  []string{SignalTypeGBaseSQL},
		OptionalSources:  []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		InputSources:     []string{SignalTypeGBaseSQL, SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"manual", "cron", "chat_intent"},
	},
	{
		Key:              "ops-bch-health",
		DomainKey:        DomainHadoop,
		CapabilityKey:    "health-inspection",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-bch-health"},
		PlatformToolKeys: []string{"query_vm_metrics"},
		RequiredSources:  []string{SignalTypeMetrics, SignalTypeJMX, SignalTypeBCHWorkload},
		OptionalSources:  []string{SignalTypeAlerts, SignalTypeInspection, SignalTypeAssetStatus},
		InputSources:     []string{SignalTypeMetrics, SignalTypeJMX, SignalTypeBCHWorkload, SignalTypeAlerts, SignalTypeInspection},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"manual", "cron", "chat_intent"},
	},
	{
		Key:              "ops-fi-health",
		DomainKey:        DomainFI,
		CapabilityKey:    "health-inspection",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-fi-health"},
		PlatformToolKeys: []string{"query_vm_metrics"},
		RequiredSources:  []string{SignalTypeFIManager},
		OptionalSources:  []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection, SignalTypeAssetStatus},
		InputSources:     []string{SignalTypeFIManager, SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"manual", "cron", "chat_intent"},
	},
	{
		Key:              "ops-governance-health",
		DomainKey:        DomainGovernance,
		CapabilityKey:    "health-inspection",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-governance-health"},
		PlatformToolKeys: []string{"query_vm_metrics", "query_governance_lineage"},
		RequiredSources:  []string{SignalTypeGovernanceAPI, SignalTypeMetrics},
		OptionalSources:  []string{SignalTypeAlerts, SignalTypeInspection},
		InputSources:     []string{SignalTypeGovernanceAPI, SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"manual", "cron", "chat_intent"},
	},
	{
		Key:              "ops-dataapps-health",
		DomainKey:        DomainDataApps,
		CapabilityKey:    "health-inspection",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-dataapps-health"},
		PlatformToolKeys: []string{"query_vm_metrics"},
		RequiredSources:  []string{SignalTypeSchedulerAPI},
		OptionalSources:  []string{SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		InputSources:     []string{SignalTypeSchedulerAPI, SignalTypeMetrics, SignalTypeAlerts, SignalTypeInspection},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"manual", "cron", "chat_intent"},
	},
	{
		Key:              "ops-diagnosis",
		DomainKey:        "",
		CapabilityKey:    "diagnosis",
		ObjectType:       HealthObjectCluster,
		SkillIDs:         []string{"ops-diagnosis"},
		PlatformToolKeys: []string{"query_vm_metrics", "query_logs"},
		RequiredSources:  []string{},
		OptionalSources:  []string{SignalTypeMetrics, SignalTypeAlerts},
		InputSources:     []string{SignalTypeMetrics, SignalTypeAlerts},
		OutputSchema:     "InspectionReport",
		TriggerTypes:     []string{"alert_hook", "chat_intent"},
	},
}

func ListOpsScenarios() []OpsScenario {
	out := make([]OpsScenario, len(builtinOpsScenarios))
	copy(out, builtinOpsScenarios)
	return out
}

func GetOpsScenario(key string) (OpsScenario, bool) {
	for _, s := range builtinOpsScenarios {
		if s.Key == key {
			return s, true
		}
	}
	return OpsScenario{}, false
}
