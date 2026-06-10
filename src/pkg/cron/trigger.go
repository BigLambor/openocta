package cron

import (
	"fmt"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/workqueue"
)

// TriggerEnvelope builds a unified trigger envelope for work queue submission.
func TriggerEnvelope(job CronJob, mode, domain, clusterID, component, scenarioKey string, scheduledAtMs int64) workqueue.TriggerEnvelope {
	if scheduledAtMs <= 0 {
		scheduledAtMs = time.Now().UnixMilli()
	}
	triggerType := jobrun.TriggerTypeForCronMode(mode)
	priority := workqueue.PriorityLow
	if mode == "force" {
		priority = workqueue.PriorityNormal
	}
	ref := strings.TrimSpace(job.ID)
	idempotency := fmt.Sprintf("cron:%s:%d", ref, scheduledAtMs)
	return workqueue.TriggerEnvelope{
		TriggerType:    triggerType,
		TriggerRef:     ref,
		ScenarioKey:    strings.TrimSpace(scenarioKey),
		Priority:       priority,
		IdempotencyKey: idempotency,
		ScheduledAtMs:  scheduledAtMs,
		CronJob:        snapshotCronJob(job),
		CronMode:       mode,
		Message:        strings.TrimSpace(job.Payload.Message),
		Domain:         strings.TrimSpace(domain),
		ClusterID:      strings.TrimSpace(clusterID),
		Component:      strings.TrimSpace(component),
		Scope: workqueue.TriggerScope{
			ObjectType: "cluster",
			ObjectIDs:  []string{strings.TrimSpace(clusterID)},
			ClusterID:  strings.TrimSpace(clusterID),
			Domain:     strings.TrimSpace(domain),
			Component:  strings.TrimSpace(component),
		},
	}
}

func snapshotCronJob(job CronJob) workqueue.CronJobSnapshot {
	return workqueue.CronJobSnapshot{
		ID:                job.ID,
		AgentID:           job.AgentID,
		DigitalEmployeeID: job.DigitalEmployeeID,
		SessionTarget:     job.SessionTarget,
		SessionKey:        job.SessionKey,
		PayloadKind:       job.Payload.Kind,
		PayloadMessage:    job.Payload.Message,
	}
}
