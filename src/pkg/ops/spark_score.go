package ops

import "strings"

// SparkScoredJob holds rule-based Spark scoring output.
type SparkScoredJob struct {
	ID       string
	Name     string
	Owner    string
	Cluster  string
	Status   string
	Score    int
	Labels   []string
	Metrics  SparkJobMetric
	Diagnosis string
}

// ScoreSparkJob applies rule-based scoring for Spark batch L0.
func ScoreSparkJob(job SparkJob) SparkScoredJob {
	score := 100
	diagnosis := "作业运行正常。"
	status := strings.ToUpper(strings.TrimSpace(job.Status))

	switch status {
	case "FAILED":
		score = 25
		diagnosis = "作业失败，需排查 Executor/Driver 日志。"
	case "RUNNING":
		if job.Metrics.FailedTasks > 0 {
			score -= minInt(job.Metrics.FailedTasks*4, 40)
			diagnosis = "运行中存在失败 Task，可能存在节点或数据问题。"
		}
	}

	if job.Metrics.CpuSkewRatio > 5 {
		score -= 15
		diagnosis = "检测到 CPU 倾斜，单 Task 热点明显。"
	}
	if job.Metrics.MemorySkewRatio > 4 {
		score -= 15
		if diagnosis == "作业运行正常。" {
			diagnosis = "检测到内存倾斜，Shuffle 阶段可能不均衡。"
		}
	}
	if job.Metrics.AvgTaskDurationSec > 0 && job.Metrics.MaxTaskDurationSec > job.Metrics.AvgTaskDurationSec*10 {
		score -= 10
		diagnosis = "存在长尾 Task，建议开启推测执行或排查慢节点。"
	}
	if score < 0 {
		score = 0
	}

	return SparkScoredJob{
		ID:        job.ID,
		Name:      job.Name,
		Owner:     job.Owner,
		Cluster:   job.Cluster,
		Status:    status,
		Score:     score,
		Labels:    job.Labels,
		Metrics:   job.Metrics,
		Diagnosis: diagnosis,
	}
}

func sparkStatusFromScore(score int) string {
	switch {
	case score >= 85:
		return HealthStatusHealthy
	case score >= 70:
		return HealthStatusWarning
	default:
		return HealthStatusCritical
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
