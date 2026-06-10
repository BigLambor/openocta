package cron

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/db"
	"github.com/openocta/openocta/pkg/jobrun"
	"github.com/openocta/openocta/pkg/ops"
	"github.com/openocta/openocta/pkg/workqueue"
)

// Service manages cron jobs.
type Service struct {
	storePath string
	mu        sync.RWMutex
	store     *StoreFile
	repo      *jobRepository
	deps      *Deps
	done      chan struct{}
	wg        sync.WaitGroup
}

func normalizeDigitalEmployeeID(raw string) string {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return ""
	}
	// 兼容 UI/目录层 local:<id>：cron 存储与会话 key 一律使用去前缀后的稳定 id。
	s = strings.TrimPrefix(s, "local:")
	// employee 会话 key 以 ":" 分隔，id 内不能包含冒号
	s = strings.ReplaceAll(s, ":", "-")
	return strings.TrimSpace(s)
}

// cronRunLogEntry mirrors the TS CronRunLogEntry used by the control UI run history.
type cronRunLogEntry struct {
	Ts          int64                 `json:"ts"`
	JobID       string                `json:"jobId"`
	RunID       string                `json:"runId,omitempty"`
	Action      string                `json:"action"`
	Status      string                `json:"status,omitempty"`
	Error       string                `json:"error,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	SessionID   string                `json:"sessionId,omitempty"`
	SessionKey  string                `json:"sessionKey,omitempty"`
	RunAtMs     *int64                `json:"runAtMs,omitempty"`
	DurationMs  *int64                `json:"durationMs,omitempty"`
	NextRunAtMs *int64                `json:"nextRunAtMs,omitempty"`
	Domain      string                `json:"domain,omitempty"`
	ClusterID   string                `json:"clusterId,omitempty"`
	Component   string                `json:"component,omitempty"`
	ScenarioKey string                `json:"scenarioKey,omitempty"`
	Result      *ops.InspectionResult `json:"result,omitempty"`
}

// resolveRunLogPath returns the JSONL run log path for a job ID.
func (s *Service) resolveRunLogPath(jobID string) string {
	dir := filepath.Dir(s.storePath)
	return filepath.Join(dir, "runs", jobID+".jsonl")
}

// appendRunLogEntry appends one finished-action entry to the job's run log.
func (s *Service) appendRunLogEntry(job CronJob, runID, status, errMsg, summary, sessionKey, sessionID string, runAtMs, durationMs, nextRunAtMs *int64, domain, clusterID, component, scenarioKey string, result *ops.InspectionResult) {
	entry := cronRunLogEntry{
		Ts:          time.Now().UnixMilli(),
		JobID:       job.ID,
		RunID:       strings.TrimSpace(runID),
		Action:      "finished",
		Status:      status,
		Error:       errMsg,
		Summary:     summary,
		SessionID:   sessionID,
		SessionKey:  sessionKey,
		RunAtMs:     runAtMs,
		DurationMs:  durationMs,
		NextRunAtMs: nextRunAtMs,
		Domain:      domain,
		ClusterID:   clusterID,
		Component:   component,
		ScenarioKey: scenarioKey,
		Result:      result,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	path := s.resolveRunLogPath(job.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

// SetDeps sets execution dependencies (call after creation from gateway).
func (s *Service) SetDeps(deps *Deps) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deps = deps
}

// NewService creates a new cron service.
func NewService(storePath string) (*Service, error) {
	if storePath == "" {
		storePath = filepath.Join(".openocta", "cron", "jobs.json")
	}

	sqliteDB := db.GetDB()
	if sqliteDB != nil {
		repo := newJobRepository(sqliteDB)
		if repo == nil {
			return nil, fmt.Errorf("cron job repository 未初始化")
		}
		if _, err := repo.ImportJSON(storePath); err != nil {
			fmt.Printf("warning: cron JSON import failed: %v\n", err)
		}
		if _, err := repo.ImportLegacyCronJobs(); err != nil {
			fmt.Printf("warning: cron legacy blob import failed: %v\n", err)
		}
		svc := &Service{
			storePath: storePath,
			repo:      repo,
		}
		_ = svc.ensureDefaultJobs()
		return svc, nil
	}

	// 确保存储路径所在目录存在，不存在则创建
	dir := filepath.Dir(storePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, err
		}
	}
	store, err := LoadStore(storePath)
	if err != nil {
		return nil, err
	}
	svc := &Service{
		storePath: storePath,
		store:     store,
	}
	_ = svc.ensureDefaultJobs()
	return svc, nil
}

// List returns all jobs, optionally including disabled.
func (s *Service) List(includeDisabled bool) ([]CronJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.repo != nil {
		return s.repo.List(includeDisabled)
	}

	var out []CronJob
	for _, j := range s.store.Jobs {
		if j.Enabled || includeDisabled {
			out = append(out, j)
		}
	}
	return out, nil
}

// JobCreate is the input for adding a job.
type JobCreate struct {
	Name          string
	Description   string
	AgentID       string
	Schedule      CronSchedule
	Payload       CronPayload
	SessionTarget string
	SessionKey    string // 定时调度时使用的 sessionKey，格式 agent:main:cron:<jobId>
	// DigitalEmployeeID 选择的数字员工 id（可选）
	DigitalEmployeeID string
	WakeMode          string
	Enabled           bool
	Delivery          *CronDelivery
}

// Add adds a new job.
func (s *Service) Add(input JobCreate) (CronJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	next := ComputeNextRunAtMs(input.Schedule, now)
	j := CronJob{
		ID:                uuid.New().String(),
		AgentID:           strings.TrimSpace(strings.ToLower(input.AgentID)),
		Name:              input.Name,
		Description:       strings.TrimSpace(input.Description),
		DigitalEmployeeID: normalizeDigitalEmployeeID(input.DigitalEmployeeID),
		Enabled:           input.Enabled,
		CreatedAtMs:       now,
		UpdatedAtMs:       now,
		Schedule:          input.Schedule,
		SessionTarget:     input.SessionTarget,
		SessionKey:        strings.TrimSpace(input.SessionKey),
		WakeMode:          input.WakeMode,
		Payload:           input.Payload,
		Delivery:          input.Delivery,
		State: CronJobState{
			NextRunAtMs: &next,
		},
	}
	if j.SessionTarget == "" {
		j.SessionTarget = "main"
	}
	if j.WakeMode == "" {
		j.WakeMode = "next-heartbeat"
	}

	if s.repo != nil {
		return j, s.repo.Upsert(j)
	}

	s.store.Jobs = append(s.store.Jobs, j)
	return j, SaveStore(s.storePath, s.store)
}

// JobPatch is a partial update for a job.
type JobPatch struct {
	Enabled           *bool
	Name              string
	Description       *string
	AgentID           *string
	Schedule          *CronSchedule
	SessionKey        *string // 定时调度时使用的 sessionKey，nil 表示不修改
	DigitalEmployeeID *string // 数字员工 id，nil 表示不修改；空字符串表示清空
	SessionTarget     *string
	WakeMode          *string
	Payload           *CronPayload
	Delivery          *CronDelivery
}

// GetJob returns a copy of the job by ID, or false if not found.
func (s *Service) GetJob(id string) (CronJob, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.repo != nil {
		job, ok, err := s.repo.Get(id)
		if err != nil || !ok {
			return CronJob{}, false
		}
		return job, true
	}

	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == id {
			return s.store.Jobs[i], true
		}
	}
	return CronJob{}, false
}

// Update updates a job by ID.
func (s *Service) Update(id string, patch JobPatch) (CronJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repo != nil {
		j, ok, err := s.repo.Get(id)
		if err != nil {
			return CronJob{}, err
		}
		if !ok {
			return CronJob{}, nil
		}

		if patch.Enabled != nil {
			j.Enabled = *patch.Enabled
		}
		if patch.Name != "" {
			j.Name = patch.Name
		}
		if patch.Description != nil {
			j.Description = strings.TrimSpace(*patch.Description)
		}
		if patch.AgentID != nil {
			j.AgentID = strings.TrimSpace(strings.ToLower(*patch.AgentID))
		}
		if patch.Schedule != nil {
			j.Schedule = *patch.Schedule
		}
		if patch.Payload != nil {
			j.Payload = *patch.Payload
		}
		if patch.SessionTarget != nil {
			j.SessionTarget = strings.TrimSpace(*patch.SessionTarget)
		}
		if patch.WakeMode != nil {
			j.WakeMode = strings.TrimSpace(*patch.WakeMode)
		}
		if patch.Delivery != nil {
			j.Delivery = patch.Delivery
		}
		if patch.DigitalEmployeeID != nil {
			j.DigitalEmployeeID = normalizeDigitalEmployeeID(*patch.DigitalEmployeeID)
		}
		if patch.SessionKey != nil {
			j.SessionKey = strings.TrimSpace(*patch.SessionKey)
		}
		j.UpdatedAtMs = time.Now().UnixMilli()
		if err := s.repo.Upsert(j); err != nil {
			return CronJob{}, err
		}
		return j, nil
	}

	for i := range s.store.Jobs {
		if s.store.Jobs[i].ID == id {
			if patch.Enabled != nil {
				s.store.Jobs[i].Enabled = *patch.Enabled
			}
			if patch.Name != "" {
				s.store.Jobs[i].Name = patch.Name
			}
			if patch.Description != nil {
				s.store.Jobs[i].Description = strings.TrimSpace(*patch.Description)
			}
			if patch.AgentID != nil {
				s.store.Jobs[i].AgentID = strings.TrimSpace(strings.ToLower(*patch.AgentID))
			}
			if patch.Schedule != nil {
				s.store.Jobs[i].Schedule = *patch.Schedule
			}
			if patch.Payload != nil {
				s.store.Jobs[i].Payload = *patch.Payload
			}
			if patch.SessionTarget != nil {
				s.store.Jobs[i].SessionTarget = strings.TrimSpace(*patch.SessionTarget)
			}
			if patch.WakeMode != nil {
				s.store.Jobs[i].WakeMode = strings.TrimSpace(*patch.WakeMode)
			}
			if patch.Delivery != nil {
				s.store.Jobs[i].Delivery = patch.Delivery
			}
			if patch.DigitalEmployeeID != nil {
				s.store.Jobs[i].DigitalEmployeeID = normalizeDigitalEmployeeID(*patch.DigitalEmployeeID)
			}
			if patch.SessionKey != nil {
				s.store.Jobs[i].SessionKey = strings.TrimSpace(*patch.SessionKey)
			}
			s.store.Jobs[i].UpdatedAtMs = time.Now().UnixMilli()
			j := s.store.Jobs[i]
			return j, SaveStore(s.storePath, s.store)
		}
	}
	return CronJob{}, nil // not found
}

// Remove removes a job by ID.
func (s *Service) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.repo != nil {
		err := s.repo.Delete(id)
		if err == sql.ErrNoRows {
			return nil
		}
		return err
	}

	for i, j := range s.store.Jobs {
		if j.ID == id {
			s.store.Jobs = append(s.store.Jobs[:i], s.store.Jobs[i+1:]...)
			return SaveStore(s.storePath, s.store)
		}
	}
	return nil
}

// Run runs a job by ID. mode is "due" or "force".
func (s *Service) Run(id string, mode string, domain, clusterId, component, scenarioKey string) error {
	startMs := time.Now().UnixMilli()

	// Snapshot job and deps under lock so we can safely execute without holding the mutex.
	s.mu.Lock()
	var jobCopy CronJob
	var found bool
	if s.repo != nil {
		var err error
		jobCopy, found, err = s.repo.Get(id)
		if err != nil {
			s.mu.Unlock()
			return err
		}
	} else {
		for i := range s.store.Jobs {
			if s.store.Jobs[i].ID == id {
				jobCopy = s.store.Jobs[i]
				found = true
				break
			}
		}
	}
	if !found {
		s.mu.Unlock()
		return nil
	}
	deps := s.deps
	s.mu.Unlock()

	status := "ok"
	errMsg := ""
	var sessionKey, cronSessionID string
	var trackedRunID string
	scenarioKey = strings.TrimSpace(scenarioKey)
	if scenarioKey == "" {
		scenarioKey = ops.ScenarioKeyForInspection(ops.InspectionReport{
			JobID:  id,
			Domain: strings.TrimSpace(domain),
		})
	}

	// Validate payload/sessionTarget combinations (mirrors TS semantics).
	if jobCopy.SessionTarget == "main" {
		if jobCopy.Payload.Kind != "systemEvent" {
			status = "skipped"
			errMsg = `main job requires payload.kind="systemEvent"`
		}
	} else if jobCopy.SessionTarget == "isolated" {
		if jobCopy.Payload.Kind != "agentTurn" {
			status = "skipped"
			errMsg = `isolated job requires payload.kind="agentTurn"`
		} else {
			// 若选择了数字员工，则优先使用数字员工稳定会话 key：
			// agent:main:employee:<employeeId>
			// 并让网关通过 sessions.ensure / sessions store 解析 sessionId（首次会话也能自动构建）。
			if emp := strings.TrimSpace(strings.ToLower(jobCopy.DigitalEmployeeID)); emp != "" {
				sessionKey = "agent:main:employee:" + emp
				cronSessionID = ""
			} else {
				// 手动触发（mode=force）：生成新 sessionKey agent:main:cron:<jobId>:run:<sessionId>
				// 定时调度（mode=due）：使用 jobs.json 中的 sessionKey，缺省为 agent:main:cron:<jobId>
				if mode == "force" {
					cronSessionID = uuid.New().String()
					sessionKey = "agent:main:cron:" + jobCopy.ID + ":run:" + cronSessionID
				} else {
					sessionKey = strings.TrimSpace(jobCopy.SessionKey)
					if sessionKey == "" {
						sessionKey = "agent:main:cron:" + jobCopy.ID
					}
					cronSessionID = jobCopy.ID
				}
			}
		}
	}

	var inspectionResult *ops.InspectionResult
	runSummary := ""

	// Execute side effects when not skipped and deps are available.
	if status != "skipped" {
		useWorkQueue := workqueue.Enabled() && shouldUseWorkQueue(jobCopy)
		legacyRunID := ""
		if !useWorkQueue {
			legacyRunID = uuid.New().String()
			trackedRunID = startCronJobRun(id, mode, domain, clusterId, component, scenarioKey, legacyRunID)
		}
		if deps == nil {
			status = "error"
			errMsg = "cron deps not configured"
		} else if useWorkQueue {
			scheduledAt := scheduledAtForJob(jobCopy, startMs)
			trackedRunID, status, errMsg, runSummary, inspectionResult = s.executeViaWorkQueue(
				jobCopy, mode, domain, clusterId, component, scenarioKey, scheduledAt,
			)
			runSummary = cronFinishSummary(status, runSummary)
		} else {
			_, hasNativeScenario := ops.GetOpsScenario(scenarioKey)
			if hasNativeScenario {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				if cronSessionID == "" {
					cronSessionID = legacyRunID
				}
				res, err := ops.RunScenario(ctx, scenarioKey, clusterId, ops.RunOpts{
					SessionID:   cronSessionID,
					RunID:       legacyRunID,
					EmployeeID:  jobCopy.DigitalEmployeeID,
					JobID:       id,
					TriggerType: jobrun.TriggerTypeForCronMode(mode),
					TriggerRef:  scenarioKey,
				})
				if err != nil {
					status = "error"
					errMsg = err.Error()
				} else {
					inspectionResult = &res
					runSummary = strings.TrimSpace(res.ReportMarkdown)
					if runSummary == "" && res.Score != nil {
						runSummary = fmt.Sprintf("健康得分：%d", *res.Score)
					}
				}
			} else if jobCopy.SessionTarget == "main" && jobCopy.Payload.Kind == "systemEvent" {
				if deps.EnqueueSystemEvent != nil {
					deps.EnqueueSystemEvent(jobCopy.Payload.Text)
				}
				if jobCopy.WakeMode == "now" && deps.RequestHeartbeatNow != nil {
					deps.RequestHeartbeatNow("agent:main:cron:" + id)
				}
			} else if jobCopy.SessionTarget == "isolated" && jobCopy.Payload.Kind == "agentTurn" {
				message := jobCopy.Payload.Message
				if domain != "" {
					prefix := tools.BuildOpsContextLine(domain, clusterId, component)
					if prefix != "" && !strings.Contains(message, "[运维上下文]") {
						message = prefix + "\n\n" + message
					}
				}
				idempotencyKey := fmt.Sprintf("cron:%s:%d", jobCopy.ID, scheduledAtForJob(jobCopy, startMs))
				if deps.RunCronChat != nil {
					_, _ = deps.RunCronChat(jobCopy, sessionKey, cronSessionID, message, idempotencyKey)
				} else if deps.RunIsolatedAgentJob != nil {
					deps.RunIsolatedAgentJob(jobCopy, message)
				}
			}
		}
	}

	endMs := time.Now().UnixMilli()
	durationMs := endMs - startMs

	// Update job state and persist.
	s.mu.Lock()
	defer s.mu.Unlock()
	var nextRunAtMs *int64
	if s.repo != nil {
		j, ok, err := s.repo.Get(id)
		if err == nil && ok {
			j.State.LastRunAtMs = &endMs
			j.State.LastStatus = status
			if errMsg != "" {
				j.State.LastError = errMsg
			} else {
				j.State.LastError = ""
			}
			j.State.RunningAtMs = nil
			j.State.LastDurationMs = &durationMs
			if status == "ok" || status == "partial" {
				j.State.ConsecutiveErrors = 0
			} else if status == "error" {
				j.State.ConsecutiveErrors++
			}
			next := ComputeNextRunAtMs(j.Schedule, endMs)
			j.State.NextRunAtMs = &next
			nextRunAtMs = &next
			_ = s.repo.Upsert(j)
			jobCopy = j
		}
	} else {
		for i := range s.store.Jobs {
			if s.store.Jobs[i].ID == id {
				s.store.Jobs[i].State.LastRunAtMs = &endMs
				s.store.Jobs[i].State.LastStatus = status
				if errMsg != "" {
					s.store.Jobs[i].State.LastError = errMsg
				} else {
					s.store.Jobs[i].State.LastError = ""
				}
				s.store.Jobs[i].State.RunningAtMs = nil
				s.store.Jobs[i].State.LastDurationMs = &durationMs
				if status == "ok" || status == "partial" {
					s.store.Jobs[i].State.ConsecutiveErrors = 0
				} else if status == "error" {
					s.store.Jobs[i].State.ConsecutiveErrors++
				}
				next := ComputeNextRunAtMs(s.store.Jobs[i].Schedule, endMs)
				s.store.Jobs[i].State.NextRunAtMs = &next
				nextRunAtMs = &next
				_ = SaveStore(s.storePath, s.store)
				// Use the full job value for logging.
				jobCopy = s.store.Jobs[i]
				break
			}
		}
	}

	// Append run log entry without holding the lock.
	runAt := startMs
	finishCronJobRun(trackedRunID, status, errMsg, runSummary, scenarioKey, domain, clusterId, component, mode)
	s.appendRunLogEntry(jobCopy, trackedRunID, status, errMsg, runSummary, sessionKey, cronSessionID, &runAt, &durationMs, nextRunAtMs, domain, clusterId, component, scenarioKey, inspectionResult)

	return nil
}

func startCronJobRun(jobID, mode, domain, clusterID, component, scenarioKey, runID string) string {
	jr := jobrun.Default()
	if jr == nil {
		return ""
	}
	run, err := jr.Start(jobrun.StartInput{
		RunID:       runID,
		JobID:       jobID,
		TriggerType: jobrun.TriggerTypeForCronMode(mode),
		TriggerRef:  scenarioKey,
		Input: map[string]interface{}{
			"mode":        mode,
			"domain":      domain,
			"clusterId":   clusterID,
			"component":   component,
			"scenarioKey": scenarioKey,
		},
	})
	if err != nil {
		return ""
	}
	return run.ID
}

func finishCronJobRun(runID, status, errMsg, summary, scenarioKey, domain, clusterID, component, mode string) {
	jr := jobrun.Default()
	if jr == nil || strings.TrimSpace(runID) == "" {
		return
	}
	output := map[string]interface{}{
		"status":      status,
		"summary":     summary,
		"scenarioKey": scenarioKey,
		"domain":      domain,
		"clusterId":   clusterID,
		"component":   component,
		"mode":        mode,
	}
	if status == "error" {
		_ = jr.Fail(runID, errMsg, output)
		return
	}
	_ = jr.Succeed(runID, jobrun.FinishInput{Output: output})
}

// RecomputeNextRuns updates NextRunAtMs for all jobs and persists.
func (s *Service) RecomputeNextRuns() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()
	if s.repo != nil {
		jobs, err := s.repo.List(true)
		if err != nil {
			return err
		}
		for _, j := range jobs {
			next := ComputeNextRunAtMs(j.Schedule, now)
			j.State.NextRunAtMs = &next
			if err := s.repo.Upsert(j); err != nil {
				return err
			}
		}
		return nil
	}

	for i := range s.store.Jobs {
		next := ComputeNextRunAtMs(s.store.Jobs[i].Schedule, now)
		s.store.Jobs[i].State.NextRunAtMs = &next
	}
	return SaveStore(s.storePath, s.store)
}

// NextWakeAtMs returns the soonest next run time in ms, or 0.
func (s *Service) NextWakeAtMs() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var min int64
	if s.repo != nil {
		jobs, err := s.repo.List(true)
		if err == nil {
			for _, j := range jobs {
				if j.Enabled && j.State.NextRunAtMs != nil {
					n := *j.State.NextRunAtMs
					if n > 0 && (min == 0 || n < min) {
						min = n
					}
				}
			}
		}
		return min
	}

	for _, j := range s.store.Jobs {
		if !j.Enabled || j.State.NextRunAtMs == nil {
			continue
		}
		n := *j.State.NextRunAtMs
		if n > 0 && (min == 0 || n < min) {
			min = n
		}
	}
	return min
}

// dueJobIDs returns job IDs that are due (NextRunAtMs <= nowMs). Caller holds no lock.
func (s *Service) dueJobIDs(nowMs int64) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var ids []string
	if s.repo != nil {
		jobs, err := s.repo.List(true)
		if err == nil {
			for _, j := range jobs {
				if j.Enabled && j.State.NextRunAtMs != nil && *j.State.NextRunAtMs <= nowMs {
					ids = append(ids, j.ID)
				}
			}
		}
		return ids
	}

	for _, j := range s.store.Jobs {
		if !j.Enabled || j.State.NextRunAtMs == nil {
			continue
		}
		if *j.State.NextRunAtMs <= nowMs {
			ids = append(ids, j.ID)
		}
	}
	return ids
}

const maxTimerSleepMs = 60000

// Start starts the timer loop (recompute next runs, then sleep/wake and execute due jobs).
// Call Stop() to stop the loop.
func (s *Service) Start() {
	s.mu.Lock()
	if s.done != nil {
		s.mu.Unlock()
		return
	}
	s.done = make(chan struct{})
	s.mu.Unlock()
	_ = s.RecomputeNextRuns()
	go func() {
		for {
			nextMs := s.NextWakeAtMs()
			nowMs := time.Now().UnixMilli()
			sleepMs := int64(maxTimerSleepMs)
			if nextMs > 0 {
				if nextMs <= nowMs {
					// 已到或已过执行时间，立即执行（sleep 0）
					sleepMs = 0
				} else {
					d := nextMs - nowMs
					if d < sleepMs {
						sleepMs = d
					}
				}
			}
			select {
			case <-time.After(time.Duration(sleepMs) * time.Millisecond):
				// fall through and run due jobs
			case <-s.done:
				return
			}
			nowMs = time.Now().UnixMilli()
			dueIds := s.dueJobIDs(nowMs)
			for _, id := range dueIds {
				s.wg.Add(1)
				go func(jobID string) {
					defer s.wg.Done()
					_ = s.Run(jobID, "due", "", "", "", "")
				}(id)
			}
			// 仅在有任务实际执行时重算下次运行时间，避免覆盖「即将到期」任务的 NextRunAtMs
			// （例如因时钟偏差未命中 dueJobIDs，RecomputeNextRuns 会错误跳过该次执行）
			if len(dueIds) > 0 {
				_ = s.RecomputeNextRuns()
			}
		}
	}()
}

// Stop stops the timer loop.
func (s *Service) Stop() {
	s.mu.Lock()
	done := s.done
	s.done = nil
	s.mu.Unlock()
	if done != nil {
		close(done)
	}
	s.wg.Wait()
}

// ensureDefaultJobs populates 5 default health inspection cron jobs for Hadoop, FI, GBase, Governance, and DataApps.
func (s *Service) ensureDefaultJobs() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UnixMilli()

	defaultJobs := []struct {
		ID          string
		Name        string
		Description string
		Message     string
	}{
		{
			ID:          "job-inspect-hadoop",
			Name:        "深度健康巡检 - Hadoop",
			Description: "每天定时对开源 Hadoop 生态集群的 CPU、内存、HDFS 及 YARN 状态进行深度巡检",
			Message: `你是一个大数据运维专家。请调用 query_vm_metrics 工具对 Hadoop 集群的健康度进行深度巡检。
请运行以下 VictoriaMetrics PromQL 查询：
- YARN 节点总数及活跃节点数：sum(yarn_resourcemanager_active_nodes)
- HDFS 存储容量使用率：sum(hadoop_namenode_dfs_used_percentage)
- NameNode JVM 堆内存使用情况：hadoop_namenode_jvm_heap_used_bytes / hadoop_namenode_jvm_heap_max_bytes
- YARN 活跃 Container 数：sum(yarn_resourcemanager_allocated_containers)
并运行针对 node_cpu_seconds_total 的平均 CPU 负载查询。
请根据查询结果分析 Hadoop 集群运行状态，给出 0-100 的健康得分（格式如：健康得分：XX），编写结构化的中文巡检报告，包括指标详情、异常诊断和修复优化建议。`,
		},
		{
			ID:          "job-inspect-fi",
			Name:        "深度健康巡检 - FI 商业生态",
			Description: "每天定时对 FusionInsight 商业大数据平台健康度及核心组件进行深度巡检",
			Message: `你是一个 FusionInsight (FI) 商业大数据平台运维专家。请调用 query_vm_metrics 工具对 FusionInsight 集群健康度进行深度巡检。
请运行以下 VictoriaMetrics PromQL 查询：
- FI HBase RegionServer 活跃数：sum(fi_hbase_regionserver_active_count)
- FI YARN 资源队列占比：sum(fi_yarn_queue_allocated_memory_bytes) by (queue)
根据结果评估集群健康状况，给出 0-100 的健康得分（格式如：健康得分：XX），编写结构化的中文巡检报告，包括主备互信状态、指标详情和优化建议。`,
		},
		{
			ID:          "job-inspect-gbase",
			Name:        "深度健康巡检 - GBase 数据库",
			Description: "每天定时对 GBase 数据库连接数、慢 SQL 及吞吐量指标进行深度巡检",
			Message: `你是一个 GBase 数据库专家。请调用 query_vm_metrics 工具对 GBase 数据库进行巡检。
请首先调用 query_gbase_slow_sql 获取慢 SQL 列表，再调用 query_vm_metrics 运行以下 VictoriaMetrics PromQL 查询：
- GBase 活跃连接数：sum(gbase_active_connections)
- 慢 SQL 数量：sum(gbase_slow_queries_total)
- QPS (每秒查询数) 或 TPS (每秒事务数)：rate(gbase_queries_total[5m])
评估数据库运行健康度，并在最终回答中优先输出一个 JSON 代码块，字段包括 domain、clusterId、score、scoreStatus、toolRuns、metricsEvidence、errors、reportMarkdown；随后可补充中文说明。不要只写自然语言“健康得分”。`,
		},
		{
			ID:          "job-inspect-governance",
			Name:        "深度健康巡检 - 开发治理平台",
			Description: "每天定时对开发治理平台服务 API 成功率、元数据血缘及质量校验告警进行深度巡检",
			Message: `你是一个数据开发治理平台运维专家。请首先调用 query_governance_lineage 工具查询元数据链路健康度和规则告警，再调用 query_vm_metrics 运行以下 VictoriaMetrics PromQL 查询：
- 治理平台服务 API 成功率：rate(governance_api_requests_total{status='200'}[5m]) / rate(governance_api_requests_total[5m])
- 数据质量规则校验告警数：sum(governance_quality_alerts_total)
评估治理平台的稳定性和数据资产健康度，给出 0-100 的健康得分（格式如：健康得分：XX），编写结构化的中文巡检报告，包括元数据血缘、质量稽核及资产优化建议。`,
		},
		{
			ID:          "job-inspect-dataapps",
			Name:        "深度健康巡检 - 数据 App 运维",
			Description: "每天定时对 30 多个数据 App 跑批 SLA、失败链路及数据管道时延进行深度巡检",
			Message: `你是一个大数据应用运维专家，负责 30 多个数据 App（调度任务、报表链路等）的运维。请调用 query_vm_metrics 工具对数据 App 链路健康度进行深度巡检。
请运行以下 VictoriaMetrics PromQL 查询：
- 调度系统失败任务数：sum(dataapps_scheduler_failed_tasks)
- 核心数据报表跑批时延/完成度：sum(dataapps_pipeline_delay_seconds)
评估各数据应用的可用性及跑批 SLA，给出 0-100 的健康得分（格式如：健康得分：XX），编写结构化的中文巡检报告，包括跑批 SLA 达标情况、失败链路根因及运维改进建议。`,
		},
	}

	if s.repo != nil {
		for _, dj := range defaultJobs {
			j, found, err := s.repo.Get(dj.ID)
			if err != nil {
				return err
			}
			empID := ""
			if dj.ID == "job-inspect-hadoop" || dj.ID == "job-inspect-fi" {
				empID = "emp_bch_inspect"
			} else {
				empID = "emp_bch_diagnose"
			}
			if found {
				if j.DigitalEmployeeID == "" {
					j.DigitalEmployeeID = empID
					j.UpdatedAtMs = now
					if err := s.repo.Upsert(j); err != nil {
						return err
					}
				}
				continue
			}
			sched := CronSchedule{Kind: "cron", Expr: "0 8,20 * * *"}
			next := ComputeNextRunAtMs(sched, now)
			newJob := CronJob{
				ID:                dj.ID,
				AgentID:           "main",
				Name:              dj.Name,
				Description:       dj.Description,
				Enabled:           true,
				CreatedAtMs:       now,
				UpdatedAtMs:       now,
				DigitalEmployeeID: empID,
				Schedule:          sched,
				SessionTarget:     "isolated",
				SessionKey:        "agent:main:cron:" + dj.ID,
				WakeMode:          "next-heartbeat",
				Payload: CronPayload{
					Kind:    "agentTurn",
					Message: dj.Message,
				},
				State: CronJobState{
					NextRunAtMs: &next,
				},
			}
			if err := s.repo.Upsert(newJob); err != nil {
				return err
			}
		}
		return s.ensureDefaultBatchL0Jobs(now)
	}

	changed := false
	for _, dj := range defaultJobs {
		found := false
		for i, j := range s.store.Jobs {
			if j.ID == dj.ID {
				found = true
				empID := ""
				if dj.ID == "job-inspect-hadoop" || dj.ID == "job-inspect-fi" {
					empID = "emp_bch_inspect"
				} else {
					empID = "emp_bch_diagnose"
				}
				if s.store.Jobs[i].DigitalEmployeeID == "" {
					s.store.Jobs[i].DigitalEmployeeID = empID
					s.store.Jobs[i].UpdatedAtMs = now
					changed = true
				}
				break
			}
		}
		if !found {
			sched := CronSchedule{Kind: "cron", Expr: "0 8,20 * * *"}
			next := ComputeNextRunAtMs(sched, now)
			empID := ""
			if dj.ID == "job-inspect-hadoop" || dj.ID == "job-inspect-fi" {
				empID = "emp_bch_inspect"
			} else {
				empID = "emp_bch_diagnose"
			}
			j := CronJob{
				ID:                dj.ID,
				AgentID:           "main",
				Name:              dj.Name,
				Description:       dj.Description,
				Enabled:           true,
				CreatedAtMs:       now,
				UpdatedAtMs:       now,
				DigitalEmployeeID: empID,
				Schedule:          sched,
				SessionTarget:     "isolated",
				SessionKey:        "agent:main:cron:" + dj.ID,
				WakeMode:          "next-heartbeat",
				Payload: CronPayload{
					Kind:    "agentTurn",
					Message: dj.Message,
				},
				State: CronJobState{
					NextRunAtMs: &next,
				},
			}
			s.store.Jobs = append(s.store.Jobs, j)
			changed = true
		}
	}

	if changed {
		if err := SaveStore(s.storePath, s.store); err != nil {
			return err
		}
	}
	return s.ensureDefaultBatchL0Jobs(now)
}

type defaultBatchL0Job struct {
	ID          string
	Name        string
	Description string
	Message     string
	Schedule    string
	EmployeeID  string
}

func defaultBatchL0Jobs() []defaultBatchL0Job {
	return []defaultBatchL0Job{
		{
			ID: "job-inspect-flink", Name: "Flink 作业批量健康巡检",
			Description: "每小时对 Flink 作业进行 L0 批量指标采集与规则评分",
			Message:     "Flink L0 批量巡检：由 Work Queue 执行 flink_metrics_batch 采集与评分，异常作业条件升级 L2。",
			Schedule:    "0 * * * *", EmployeeID: "emp_bch_inspect",
		},
		{
			ID: "job-inspect-spark", Name: "Spark 作业批量健康巡检",
			Description: "每小时对 Spark 作业进行 L0 批量采集与倾斜/失败规则评分",
			Message:     "Spark L0 批量巡检：由 Work Queue 执行 spark_metrics_batch 采集与评分，异常作业条件升级 L2。",
			Schedule:    "15 * * * *", EmployeeID: "emp_bch_inspect",
		},
		{
			ID: "job-inspect-yarn", Name: "YARN 队列容量批量评估",
			Description: "每 2 小时对 YARN 队列进行 L0 容量水位批量评估",
			Message:     "YARN L0 批量巡检：由 Work Queue 执行 yarn_queue_batch 采集与评分，异常队列条件升级 L2。",
			Schedule:    "0 */2 * * *", EmployeeID: "emp_bch_inspect",
		},
		{
			ID: "job-inspect-gbase-instances", Name: "GBase 实例批量健康巡检",
			Description: "每 4 小时对 GBase 实例连接与慢 SQL 进行 L0 批量评估",
			Message:     "GBase 实例 L0 批量巡检：由 Work Queue 执行 gbase_instance_batch 采集与评分，异常实例条件升级 L2。",
			Schedule:    "0 */4 * * *", EmployeeID: "emp_gbase_diagnose",
		},
		{
			ID: "job-inspect-dataapps-pipelines", Name: "数据 App 管道批量 SLA 巡检",
			Description: "每小时对数据 App 管道跑批 SLA 与失败链路进行 L0 批量评估",
			Message:     "DataApp 管道 L0 批量巡检：由 Work Queue 执行 pipeline_batch 采集与评分，异常管道条件升级 L2。",
			Schedule:    "30 * * * *", EmployeeID: "emp_dataapps_ops",
		},
	}
}

// ensureDefaultBatchL0Jobs adds default L0 batch inspection cron jobs (Phase B/D).
func (s *Service) ensureDefaultBatchL0Jobs(now int64) error {
	for _, dj := range defaultBatchL0Jobs() {
		if err := s.ensureOneBatchL0Job(now, dj); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) ensureOneBatchL0Job(now int64, dj defaultBatchL0Job) error {
	sched := CronSchedule{Kind: "cron", Expr: dj.Schedule}
	empID := dj.EmployeeID
	if empID == "" {
		empID = "emp_bch_inspect"
	}

	if s.repo != nil {
		j, found, err := s.repo.Get(dj.ID)
		if err != nil {
			return err
		}
		if found {
			if j.DigitalEmployeeID == "" {
				j.DigitalEmployeeID = empID
				j.UpdatedAtMs = now
				if err := s.repo.Upsert(j); err != nil {
					return err
				}
			}
			return nil
		}
		next := ComputeNextRunAtMs(sched, now)
		return s.repo.Upsert(CronJob{
			ID: dj.ID, AgentID: "main", Name: dj.Name, Description: dj.Description,
			Enabled: true, CreatedAtMs: now, UpdatedAtMs: now, DigitalEmployeeID: empID,
			Schedule: sched, SessionTarget: "isolated", SessionKey: "agent:main:cron:" + dj.ID,
			WakeMode: "next-heartbeat",
			Payload:  CronPayload{Kind: "agentTurn", Message: dj.Message},
			State:    CronJobState{NextRunAtMs: &next},
		})
	}

	for i, j := range s.store.Jobs {
		if j.ID == dj.ID {
			if s.store.Jobs[i].DigitalEmployeeID == "" {
				s.store.Jobs[i].DigitalEmployeeID = empID
				s.store.Jobs[i].UpdatedAtMs = now
				return SaveStore(s.storePath, s.store)
			}
			return nil
		}
	}
	next := ComputeNextRunAtMs(sched, now)
	s.store.Jobs = append(s.store.Jobs, CronJob{
		ID: dj.ID, AgentID: "main", Name: dj.Name, Description: dj.Description,
		Enabled: true, CreatedAtMs: now, UpdatedAtMs: now, DigitalEmployeeID: empID,
		Schedule: sched, SessionTarget: "isolated", SessionKey: "agent:main:cron:" + dj.ID,
		WakeMode: "next-heartbeat",
		Payload:  CronPayload{Kind: "agentTurn", Message: dj.Message},
		State:    CronJobState{NextRunAtMs: &next},
	})
	return SaveStore(s.storePath, s.store)
}
