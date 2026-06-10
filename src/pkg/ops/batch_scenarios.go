package ops

import "strings"

const (
	HealthObjectQueue      = "queue"
	HealthObjectDBInstance = "db_instance"
	HealthObjectPipeline   = "pipeline"

	ScenarioSparkHealth             = "ops-spark-health"
	ScenarioYarnHealth              = "ops-yarn-health"
	ScenarioGBaseInstanceHealth     = "ops-gbase-instance-health"
	ScenarioDataAppsPipelineHealth  = "ops-dataapps-pipeline-health"

	SparkDomainSnapshotID            = "hadoop:spark"
	YarnDomainSnapshotID             = "hadoop:yarn"
	GBaseInstancesDomainSnapshotID   = "gbase:instances"
	DataAppsPipelinesDomainSnapshotID = "dataapps:pipelines"
)

// IsBatchL0Scenario reports whether a scenario runs L0-only batch collection via Work Queue.
func IsBatchL0Scenario(scenarioKey string) bool {
	switch strings.TrimSpace(scenarioKey) {
	case ScenarioFlinkHealth, ScenarioSparkHealth, ScenarioYarnHealth,
		ScenarioGBaseInstanceHealth, ScenarioDataAppsPipelineHealth:
		return true
	default:
		return false
	}
}
