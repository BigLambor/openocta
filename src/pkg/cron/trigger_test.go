package cron

import "testing"

func TestTriggerEnvelopeIdempotencyUsesScheduledAt(t *testing.T) {
	job := CronJob{ID: "job-inspect-fi"}
	env := TriggerEnvelope(job, "due", "fi", "cluster-a", "", "", 1_700_000_123_000)
	if env.IdempotencyKey != "cron:job-inspect-fi:1700000123000" {
		t.Fatalf("unexpected key: %s", env.IdempotencyKey)
	}
	if env.CronJob.PayloadKind != "" && env.CronJob.ID != "job-inspect-fi" {
		t.Fatalf("unexpected snapshot: %+v", env.CronJob)
	}
}
