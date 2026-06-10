package ops

import (
	"context"
	"fmt"
	"strings"
)

// Flink VM metric names (Flink Prometheus Reporter; label: job_id).
const (
	flinkPromQLLagMax         = `max by (job_id) (flink_taskmanager_job_task_operator_records_lag_max)`
	flinkPromQLRestarts1h     = `sum by (job_id) (increase(flink_jobmanager_job_numRestarts[1h]))`
	flinkPromQLBackpressure   = `max by (job_id) (flink_task_isBackPressured)`
	flinkPromQLCPUMax         = `max by (job_id) (flink_taskmanager_Status_JVM_CPU_Load * 100)`
)

type flinkVMMetrics struct {
	byJob  map[string]FlinkMetricInput
	source string
}

func collectFlinkMetricsFromVM(ctx context.Context, clusterID string) (flinkVMMetrics, error) {
	out := flinkVMMetrics{byJob: map[string]FlinkMetricInput{}, source: "vm"}
	client := newVMClient()
	if !client.configured() {
		return out, fmt.Errorf("vm not configured")
	}
	if clusterID != "" {
		if c, err := GetCluster(clusterID); err == nil {
			if u := resolveClusterMetricsBaseURL(c); u != "" {
				client = &vmClient{baseURL: u, http: client.http}
			}
		}
	}

	labelFilter := ""
	if clusterID != "" {
		if c, err := GetCluster(clusterID); err == nil {
			labelFilter = strings.TrimSpace(c.MonitorLabels)
		}
	}

	type querySpec struct {
		query string
		apply func(*FlinkMetricInput, float64)
	}
	specs := []querySpec{
		{flinkPromQLLagMax, func(m *FlinkMetricInput, v float64) {
			m.MaxLag = int64(v)
			if m.AvgLag == 0 {
				m.AvgLag = int64(v)
			}
			if v > 0 {
				m.LagTrend = 1
			}
		}},
		{flinkPromQLRestarts1h, func(m *FlinkMetricInput, v float64) { m.Restarts = int(v) }},
		{flinkPromQLBackpressure, func(m *FlinkMetricInput, v float64) { m.IsBP = v > 0 }},
		{flinkPromQLCPUMax, func(m *FlinkMetricInput, v float64) {
			m.CpuMax = int(v)
			m.CpuAvg = int(v * 0.8)
		}},
	}

	for _, spec := range specs {
		query := spec.query
		if labelFilter != "" {
			query = injectLabelsIntoPromQL(query, labelFilter)
		}
		series, err := client.queryInstantByLabel(ctx, query, "job_id")
		if err != nil {
			continue
		}
		for jobID, val := range series {
			jobID = strings.TrimSpace(jobID)
			if jobID == "" {
				continue
			}
			cur := out.byJob[jobID]
			spec.apply(&cur, val)
			out.byJob[jobID] = cur
		}
	}
	if len(out.byJob) == 0 {
		return out, fmt.Errorf("vm returned no flink job series")
	}
	return out, nil
}

func injectLabelsIntoPromQL(query, labels string) string {
	labels = strings.TrimSpace(labels)
	if labels == "" {
		return query
	}
	open := strings.Index(query, "(")
	if open < 0 {
		return query
	}
	close := strings.Index(query[open:], ")")
	if close <= 0 {
		return query
	}
	inner := query[open+1 : open+close]
	if strings.Contains(inner, "{") {
		inner = strings.TrimSuffix(strings.TrimSpace(inner), "}")
		sep := ","
		if strings.HasSuffix(inner, "{") || strings.HasSuffix(inner, ",") {
			sep = ""
		}
		inner = inner + sep + labels + "}"
	} else {
		inner = inner + "{" + labels + "}"
	}
	return query[:open+1] + inner + query[open+close:]
}
