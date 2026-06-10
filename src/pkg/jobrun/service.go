package jobrun

import (
	"fmt"
	"strings"
	"sync"

	openoctadb "github.com/openocta/openocta/pkg/db"
)

// Service manages job run lifecycle and steps.
type Service struct {
	repo *jobRunRepository
	mu   sync.Mutex
}

var defaultService *Service

// Init wires the default job run service to openocta.db.
func Init() error {
	db := openoctadb.GetDB()
	if db == nil {
		defaultService = nil
		return nil
	}
	defaultService = NewService(newJobRunRepository(db))
	return nil
}

// Default returns the process-wide job run service, or nil when DB is unavailable.
func Default() *Service {
	return defaultService
}

// NewService creates a job run service backed by repository.
func NewService(repo *jobRunRepository) *Service {
	if repo == nil {
		return nil
	}
	return &Service{repo: repo}
}

// Start creates a running job run record.
func (s *Service) Start(input StartInput) (JobRun, error) {
	if s == nil || s.repo == nil {
		return JobRun{}, fmt.Errorf("job run service 未初始化")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := nowMs()
	run := JobRun{
		ID:          newRunID(input.RunID),
		JobID:       strings.TrimSpace(input.JobID),
		TaskID:      strings.TrimSpace(input.TaskID),
		ParentRunID: strings.TrimSpace(input.ParentRunID),
		TriggerType: strings.TrimSpace(input.TriggerType),
		TriggerRef:  strings.TrimSpace(input.TriggerRef),
		Status:      StatusRunning,
		StartedAt:   now,
		Input:       cloneMap(input.Input),
		Output:      map[string]interface{}{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if run.TriggerType == "" {
		run.TriggerType = TriggerInspection
	}
	if err := s.repo.Insert(run); err != nil {
		return JobRun{}, err
	}
	return run, nil
}

// Succeed marks a run as succeeded.
func (s *Service) Succeed(runID string, input FinishInput) error {
	return s.finish(runID, StatusSucceeded, "", input.Output)
}

// Fail marks a run as failed.
func (s *Service) Fail(runID, errMsg string, output map[string]interface{}) error {
	return s.finish(runID, StatusFailed, errMsg, output)
}

// Cancel marks a run as cancelled.
func (s *Service) Cancel(runID, reason string) error {
	return s.finish(runID, StatusCancelled, reason, nil)
}

// WaitApproval moves a running run into waiting_approval.
func (s *Service) WaitApproval(runID string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("job run service 未初始化")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	run, err := s.repo.Get(runID)
	if err != nil {
		return err
	}
	if run.Status != StatusRunning {
		return fmt.Errorf("job run %s cannot enter waiting_approval from status %s", runID, run.Status)
	}
	run.Status = StatusWaitingApproval
	run.UpdatedAt = nowMs()
	return s.repo.Update(run)
}

// Get returns one run by id.
func (s *Service) Get(runID string) (JobRun, error) {
	if s == nil || s.repo == nil {
		return JobRun{}, fmt.Errorf("job run service 未初始化")
	}
	return s.repo.Get(runID)
}

// ListByJobID returns recent runs for a job.
func (s *Service) ListByJobID(jobID string, limit int) ([]JobRun, error) {
	return s.List(ListFilter{JobID: jobID, Limit: limit})
}

// List returns job runs matching filter criteria.
func (s *Service) List(filter ListFilter) ([]JobRun, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("job run service 未初始化")
	}
	return s.repo.List(filter)
}

// GetDetail returns one run with ordered steps.
func (s *Service) GetDetail(runID string) (RunDetail, error) {
	if s == nil || s.repo == nil {
		return RunDetail{}, fmt.Errorf("job run service 未初始化")
	}
	run, err := s.repo.Get(runID)
	if err != nil {
		return RunDetail{}, err
	}
	steps, err := s.repo.ListSteps(runID)
	if err != nil {
		return RunDetail{}, err
	}
	if steps == nil {
		steps = []RunStep{}
	}
	invocations, err := s.repo.ListToolInvocations(runID)
	if err != nil {
		return RunDetail{}, err
	}
	if invocations == nil {
		invocations = []ToolInvocation{}
	}
	return RunDetail{Run: run, Steps: steps, ToolInvocations: invocations}, nil
}

// AddStep appends a step to a run.
func (s *Service) AddStep(runID string, input StepInput) (RunStep, error) {
	if s == nil || s.repo == nil {
		return RunStep{}, fmt.Errorf("job run service 未初始化")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := s.repo.Get(runID); err != nil {
		return RunStep{}, err
	}
	order, err := s.repo.nextStepOrder(runID)
	if err != nil {
		return RunStep{}, err
	}
	now := nowMs()
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = StatusSucceeded
	}
	step := RunStep{
		ID:            newRunID(""),
		RunID:         runID,
		StepOrder:     order,
		Kind:          strings.TrimSpace(input.Kind),
		Name:          strings.TrimSpace(input.Name),
		Status:        status,
		StartedAt:     now,
		FinishedAt:    now,
		Error:         strings.TrimSpace(input.Error),
		InputSummary:  strings.TrimSpace(input.InputSummary),
		OutputSummary: strings.TrimSpace(input.OutputSummary),
	}
	if err := s.repo.InsertStep(step); err != nil {
		return RunStep{}, err
	}
	return step, nil
}

// ListSteps returns ordered steps for a run.
func (s *Service) ListSteps(runID string) ([]RunStep, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("job run service 未初始化")
	}
	return s.repo.ListSteps(runID)
}

func (s *Service) finish(runID, status, errMsg string, output map[string]interface{}) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("job run service 未初始化")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	run, err := s.repo.Get(runID)
	if err != nil {
		return err
	}
	if run.Status != StatusRunning && run.Status != StatusWaitingApproval {
		return fmt.Errorf("job run %s cannot finish from status %s", runID, run.Status)
	}
	now := nowMs()
	run.Status = status
	run.FinishedAt = now
	run.UpdatedAt = now
	run.Error = strings.TrimSpace(errMsg)
	if output != nil {
		run.Output = cloneMap(output)
	}
	return s.repo.Update(run)
}

func cloneMap(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return map[string]interface{}{}
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// TriggerTypeForCronMode maps cron execution mode to trigger type.
func TriggerTypeForCronMode(mode string) string {
	if strings.TrimSpace(mode) == "force" {
		return TriggerManual
	}
	return TriggerCron
}
