package ops

import "testing"

func TestParseMonitorLabels(t *testing.T) {
	pairs, err := ParseMonitorLabels(`job="hadoop-prod",cluster="bj-bch-prod"`)
	if err != nil {
		t.Fatal(err)
	}
	if len(pairs) != 2 || pairs[0].Key != "job" || pairs[1].Value != "bj-bch-prod" {
		t.Fatalf("unexpected pairs: %+v", pairs)
	}

	if _, err := ParseMonitorLabels(`{"job":"hadoop-prod"}`); err == nil {
		t.Fatal("expected JSON rejection")
	}
	if _, err := ParseMonitorLabels(""); err != nil {
		t.Fatalf("empty should be nil pairs: %v", err)
	}
}

func TestValidateMonitorLabelsForCluster(t *testing.T) {
	if err := ValidateMonitorLabelsForCluster(DomainHadoop, "inactive", ""); err != nil {
		t.Fatalf("inactive empty: %v", err)
	}
	if err := ValidateMonitorLabelsForCluster(DomainHadoop, "healthy", ""); err == nil {
		t.Fatal("expected required labels for healthy")
	}
	if err := ValidateMonitorLabelsForCluster(DomainHadoop, "healthy", `env="prod"`); err == nil {
		t.Fatal("expected domain key requirement")
	}
	if err := ValidateMonitorLabelsForCluster(DomainHadoop, "healthy", `job="hadoop-prod"`); err != nil {
		t.Fatalf("valid hadoop labels: %v", err)
	}
	if err := ValidateMonitorLabelsForCluster(DomainFI, "warning", `cluster="huhe-fi"`); err != nil {
		t.Fatalf("valid fi labels: %v", err)
	}
	if err := ValidateMonitorLabelsForCluster(DomainGBase, "unknown", `instance="gbase-1"`); err != nil {
		t.Fatalf("valid gbase labels: %v", err)
	}
}

func TestNormalizeMonitorLabels(t *testing.T) {
	got, err := NormalizeMonitorLabels(`job=hadoop-prod, cluster=bj`)
	if err != nil {
		t.Fatal(err)
	}
	want := `job="hadoop-prod",cluster="bj"`
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
