package workqueue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
	openoctadb "github.com/openocta/openocta/pkg/db"
)

// Service manages work plans and background workers.
type Service struct {
	repo     *repository
	executor *executor
	cfg      RuntimeConfig
	mu       sync.Mutex
	workerID string
	done     chan struct{}
	wg       sync.WaitGroup
}

var defaultService *Service

// Init wires the default work queue service.
func Init(cfg RuntimeConfig, deps *ExecutorDeps) error {
	db := openoctadb.GetDB()
	if db == nil {
		defaultService = nil
		return nil
	}
	defaultService = NewService(db, cfg, deps)
	defaultService.Start()
	return nil
}

// Default returns the process-wide work queue service.
func Default() *Service {
	return defaultService
}

// NewService creates a work queue service.
func NewService(db *sql.DB, cfg RuntimeConfig, deps *ExecutorDeps) *Service {
	if cfg.MaxConcurrentL2Runs <= 0 {
		cfg.MaxConcurrentL2Runs = defaultMaxConcurrentL2Runs
	}
	return &Service{
		repo:     newRepository(db),
		executor: &executor{deps: deps, cfg: cfg},
		cfg:      cfg,
		workerID: fmt.Sprintf("worker-%d", time.Now().UnixNano()),
	}
}

// Enabled reports whether the work queue is available.
func Enabled() bool {
	return defaultService != nil && defaultService.repo != nil
}

// Submit enqueues a plan without waiting for completion.
func Submit(env TriggerEnvelope) (SubmitResult, error) {
	if defaultService == nil {
		return SubmitResult{}, fmt.Errorf("work queue service 未初始化")
	}
	return defaultService.Submit(env)
}

// Submit enqueues a plan and returns immediately.
func (s *Service) Submit(env TriggerEnvelope) (SubmitResult, error) {
	if s == nil || s.repo == nil {
		return SubmitResult{}, fmt.Errorf("work queue service 未初始化")
	}
	plan, err := BuildPlan(env)
	if err != nil {
		return SubmitResult{}, err
	}

	if existing, err := s.repo.getPlanByIdempotency(plan.IdempotencyKey); err == nil {
		return SubmitResult{
			PlanID:      existing.ID,
			ParentRunID: existing.ParentRunID,
			Status:      existing.Status,
		}, nil
	} else if err != sql.ErrNoRows {
		return SubmitResult{}, err
	}

	if err := s.persistPlan(plan); err != nil {
		if existing, gerr := s.repo.getPlanByIdempotency(plan.IdempotencyKey); gerr == nil {
			return SubmitResult{
				PlanID:      existing.ID,
				ParentRunID: existing.ParentRunID,
				Status:      existing.Status,
			}, nil
		}
		return SubmitResult{}, err
	}

	if err := s.startParentRun(plan); err != nil {
		_ = s.repo.updatePlanStatus(plan.ID, PlanStatusFailed, err.Error())
		return SubmitResult{}, err
	}

	return SubmitResult{
		PlanID:      plan.ID,
		ParentRunID: plan.ParentRunID,
		Status:      PlanStatusQueued,
	}, nil
}

// SubmitAndWait enqueues a plan and blocks until it reaches a terminal state.
func (s *Service) SubmitAndWait(ctx context.Context, env TriggerEnvelope) (SubmitResult, error) {
	if s == nil || s.repo == nil {
		return SubmitResult{}, fmt.Errorf("work queue service 未初始化")
	}
	plan, err := BuildPlan(env)
	if err != nil {
		return SubmitResult{}, err
	}

	if existing, err := s.repo.getPlanByIdempotency(plan.IdempotencyKey); err == nil {
		return s.waitForPlan(ctx, existing.ID, existing.ParentRunID)
	} else if err != sql.ErrNoRows {
		return SubmitResult{}, err
	}

	if err := s.persistPlan(plan); err != nil {
		if existing, gerr := s.repo.getPlanByIdempotency(plan.IdempotencyKey); gerr == nil {
			return s.waitForPlan(ctx, existing.ID, existing.ParentRunID)
		}
		return SubmitResult{}, err
	}

	if err := s.startParentRun(plan); err != nil {
		_ = s.repo.updatePlanStatus(plan.ID, PlanStatusFailed, err.Error())
		return SubmitResult{}, err
	}

	_ = s.repo.updatePlanStatus(plan.ID, PlanStatusRunning, "")
	return s.waitForPlan(ctx, plan.ID, plan.ParentRunID)
}

func (s *Service) persistPlan(plan WorkPlan) error {
	now := nowMs()
	envelopeJSON, _ := json.Marshal(plan.Envelope)
	planJSON, _ := json.Marshal(plan)
	stored := storedPlan{
		ID:             plan.ID,
		TenantID:       "default",
		TriggerType:    plan.TriggerType,
		TriggerRef:     plan.TriggerRef,
		ScenarioKey:    plan.ScenarioKey,
		ParentRunID:    plan.ParentRunID,
		Status:         PlanStatusQueued,
		Priority:       plan.Priority,
		IdempotencyKey: plan.IdempotencyKey,
		ScheduledAtMs:  plan.ScheduledAtMs,
		EnvelopeJSON:   string(envelopeJSON),
		PlanJSON:       string(planJSON),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.insertPlan(stored); err != nil {
		return err
	}

	for _, step := range plan.Steps {
		task := WorkTask{
			ID:             newTaskID(),
			PlanID:         plan.ID,
			TenantID:       "default",
			Tier:           step.Tier,
			Action:         step.Action,
			ObjectType:     plan.Envelope.Scope.ObjectType,
			ObjectID:       firstNonEmpty(plan.Envelope.ClusterID, plan.Envelope.Scope.ClusterID),
			ParentRunID:    plan.ParentRunID,
			Status:         TaskStatusQueued,
			Priority:       plan.Priority,
			IdempotencyKey: fmt.Sprintf("%s:%s:%s", plan.IdempotencyKey, step.Tier, step.Action),
			CreatedAt:      now,
			UpdatedAt:      now,
			Input: map[string]interface{}{
				"scenarioKey": plan.ScenarioKey,
				"action":      step.Action,
			},
		}
		if step.Tier == TierL2 {
			task.ChildRunID = task.ID
		}
		if err := s.repo.insertTask(task); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) startParentRun(plan WorkPlan) error {
	jr := jobrun.Default()
	if jr == nil {
		return nil
	}
	if _, err := jr.Get(plan.ParentRunID); err == nil {
		return nil
	}
	_, err := jr.Start(jobrun.StartInput{
		RunID:       plan.ParentRunID,
		JobID:       plan.TriggerRef,
		TriggerType: plan.TriggerType,
		TriggerRef:  plan.ScenarioKey,
		Input: map[string]interface{}{
			"planId":      plan.ID,
			"scenarioKey": plan.ScenarioKey,
			"scheduledAt": plan.ScheduledAtMs,
		},
	})
	return err
}

func (s *Service) waitForPlan(ctx context.Context, planID, parentRunID string) (SubmitResult, error) {
	deadline := time.Now().Add(time.Duration(s.cfg.ParentRunTimeoutMs) * time.Millisecond)
	ticker := time.NewTicker(time.Duration(s.cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return SubmitResult{}, ctx.Err()
		default:
		}
		status, errMsg, err := s.aggregatePlanStatus(planID)
		if err != nil {
			return SubmitResult{}, err
		}
		if isTerminalPlanStatus(status) {
			out := SubmitResult{
				PlanID:      planID,
				ParentRunID: parentRunID,
				Status:      status,
				Error:       errMsg,
			}
			s.finishParentRun(parentRunID, status, errMsg)
			return out, nil
		}
		if time.Now().After(deadline) {
			_ = s.repo.updatePlanStatus(planID, PlanStatusPartial, "parent run timeout")
			s.finishParentRun(parentRunID, jobrun.StatusPartial, "parent run timeout")
			return SubmitResult{
				PlanID:      planID,
				ParentRunID: parentRunID,
				Status:      PlanStatusPartial,
				Error:       "parent run timeout",
			}, nil
		}
		select {
		case <-ctx.Done():
			return SubmitResult{}, ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) aggregatePlanStatus(planID string) (string, string, error) {
	tasks, err := s.repo.listTasksByPlan(planID)
	if err != nil {
		return "", "", err
	}
	if len(tasks) == 0 {
		return PlanStatusFailed, "no tasks", nil
	}

	queuedOrRunning := 0
	failed := 0
	timeout := 0
	for _, t := range tasks {
		switch t.Status {
		case TaskStatusQueued, TaskStatusRunning:
			queuedOrRunning++
		case TaskStatusFailed, TaskStatusCancelled:
			failed++
		case TaskStatusTimeout:
			timeout++
		}
	}
	if queuedOrRunning > 0 {
		return PlanStatusRunning, "", nil
	}
	if failed == 0 && timeout == 0 {
		_ = s.repo.updatePlanStatus(planID, PlanStatusSucceeded, "")
		return PlanStatusSucceeded, "", nil
	}
	if failed+timeout < len(tasks) {
		_ = s.repo.updatePlanStatus(planID, PlanStatusPartial, "some tasks failed")
		return PlanStatusPartial, "some tasks failed", nil
	}
	_ = s.repo.updatePlanStatus(planID, PlanStatusFailed, "all tasks failed")
	return PlanStatusFailed, "all tasks failed", nil
}

func (s *Service) finishParentRun(parentRunID, status, errMsg string) {
	jr := jobrun.Default()
	if jr == nil || strings.TrimSpace(parentRunID) == "" {
		return
	}
	output := map[string]interface{}{"status": status}
	switch status {
	case PlanStatusSucceeded, PlanStatusPartial:
		_ = jr.Succeed(parentRunID, jobrun.FinishInput{Output: output})
	default:
		_ = jr.Fail(parentRunID, errMsg, output)
	}
}

// Start launches background workers.
func (s *Service) Start() {
	if s == nil {
		return
	}
	s.mu.Lock()
	if s.done != nil {
		s.mu.Unlock()
		return
	}
	s.done = make(chan struct{})
	s.mu.Unlock()

	_, _ = s.repo.reclaimStaleTasks(nowMs())
	workers := s.cfg.MaxConcurrentL2Runs
	if workers < 2 {
		workers = 2
	}
	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.workerLoop(fmt.Sprintf("%s-%d", s.workerID, i))
	}
}

// Stop stops background workers.
func (s *Service) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	done := s.done
	s.done = nil
	s.mu.Unlock()
	if done != nil {
		close(done)
	}
	s.wg.Wait()
}

func (s *Service) workerLoop(workerID string) {
	defer s.wg.Done()
	ticker := time.NewTicker(time.Duration(s.cfg.PollIntervalMs) * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		default:
		}
		_, _ = s.repo.reclaimStaleTasks(nowMs())
		l2Running, _ := s.repo.countRunningL2()
		task, err := s.repo.claimNextTask(workerID, s.cfg.TaskLeaseMs, l2Running, s.cfg.MaxConcurrentL2Runs)
		if err != nil || task == nil {
			select {
			case <-s.done:
				return
			case <-ticker.C:
			}
			continue
		}
		s.runTask(*task)
	}
}

func (s *Service) runTask(task WorkTask) {
	plan, err := s.repo.getPlanByID(task.PlanID)
	if err != nil {
		_ = s.repo.finishTask(task.ID, TaskStatusFailed, err.Error(), nil)
		return
	}
	var env TriggerEnvelope
	_ = json.Unmarshal([]byte(plan.EnvelopeJSON), &env)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(s.cfg.ParentRunTimeoutMs)*time.Millisecond)
	defer cancel()

	err = s.executor.executeTask(ctx, task, env)
	if err != nil {
		status := TaskStatusFailed
		if strings.Contains(err.Error(), "timeout") {
			status = TaskStatusTimeout
		}
		_ = s.repo.finishTask(task.ID, status, err.Error(), nil)
		return
	}
	_ = s.repo.finishTask(task.ID, TaskStatusSucceeded, "", map[string]interface{}{"status": "ok"})

	if task.Tier == TierL0 && ops.IsBatchL0Scenario(env.ScenarioKey) {
		_ = s.enqueueBatchEscalations(plan, env)
	}
	_ = s.maybeEnqueueDomainReduce(plan, env)
}

func isTerminalPlanStatus(status string) bool {
	switch status {
	case PlanStatusSucceeded, PlanStatusPartial, PlanStatusFailed:
		return true
	default:
		return false
	}
}

func nowMs() int64 {
	return time.Now().UnixMilli()
}
