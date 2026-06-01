package tools

import (
	"testing"
)

func TestInjectLabelsIntoPromQL(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		labels    string
		expect    string
	}{
		{
			name:   "empty labels",
			query:  "avg(up)",
			labels: "",
			expect: "avg(up)",
		},
		{
			name:   "simple query",
			query:  "up",
			labels: `cluster="hadoop-prod"`,
			expect: `up{cluster="hadoop-prod"}`,
		},
		{
			name:   "simple with function",
			query:  "avg(up)",
			labels: `cluster="hadoop-prod"`,
			expect: `avg(up{cluster="hadoop-prod"})`,
		},
		{
			name:   "existing selector",
			query:  `up{job="hadoop"}`,
			labels: `cluster="hadoop-prod"`,
			expect: `up{job="hadoop",cluster="hadoop-prod"}`,
		},
		{
			name:   "complex multiple selector and or",
			query:  `avg(up{job=~".*(hadoop|yarn|hdfs).*"} or up{instance=~".*hadoop.*"})`,
			labels: `cluster="hadoop-prod",env="prod"`,
			expect: `avg(up{job=~".*(hadoop|yarn|hdfs).*",cluster="hadoop-prod",env="prod"} or up{instance=~".*hadoop.*",cluster="hadoop-prod",env="prod"})`,
		},
		{
			name:   "query with by label keyword",
			query:  `sum(fi_yarn_queue_allocated_memory_bytes) by (queue)`,
			labels: `cluster="fi-prod"`,
			expect: `sum(fi_yarn_queue_allocated_memory_bytes{cluster="fi-prod"}) by (queue)`,
		},
		{
			name:   "query with rate and division",
			query:  `rate(governance_api_requests_total{status='200'}[5m]) / rate(governance_api_requests_total[5m])`,
			labels: `env="prod"`,
			expect: `rate(governance_api_requests_total{status='200',env="prod"}[5m]) / rate(governance_api_requests_total{env="prod"}[5m])`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := InjectLabelsIntoPromQL(tc.query, tc.labels)
			if got != tc.expect {
				t.Errorf("InjectLabelsIntoPromQL(%q, %q) = %q; want %q", tc.query, tc.labels, got, tc.expect)
			}
		})
	}
}
