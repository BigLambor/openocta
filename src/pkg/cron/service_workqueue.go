package cron

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/workqueue"
)

func shouldUseWorkQueue(job CronJob) bool {
	if job.SessionTarget == "main" && job.Payload.Kind == "systemEvent" {
		return false
	}
	return job.SessionTarget == "isolated" && job.Payload.Kind == "agentTurn"
}

func (s *Service) executeViaWorkQueue(
	job CronJob,
	mode, domain, clusterID, component, scenarioKey string,
	scheduledAtMs int64,
) (trackedRunID, status, errMsg, runSummary string, inspectionResult *ops.InspectionResult) {
	wq := workqueue.Default()
	if wq == nil {
		status = "error"
		errMsg = "work queue 未初始化"
		return "", status, errMsg, "", nil
	}

	env := TriggerEnvelope(job, mode, domain, clusterID, component, scenarioKey, scheduledAtMs)
	env.Message = strings.TrimSpace(job.Payload.Message)

	ctx, cancel := context.WithTimeout(context.Background(), 35*time.Minute)
	defer cancel()

	result, err := wq.SubmitAndWait(ctx, env)
	if err != nil {
		status = "error"
		errMsg = err.Error()
		return "", status, errMsg, "", nil
	}

	trackedRunID = result.ParentRunID
	switch result.Status {
	case workqueue.PlanStatusSucceeded:
		status = "ok"
		runSummary = "work plan succeeded"
	case workqueue.PlanStatusPartial:
		status = "partial"
		errMsg = result.Error
		runSummary = "work plan partial"
	default:
		status = "error"
		errMsg = firstNonEmpty(result.Error, "work plan failed")
		runSummary = errMsg
	}
	return trackedRunID, status, errMsg, runSummary, nil
}

func scheduledAtForJob(job CronJob, fallbackMs int64) int64 {
	if job.State.NextRunAtMs != nil && *job.State.NextRunAtMs > 0 {
		return *job.State.NextRunAtMs
	}
	return fallbackMs
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func mapWorkQueueStatusToCronUpdate(status string, consecutive int) (newStatus string, newConsecutive int) {
	switch status {
	case "ok", "partial":
		return status, 0
	case "error":
		return status, consecutive + 1
	default:
		return status, consecutive
	}
}

func cronFinishSummary(status, runSummary string) string {
	if strings.TrimSpace(runSummary) != "" {
		return runSummary
	}
	if status == "partial" {
		return "巡检部分完成"
	}
	if status == "ok" {
		return "巡检完成"
	}
	return fmt.Sprintf("巡检状态：%s", status)
}
