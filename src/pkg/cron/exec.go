package cron

// Deps are dependencies for executing cron jobs (injected by the gateway).
type Deps struct {
	// EnqueueSystemEvent enqueues a system event for the main session.
	EnqueueSystemEvent func(text string)
	// RequestHeartbeatNow requests an immediate heartbeat run.
	RequestHeartbeatNow func(reason string)
	// RunIsolatedAgentJob runs one isolated agent turn (sessionKey = cron-jobId).
	// Deprecated: prefer RunCronChat so that cron runs go through chat.send and
	// produce proper transcripts and session store entries.
	RunIsolatedAgentJob func(job CronJob, message string)
	// RunCronChat triggers a chat.send-style run for a cron job, using the
	// provided sessionKey, sessionId and message. idempotencyKey becomes chat runId.
	RunCronChat func(job CronJob, sessionKey, sessionId, message, idempotencyKey string) (runID string, ok bool)
}
