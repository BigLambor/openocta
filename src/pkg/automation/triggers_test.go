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

	// 3. Match GBase alert (any severity)
	empID, matched = MatchAlert("gbase", "warning", "Slow Query Alert")
	if !matched {
		t.Fatalf("expected GBase alert to match")
	}
	if empID != "emp_bch_duty" {
		t.Fatalf("expected GBase to route to emp_bch_duty, got %s", empID)
	}
}
