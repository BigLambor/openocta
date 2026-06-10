package ops

import "testing"

func TestShouldEscalateFlinkSignal(t *testing.T) {
	policy := FlinkEscalationPolicy()

	healthy := HealthSignal{
		ObjectType: HealthObjectJob,
		Type:       SignalTypeBCHWorkload,
		Status:     HealthStatusHealthy,
		Score:      intPtr(95),
	}
	if ShouldEscalateFlinkSignal(healthy, policy) {
		t.Fatal("healthy job should not escalate")
	}

	lowScore := HealthSignal{
		ObjectType: HealthObjectJob,
		Type:       SignalTypeBCHWorkload,
		Status:     HealthStatusWarning,
		Score:      intPtr(65),
	}
	if !ShouldEscalateFlinkSignal(lowScore, policy) {
		t.Fatal("low score should escalate")
	}

	restarts := HealthSignal{
		ObjectType: HealthObjectJob,
		Type:       SignalTypeBCHWorkload,
		Status:     HealthStatusHealthy,
		Score:      intPtr(90),
		Evidence: map[string]interface{}{
			"metrics": map[string]interface{}{"restarts": float64(2)},
		},
	}
	if !ShouldEscalateFlinkSignal(restarts, policy) {
		t.Fatal("restarts should escalate")
	}
}

func intPtr(v int) *int { return &v }
