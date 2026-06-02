package handlers

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/employees"
	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// EmployeeTasksListHandler handles "employee.tasks.list".
func EmployeeTasksListHandler(opts HandlerOpts) error {
	env := func(k string) string { return os.Getenv(k) }
	tasks, err := employees.ListTasks(env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employee.tasks.list: " + err.Error(),
		}, nil)
		return nil
	}

	// Apply filters if passed
	filtered := []employees.EmployeeTask{}
	empIDFilter, _ := opts.Params["employeeId"].(string)
	empIDFilter = strings.TrimSpace(strings.ToLower(empIDFilter))

	domainFilter, _ := opts.Params["domainKey"].(string)
	domainFilter = strings.TrimSpace(strings.ToLower(domainFilter))

	capFilter, _ := opts.Params["capabilityKey"].(string)
	capFilter = strings.TrimSpace(strings.ToLower(capFilter))

	statusFilter, _ := opts.Params["status"].(string)
	statusFilter = strings.TrimSpace(strings.ToLower(statusFilter))

	queryFilter, _ := opts.Params["query"].(string)
	queryFilter = strings.TrimSpace(strings.ToLower(queryFilter))

	for _, t := range tasks {
		if empIDFilter != "" && !strings.Contains(strings.ToLower(t.EmployeeID), empIDFilter) {
			continue
		}
		if domainFilter != "" && !strings.Contains(strings.ToLower(t.DomainKey), domainFilter) {
			continue
		}
		if capFilter != "" && !strings.Contains(strings.ToLower(t.CapabilityKey), capFilter) {
			continue
		}
		if statusFilter != "" &&
			strings.ToLower(strings.TrimSpace(t.Status)) != statusFilter &&
			strings.ToLower(strings.TrimSpace(t.ExecutionStatus)) != statusFilter &&
			strings.ToLower(strings.TrimSpace(t.WorkflowStatus)) != statusFilter {
			continue
		}
		if queryFilter != "" &&
			!strings.Contains(strings.ToLower(t.Input), queryFilter) &&
			!strings.Contains(strings.ToLower(t.Output), queryFilter) &&
			!strings.Contains(strings.ToLower(t.Conclusion), queryFilter) &&
			!strings.Contains(strings.ToLower(t.ID), queryFilter) {
			continue
		}
		filtered = append(filtered, t)
	}

	opts.Respond(true, map[string]interface{}{
		"tasks": filtered,
	}, nil, nil)
	return nil
}

// EmployeeTasksGetHandler handles "employee.tasks.get".
func EmployeeTasksGetHandler(opts HandlerOpts) error {
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employee.tasks.get: id required",
		}, nil)
		return nil
	}

	env := func(k string) string { return os.Getenv(k) }
	t, err := employees.LoadTask(id, env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: "employee.tasks.get: task not found",
		}, nil)
		return nil
	}

	opts.Respond(true, t, nil, nil)
	return nil
}

// EmployeeTasksCreateHandler handles "employee.tasks.create".
func EmployeeTasksCreateHandler(opts HandlerOpts) error {
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id != "" && !employees.IsValidTaskID(id) {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employee.tasks.create: invalid id",
		}, nil)
		return nil
	}

	empID, _ := opts.Params["employeeId"].(string)
	domainKey, _ := opts.Params["domainKey"].(string)
	capKey, _ := opts.Params["capabilityKey"].(string)
	scenarioKey, _ := opts.Params["scenarioKey"].(string)
	objectRef, _ := opts.Params["objectRef"].(string)
	triggerType, _ := opts.Params["triggerType"].(string)
	status, _ := opts.Params["status"].(string)
	executionStatus, _ := opts.Params["executionStatus"].(string)
	workflowStatus, _ := opts.Params["workflowStatus"].(string)
	input, _ := opts.Params["input"].(string)
	output, _ := opts.Params["output"].(string)
	conclusion, _ := opts.Params["conclusion"].(string)
	operator, _ := opts.Params["operator"].(string)
	evaluation, _ := opts.Params["evaluation"].(string)

	var artifacts []string
	if rawArts, ok := opts.Params["artifacts"].([]interface{}); ok {
		for _, v := range rawArts {
			if s, ok := v.(string); ok && s != "" {
				artifacts = append(artifacts, s)
			}
		}
	}

	startedAtVal, _ := opts.Params["startedAt"].(float64)
	startedAt := int64(startedAtVal)
	if startedAt == 0 {
		startedAt = time.Now().UnixMilli()
	}

	finishedAtVal, _ := opts.Params["finishedAt"].(float64)
	finishedAt := int64(finishedAtVal)

	task := employees.EmployeeTask{
		ID:              id,
		EmployeeID:      empID,
		DomainKey:       domainKey,
		CapabilityKey:   capKey,
		ScenarioKey:     scenarioKey,
		ObjectRef:       objectRef,
		TriggerType:     triggerType,
		ExecutionStatus: employees.NormalizeExecutionStatus(firstNonEmptyString(executionStatus, status)),
		WorkflowStatus:  employees.NormalizeWorkflowStatus(workflowStatus),
		Status:          employees.LegacyStatusFromExecution(firstNonEmptyString(executionStatus, status)),
		Input:           input,
		Output:          output,
		Conclusion:      conclusion,
		Artifacts:       artifacts,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		Operator:        operator,
		Evaluation:      evaluation,
		Metrics:         taskMetricsFromParams(opts.Params, artifacts),
	}

	if task.Evaluation == "" {
		task.Evaluation = employees.EvaluationUnrated
	}

	env := func(k string) string { return os.Getenv(k) }
	if err := employees.SaveTask(&task, env); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employee.tasks.create: " + err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"id": task.ID}, nil, nil)
	return nil
}

// EmployeeTasksUpdateHandler handles "employee.tasks.update".
func EmployeeTasksUpdateHandler(opts HandlerOpts) error {
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employee.tasks.update: id required",
		}, nil)
		return nil
	}

	env := func(k string) string { return os.Getenv(k) }
	task, err := employees.LoadTask(id, env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: "employee.tasks.update: task not found",
		}, nil)
		return nil
	}

	if status, ok := opts.Params["status"].(string); ok {
		task.ExecutionStatus = employees.ExecutionFromLegacyStatus(status)
		task.Status = employees.LegacyStatusFromExecution(task.ExecutionStatus)
	}
	if executionStatus, ok := opts.Params["executionStatus"].(string); ok {
		task.ExecutionStatus = employees.NormalizeExecutionStatus(executionStatus)
		task.Status = employees.LegacyStatusFromExecution(task.ExecutionStatus)
	}
	if workflowStatus, ok := opts.Params["workflowStatus"].(string); ok {
		task.WorkflowStatus = employees.NormalizeWorkflowStatus(workflowStatus)
	}
	if input, ok := opts.Params["input"].(string); ok {
		task.Input = input
	}
	if output, ok := opts.Params["output"].(string); ok {
		task.Output = output
	}
	if conclusion, ok := opts.Params["conclusion"].(string); ok {
		task.Conclusion = conclusion
	}
	if evaluation, ok := opts.Params["evaluation"].(string); ok {
		task.Evaluation = strings.ToLower(strings.TrimSpace(evaluation))
		if task.Evaluation == employees.EvaluationAccepted {
			task.WorkflowStatus = employees.WorkflowClosed
		} else if task.Evaluation == employees.EvaluationRejected {
			task.WorkflowStatus = employees.WorkflowRejected
		}
	}
	if finishedAtVal, ok := opts.Params["finishedAt"].(float64); ok {
		task.FinishedAt = int64(finishedAtVal)
	} else if task.FinishedAt == 0 && task.ExecutionStatus != employees.ExecutionPending && task.ExecutionStatus != employees.ExecutionRunning {
		task.FinishedAt = time.Now().UnixMilli()
	}

	if rawArts, ok := opts.Params["artifacts"].([]interface{}); ok {
		var artifacts []string
		for _, v := range rawArts {
			if s, ok := v.(string); ok && s != "" {
				artifacts = append(artifacts, s)
			}
		}
		task.Artifacts = artifacts
		task.Metrics = taskMetricsFromParams(opts.Params, artifacts)
	}
	if rawMetrics, ok := opts.Params["metrics"].(map[string]interface{}); ok {
		task.Metrics = taskMetricsFromMap(rawMetrics, task.Metrics)
	}

	if err := employees.SaveTask(task, env); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employee.tasks.update: " + err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"id": task.ID}, nil, nil)
	return nil
}

// EmployeeTasksDeleteHandler handles "employee.tasks.delete".
func EmployeeTasksDeleteHandler(opts HandlerOpts) error {
	id, _ := opts.Params["id"].(string)
	id = strings.TrimSpace(id)
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employee.tasks.delete: id required",
		}, nil)
		return nil
	}

	env := func(k string) string { return os.Getenv(k) }
	if err := employees.DeleteTask(id, env); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employee.tasks.delete: " + err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true}, nil, nil)
	return nil
}

// EmployeeEffectivenessGetHandler handles "employee.effectiveness.get".
func EmployeeEffectivenessGetHandler(opts HandlerOpts) error {
	env := func(k string) string { return os.Getenv(k) }
	tasks, err := employees.ListTasks(env)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "employee.effectiveness.get: " + err.Error(),
		}, nil)
		return nil
	}

	taskCount := len(tasks)
	completedCount := 0
	closedCount := 0
	ratedCount := 0
	acceptedCount := 0

	taskBreakdown := emptyCapabilityBreakdown()
	domainBreakdown := emptyDomainBreakdown()

	savedHours := 0.0
	totalRawAlerts := 0
	totalReducedAlerts := 0
	alertMetricSamples := 0
	costSpent := 0.0
	mttrSamples := 0
	totalMTTRMs := int64(0)

	for _, t := range tasks {
		if t.ExecutionStatus == employees.ExecutionSucceeded || t.ExecutionStatus == employees.ExecutionFailed {
			completedCount++
		}
		if t.WorkflowStatus == employees.WorkflowClosed {
			closedCount++
		}
		eval := strings.ToLower(t.Evaluation)
		if eval == employees.EvaluationAccepted || eval == employees.EvaluationRejected {
			ratedCount++
			if eval == employees.EvaluationAccepted {
				acceptedCount++
			}
		}

		capKey := employees.NormalizeCapabilityKey(t.CapabilityKey)
		taskBreakdown[capKey]++
		if t.Metrics.RawAlertCount > 0 && t.Metrics.ReducedAlertCount >= 0 && t.Metrics.ReducedAlertCount <= t.Metrics.RawAlertCount {
			totalRawAlerts += t.Metrics.RawAlertCount
			totalReducedAlerts += t.Metrics.ReducedAlertCount
			alertMetricSamples++
		}
		savedHours += t.Metrics.SavedHours
		costSpent += t.Metrics.CostUSD
		if t.Metrics.MTTRMs > 0 {
			totalMTTRMs += t.Metrics.MTTRMs
			mttrSamples++
		}

		domKey := employees.NormalizeDomainKey(t.DomainKey)
		domainBreakdown[domKey]++
	}

	autoCloseRate := 0.0
	if completedCount > 0 {
		autoCloseRate = float64(closedCount) / float64(completedCount)
	}

	adoptionRate := 0.0
	if ratedCount > 0 {
		adoptionRate = float64(acceptedCount) / float64(ratedCount)
	}

	noiseReductionRate := 0.0
	if totalRawAlerts > 0 {
		noiseReductionRate = float64(totalRawAlerts-totalReducedAlerts) / float64(totalRawAlerts)
	}
	avgMTTRMs := int64(0)
	if mttrSamples > 0 {
		avgMTTRMs = totalMTTRMs / int64(mttrSamples)
	}
	metricConfidence := "measured"
	if alertMetricSamples == 0 && savedHours == 0 && costSpent == 0 && mttrSamples == 0 {
		metricConfidence = "insufficient_data"
	}

	opts.Respond(true, map[string]interface{}{
		"taskCount":          taskCount,
		"completedTaskCount": completedCount,
		"closedTaskCount":    closedCount,
		"autoCloseRate":      autoCloseRate,
		"adoptionRate":       adoptionRate,
		"noiseReductionRate": noiseReductionRate,
		"rawAlertCount":      totalRawAlerts,
		"reducedAlertCount":  totalReducedAlerts,
		"alertMetricSamples": alertMetricSamples,
		"savedHours":         savedHours,
		"costSpent":          costSpent,
		"avgMttrMs":          avgMTTRMs,
		"mttrSamples":        mttrSamples,
		"metricConfidence":   metricConfidence,
		"taskBreakdown":      taskBreakdown,
		"domainBreakdown":    domainBreakdown,
	}, nil, nil)
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func emptyCapabilityBreakdown() map[string]int {
	out := map[string]int{}
	for _, key := range employees.CanonicalCapabilityKeys {
		out[key] = 0
	}
	return out
}

func emptyDomainBreakdown() map[string]int {
	out := map[string]int{}
	for _, key := range employees.CanonicalDomainKeys {
		out[key] = 0
	}
	return out
}

func taskMetricsFromParams(params map[string]interface{}, artifacts []string) employees.EmployeeTaskMetrics {
	metrics := taskMetricsFromArtifacts(artifacts)
	if rawMetrics, ok := params["metrics"].(map[string]interface{}); ok {
		metrics = taskMetricsFromMap(rawMetrics, metrics)
	}
	return metrics
}

func taskMetricsFromMap(raw map[string]interface{}, base employees.EmployeeTaskMetrics) employees.EmployeeTaskMetrics {
	if v, ok := raw["rawAlertCount"].(float64); ok && v >= 0 {
		base.RawAlertCount = int(v)
	}
	if v, ok := raw["reducedAlertCount"].(float64); ok && v >= 0 {
		base.ReducedAlertCount = int(v)
	}
	if v, ok := raw["savedHours"].(float64); ok && v >= 0 {
		base.SavedHours = v
	}
	if v, ok := raw["costUsd"].(float64); ok && v >= 0 {
		base.CostUSD = v
	}
	if v, ok := raw["costUSD"].(float64); ok && v >= 0 {
		base.CostUSD = v
	}
	if v, ok := raw["mttaMs"].(float64); ok && v >= 0 {
		base.MTTAMs = int64(v)
	}
	if v, ok := raw["mttrMs"].(float64); ok && v >= 0 {
		base.MTTRMs = int64(v)
	}
	return base
}

func taskMetricsFromArtifacts(artifacts []string) employees.EmployeeTaskMetrics {
	return employees.EmployeeTaskMetrics{
		RawAlertCount:     int(parseTaskArtifactNumber(artifacts, "alerts_raw:", "raw_alerts:")),
		ReducedAlertCount: int(parseTaskArtifactNumber(artifacts, "alerts_reduced:", "reduced_alerts:", "alerts_deduped:")),
		SavedHours:        parseTaskArtifactNumber(artifacts, "saved_hours:", "savedHours:"),
		CostUSD:           parseTaskCostUSD(artifacts),
		MTTAMs:            int64(parseTaskArtifactNumber(artifacts, "mtta_ms:", "mttaMs:")),
		MTTRMs:            int64(parseTaskArtifactNumber(artifacts, "mttr_ms:", "mttrMs:")),
	}
}

func parseTaskArtifactNumber(artifacts []string, prefixes ...string) float64 {
	for _, raw := range artifacts {
		v := strings.TrimSpace(raw)
		lower := strings.ToLower(v)
		for _, prefix := range prefixes {
			p := strings.ToLower(prefix)
			if strings.HasPrefix(lower, p) {
				num := strings.TrimSpace(v[len(prefix):])
				if f, err := strconv.ParseFloat(num, 64); err == nil && f >= 0 {
					return f
				}
			}
		}
	}
	return 0
}

func parseTaskCostUSD(artifacts []string) float64 {
	for _, raw := range artifacts {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(v), "cost_usd:") {
			num := strings.TrimSpace(v[len("cost_usd:"):])
			if f, err := strconv.ParseFloat(num, 64); err == nil && f > 0 {
				return f
			}
		}
		if strings.HasPrefix(strings.ToLower(v), "cost:") {
			num := strings.TrimSpace(v[len("cost:"):])
			if f, err := strconv.ParseFloat(num, 64); err == nil && f > 0 {
				return f
			}
		}
	}
	return 0
}
