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
		if statusFilter != "" && strings.ToLower(strings.TrimSpace(t.Status)) != statusFilter {
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
	if id == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "employee.tasks.create: id required",
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
		ID:            id,
		EmployeeID:    empID,
		DomainKey:     domainKey,
		CapabilityKey: capKey,
		ScenarioKey:   scenarioKey,
		ObjectRef:     objectRef,
		TriggerType:   triggerType,
		Status:        status,
		Input:         input,
		Output:        output,
		Conclusion:    conclusion,
		Artifacts:     artifacts,
		StartedAt:     startedAt,
		FinishedAt:    finishedAt,
		Operator:      operator,
		Evaluation:    evaluation,
	}

	if task.Evaluation == "" {
		task.Evaluation = "unrated"
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
		task.Status = status
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
		task.Evaluation = evaluation
	}
	if finishedAtVal, ok := opts.Params["finishedAt"].(float64); ok {
		task.FinishedAt = int64(finishedAtVal)
	} else if task.FinishedAt == 0 && task.Status != "pending" && task.Status != "running" {
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
	successCount := 0
	ratedCount := 0
	acceptedCount := 0

	taskBreakdown := map[string]int{
		"observability-alert":       0,
		"health-inspection":         0,
		"diagnosis-incident":        0,
		"governance-optimization":   0,
		"capacity-performance-cost": 0,
		"change-config-compliance":  0,
	}

	domainBreakdown := map[string]int{
		"hadoop":     0,
		"fi":         0,
		"gbase":      0,
		"governance": 0,
		"data-apps":  0,
	}

	savedHours := 0.0
	observableTaskCount := 0
	observableSuccessCount := 0
	costSpent := 0.0

	for _, t := range tasks {
		if strings.EqualFold(t.Status, "success") {
			successCount++
		}
		eval := strings.ToLower(t.Evaluation)
		if eval == "accepted" || eval == "rejected" {
			ratedCount++
			if eval == "accepted" {
				acceptedCount++
			}
		}

		// Capability breakdown & saved hours
		capKey := t.CapabilityKey
		if capKey == "" {
			capKey = "observability-alert" // default fallback
		}
		taskBreakdown[capKey]++
		if capKey == "observability-alert" {
			observableTaskCount++
			if strings.EqualFold(t.Status, "success") {
				observableSuccessCount++
			}
		}

		switch capKey {
		case "observability-alert":
			savedHours += 0.5
		case "health-inspection":
			savedHours += 2.0
		case "diagnosis-incident":
			savedHours += 1.5
		case "governance-optimization":
			savedHours += 3.0
		case "capacity-performance-cost":
			savedHours += 4.0
		case "change-config-compliance":
			savedHours += 2.5
		default:
			savedHours += 1.0
		}

		// Domain breakdown
		domKey := t.DomainKey
		if domKey == "" {
			domKey = "hadoop"
		}
		domainBreakdown[domKey]++
		costSpent += parseTaskCostUSD(t.Artifacts)
	}

	autoCloseRate := 0.0
	if taskCount > 0 {
		autoCloseRate = float64(successCount) / float64(taskCount)
	}

	adoptionRate := 0.0
	if ratedCount > 0 {
		adoptionRate = float64(acceptedCount) / float64(ratedCount)
	}

	// 基于真实任务结果计算：观测告警任务中成功闭环占比。
	noiseReductionRate := 0.0
	if observableTaskCount > 0 {
		noiseReductionRate = float64(observableSuccessCount) / float64(observableTaskCount)
	}

	opts.Respond(true, map[string]interface{}{
		"taskCount":          taskCount,
		"autoCloseRate":      autoCloseRate,
		"adoptionRate":       adoptionRate,
		"noiseReductionRate": noiseReductionRate,
		"savedHours":         savedHours,
		"costSpent":          costSpent,
		"taskBreakdown":      taskBreakdown,
		"domainBreakdown":    domainBreakdown,
	}, nil, nil)
	return nil
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
