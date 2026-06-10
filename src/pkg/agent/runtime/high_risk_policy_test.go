package runtime

import "testing"

func TestApplyCommercialHighRiskDefaultsAskRestart(t *testing.T) {
	policy := ApplyCommercialHighRiskDefaults(&ResolvedCommandPolicy{
		Enabled:       true,
		DefaultPolicy: "ask",
	})
	if policy.EvaluateCommand("kubectl rollout restart deployment/foo") != "ask" {
		t.Fatalf("expected ask for restart command, got %q", policy.EvaluateCommand("kubectl rollout restart deployment/foo"))
	}
}

func TestApplyCommercialHighRiskDefaultsDenyDestructive(t *testing.T) {
	policy := ApplyCommercialHighRiskDefaults(&ResolvedCommandPolicy{
		Enabled:       true,
		DefaultPolicy: "ask",
	})
	if policy.EvaluateCommand("rm -rf /tmp/safe") != "deny" {
		t.Fatalf("expected deny for rm -rf / fragment match")
	}
}
