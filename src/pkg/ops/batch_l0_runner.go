package ops

import (
	"context"
	"fmt"
)

// RunBatchL0 executes the L0 collector for a batch scenario.
func RunBatchL0(ctx context.Context, scenarioKey string, opts BatchL0Opts) error {
	switch scenarioKey {
	case ScenarioFlinkHealth:
		_, err := RunFlinkHealthL0(ctx, FlinkL0Opts{
			RunID: opts.RunID, ScenarioKey: scenarioKey, Domain: opts.Domain, ClusterID: opts.ClusterID,
		})
		return err
	case ScenarioSparkHealth:
		_, err := RunSparkHealthL0(ctx, SparkL0Opts{
			RunID: opts.RunID, ScenarioKey: scenarioKey, Domain: opts.Domain, ClusterID: opts.ClusterID,
		})
		return err
	case ScenarioYarnHealth:
		_, err := RunYarnHealthL0(ctx, YarnL0Opts{
			RunID: opts.RunID, ScenarioKey: scenarioKey, Domain: opts.Domain, ClusterID: opts.ClusterID,
		})
		return err
	case ScenarioGBaseInstanceHealth:
		_, err := RunGBaseInstanceHealthL0(ctx, GBaseInstanceL0Opts{
			RunID: opts.RunID, ScenarioKey: scenarioKey, Domain: opts.Domain,
		})
		return err
	case ScenarioDataAppsPipelineHealth:
		_, err := RunDataAppsPipelineHealthL0(ctx, DataAppsPipelineL0Opts{
			RunID: opts.RunID, ScenarioKey: scenarioKey, Domain: opts.Domain,
		})
		return err
	default:
		return fmt.Errorf("unsupported batch L0 scenario %s", scenarioKey)
	}
}

// BatchL0Opts is shared input for batch L0 runners.
type BatchL0Opts struct {
	RunID       string
	Domain      string
	ClusterID   string
}
