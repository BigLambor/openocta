package automation

import (
	"testing"
)

func TestMatchAlert(t *testing.T) {
	// 1. Match BCH critical alert
	empID, matched := MatchAlert("hadoop", "critical", "NameNode GC Pause Alert")
	if !matched {
		t.Fatalf("expected BCH critical alert to match")
	}
	if empID != "emp_bch_duty" {
		t.Fatalf("expected employee ID to be emp_bch_duty, got %s", empID)
	}

	// 2. Do not match info level alert
	_, matched = MatchAlert("hadoop", "info", "Routine Heartbeat")
	if matched {
		t.Fatalf("expected info level alert not to match")
	}

	// 3. Match GBase alert (any severity) → routes to the GBase expert
	empID, matched = MatchAlert("gbase", "warning", "Slow Query Alert")
	if !matched {
		t.Fatalf("expected GBase alert to match")
	}
	if empID != "emp_gbase_diagnose" {
		t.Fatalf("expected GBase to route to emp_gbase_diagnose, got %s", empID)
	}

	// 4. Governance and data-app domains route to their own experts
	if empID, matched = MatchAlert("governance", "warning", "Lineage Broken"); !matched || empID != "emp_governance_remediate" {
		t.Fatalf("expected governance to route to emp_governance_remediate, got %s (matched=%v)", empID, matched)
	}
	if empID, matched = MatchAlert("dataapps", "warning", "SLA Breach"); !matched || empID != "emp_dataapps_ops" {
		t.Fatalf("expected dataapps to route to emp_dataapps_ops, got %s (matched=%v)", empID, matched)
	}

	// 5. DefaultEmployeeForDomain exposes the same mapping
	if id, ok := DefaultEmployeeForDomain("HADOOP"); !ok || id != "emp_bch_duty" {
		t.Fatalf("expected DefaultEmployeeForDomain(hadoop)=emp_bch_duty, got %s (ok=%v)", id, ok)
	}
}
