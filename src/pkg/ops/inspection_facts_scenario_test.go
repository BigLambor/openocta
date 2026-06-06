package ops

import "testing"

func TestScenarioKeyForInspection(t *testing.T) {
	cases := []struct {
		jobID  string
		domain string
		want   string
	}{
		{jobID: "job-inspect-hadoop", want: "ops-bch-health"},
		{jobID: "job-inspect-fi", want: "ops-fi-health"},
		{jobID: "job-inspect-gbase", want: "ops-gbase-health"},
		{jobID: "job-inspect-governance", want: "ops-governance-health"},
		{jobID: "job-inspect-dataapps", want: "ops-dataapps-health"},
		{domain: DomainHadoop, want: "ops-bch-health"},
	}

	for _, tc := range cases {
		got := ScenarioKeyForInspection(InspectionReport{JobID: tc.jobID, Domain: tc.domain})
		if got != tc.want {
			t.Fatalf("ScenarioKeyForInspection(%+v) = %q, want %q", tc, got, tc.want)
		}
	}
}
