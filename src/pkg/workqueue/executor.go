package workqueue

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/runnotify"
)

// ExecutorDeps are runtime hooks for task execution.
type ExecutorDeps struct {
	RunScenario func(ctx context.Context, scenarioKey, objectID string, opts ops.RunOpts) (ops.InspectionResult, error)
	RunCronChat func(job CronJobSnapshot, sessionKey, sessionID, message, idempotencyKey string) (runID string, ok bool)
}

type executor struct {
	deps *ExecutorDeps
	cfg  RuntimeConfig
}

func (e *executor) executeTask(ctx context.Context, task WorkTask, env TriggerEnvelope) error {
	if e == nil || e.deps == nil {
		return fmt.Errorf("work queue executor 未配置")
	}
	switch task.Tier {
	case TierL0:
		return e.runL0(ctx, task, env)
	case TierL1:
		return e.runL1(ctx, task, env)
	case TierL2:
		if task.Action == ActionDomainReduce {
			return e.runDomainReduce(ctx, task, env)
		}
		return e.runL2(ctx, task, env)
	default:
		return fmt.Errorf("unsupported tier %s", task.Tier)
	}
}

func (e *executor) runL0(ctx context.Context, task WorkTask, env TriggerEnvelope) error {
	if !ops.IsBatchL0Scenario(env.ScenarioKey) {
		return fmt.Errorf("unsupported L0 scenario %s", env.ScenarioKey)
	}
	return ops.RunBatchL0(ctx, env.ScenarioKey, ops.BatchL0Opts{
		RunID:     task.ParentRunID,
		Domain:    firstNonEmpty(env.Domain, env.Scope.Domain),
		ClusterID: firstNonEmpty(env.ClusterID, env.Scope.ClusterID),
	})
}

func (e *executor) runL1(ctx context.Context, task WorkTask, env TriggerEnvelope) error {
	if e.deps.RunScenario == nil {
		return fmt.Errorf("RunScenario 未配置")
	}
	objectID := firstNonEmpty(env.ClusterID, env.Scope.ClusterID)
	sessionID := task.ParentRunID
	res, err := e.deps.RunScenario(ctx, env.ScenarioKey, objectID, ops.RunOpts{
		SessionID:   sessionID,
		RunID:       task.ParentRunID,
		EmployeeID:  env.CronJob.DigitalEmployeeID,
		JobID:       env.TriggerRef,
		TriggerType: env.TriggerType,
		TriggerRef:  env.ScenarioKey,
	})
	if err != nil {
		return err
	}
	_ = res
	return nil
}

func (e *executor) runL2(ctx context.Context, task WorkTask, env TriggerEnvelope) error {
	if e.deps.RunCronChat == nil {
		return fmt.Errorf("RunCronChat 未配置")
	}
	job := env.CronJob
	message := env.Message
	if task.Input != nil {
		if msg, ok := task.Input["message"].(string); ok && strings.TrimSpace(msg) != "" {
			message = msg
		}
	}
	if message == "" {
		message = job.PayloadMessage
	}
	if domain := firstNonEmpty(env.Domain, env.Scope.Domain); domain != "" {
		prefix := tools.BuildOpsContextLine(domain, env.ClusterID, env.Component)
		if prefix != "" && !strings.Contains(message, "[运维上下文]") {
			message = prefix + "\n\n" + message
		}
	}

	childRunID := strings.TrimSpace(task.ChildRunID)
	if childRunID == "" {
		childRunID = task.ID
	}

	jr := jobrun.Default()
	if jr != nil {
		if _, err := jr.Get(childRunID); err != nil {
			triggerType := jobrun.TriggerEscalation
			if env.TriggerType == jobrun.TriggerAlert {
				triggerType = jobrun.TriggerAlert
			} else if env.TriggerType == jobrun.TriggerManual {
				triggerType = jobrun.TriggerManual
			}
			_, _ = jr.Start(jobrun.StartInput{
				RunID:       childRunID,
				JobID:       env.TriggerRef,
				ParentRunID: task.ParentRunID,
				TriggerType: triggerType,
				TriggerRef:  firstNonEmpty(task.ObjectID, env.ScenarioKey),
				Input: map[string]interface{}{
					"tier":        TierL2,
					"scenarioKey": env.ScenarioKey,
					"objectId":    task.ObjectID,
				},
			})
		}
	}

	sessionKey, sessionID := resolveCronSession(job, env.CronMode)
	runnotify.Register(childRunID)
	runID, ok := e.deps.RunCronChat(job, sessionKey, sessionID, message, childRunID)
	if !ok || strings.TrimSpace(runID) == "" {
		if jr != nil {
			_ = jr.Fail(childRunID, "chat.send failed", nil)
		}
		return fmt.Errorf("chat.send failed")
	}

	timeout := time.Duration(e.cfg.L2RunTimeoutMs) * time.Millisecond
	result, completed := runnotify.Wait(childRunID, timeout)
	if jr != nil {
		if completed && result.Status == "ok" {
			_ = jr.Succeed(childRunID, jobrun.FinishInput{Output: map[string]interface{}{"status": "ok"}})
		} else if completed {
			_ = jr.Fail(childRunID, result.Error, map[string]interface{}{"status": result.Status})
		} else {
			_ = jr.Fail(childRunID, result.Error, map[string]interface{}{"status": jobrun.StatusTimeout})
		}
	}
	if !completed || result.Status != "ok" {
		if result.Error != "" {
			return fmt.Errorf("%s", result.Error)
		}
		return fmt.Errorf("L2 run %s", result.Status)
	}
	return nil
}

func (e *executor) runDomainReduce(ctx context.Context, task WorkTask, env TriggerEnvelope) error {
	var taskInputs []ops.DomainReduceTaskInput
	if task.Input != nil {
		if raw, ok := task.Input["escalationTasksJson"].(string); ok && strings.TrimSpace(raw) != "" {
			_ = json.Unmarshal([]byte(raw), &taskInputs)
		}
	}
	inputs, err := ops.CollectDomainReduceInputs(task.ParentRunID, env.ScenarioKey, taskInputs)
	if err != nil {
		return err
	}
	result := ops.BuildRuleDomainReduce(task.ParentRunID, task.PlanID, env.ScenarioKey, env.TriggerRef, inputs)

	if e.cfg.DomainReduceUseLLM && e.deps.RunCronChat != nil {
		childRunID := strings.TrimSpace(task.ChildRunID)
		if childRunID == "" {
			childRunID = task.ID
		}
		msg := ops.BuildDomainReduceLLMMessage(result, inputs)
		job := env.CronJob
		sessionKey := strings.TrimSpace(job.SessionKey)
		if sessionKey == "" {
			sessionKey = "agent:main:cron:" + strings.TrimSpace(env.TriggerRef) + ":reduce"
		}
		jr := jobrun.Default()
		if jr != nil {
			if _, err := jr.Get(childRunID); err != nil {
				_, _ = jr.Start(jobrun.StartInput{
					RunID:       childRunID,
					JobID:       env.TriggerRef,
					ParentRunID: task.ParentRunID,
					TriggerType: jobrun.TriggerEscalation,
					TriggerRef:  env.ScenarioKey + ops.ScenarioReduceSuffix,
					Input: map[string]interface{}{
						"tier":        TierL2,
						"action":      ActionDomainReduce,
						"scenarioKey": env.ScenarioKey,
					},
				})
			}
		}
		runnotify.Register(childRunID)
		runID, ok := e.deps.RunCronChat(job, sessionKey, childRunID, msg, childRunID)
		if !ok || strings.TrimSpace(runID) == "" {
			return fmt.Errorf("domain reduce chat.send failed")
		}
		timeout := time.Duration(e.cfg.L2RunTimeoutMs) * time.Millisecond
		waitResult, completed := runnotify.Wait(childRunID, timeout)
		if jr != nil {
			if completed && waitResult.Status == "ok" {
				_ = jr.Succeed(childRunID, jobrun.FinishInput{Output: map[string]interface{}{"status": "ok"}})
			} else {
				_ = jr.Fail(childRunID, waitResult.Error, map[string]interface{}{"status": waitResult.Status})
			}
		}
		if completed && waitResult.Status == "ok" {
			if reports, _ := ops.ListInspectionReportsByRunIDs([]string{childRunID}); len(reports) > 0 {
				if rep, ok := reports[childRunID]; ok && strings.TrimSpace(rep.ReportMarkdown) != "" {
					result = ops.MergeLLMIntoDomainReduce(result, rep.ReportMarkdown)
				} else if rep, ok := reports[childRunID]; ok && strings.TrimSpace(rep.Summary) != "" {
					result = ops.MergeLLMIntoDomainReduce(result, rep.Summary)
				}
			}
		}
	}

	return ops.PersistDomainReduceSummary(result)
}

func resolveCronSession(job CronJobSnapshot, mode string) (sessionKey, sessionID string) {
	if emp := strings.TrimSpace(strings.ToLower(job.DigitalEmployeeID)); emp != "" {
		return "agent:main:employee:" + emp, ""
	}
	if mode == "force" {
		sessionID = job.ID
		sessionKey = "agent:main:cron:" + job.ID + ":run:" + sessionID
		return sessionKey, sessionID
	}
	sessionKey = strings.TrimSpace(job.SessionKey)
	if sessionKey == "" {
		sessionKey = "agent:main:cron:" + job.ID
	}
	return sessionKey, job.ID
}
