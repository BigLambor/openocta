package handlers

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/openocta/openocta/pkg/cron"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/ops"
)

// getJobID returns id or jobId from params (id takes precedence). Used by cron.update/remove/run/runs.
func getJobID(params map[string]interface{}) string {
	if id, ok := params["id"].(string); ok && id != "" {
		return id
	}
	if id, ok := params["jobId"].(string); ok && id != "" {
		return id
	}
	return ""
}

// CronListHandler handles "cron.list".
func CronListHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	includeDisabled := false
	if v, ok := opts.Params["includeDisabled"].(bool); ok {
		includeDisabled = v
	}
	jobs, err := ctx.CronService.List(includeDisabled)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"jobs": jobs}, nil, nil)
	return nil
}

// CronStatusHandler handles "cron.status". Returns status object matching TS: enabled, storePath, jobs (count), nextWakeAtMs.
func CronStatusHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	jobs, err := ctx.CronService.List(true)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	storePath := ""
	if ctx.GetCronStorePath != nil {
		storePath = ctx.GetCronStorePath()
	}
	nextWakeAtMs := interface{}(nil)
	if svc, ok := ctx.CronService.(interface{ NextWakeAtMs() int64 }); ok {
		if n := svc.NextWakeAtMs(); n > 0 {
			nextWakeAtMs = n
		}
	}
	opts.Respond(true, map[string]interface{}{
		"enabled":      true,
		"storePath":    storePath,
		"jobs":         len(jobs),
		"nextWakeAtMs": nextWakeAtMs,
	}, nil, nil)
	return nil
}

// validateScheduleTimestamp checks schedule.at (kind=at) for validity; other kinds pass. Matches TS validateScheduleTimestamp.
func validateScheduleTimestamp(s cron.CronSchedule) (ok bool, msg string) {
	if s.Kind != "at" {
		return true, ""
	}
	if s.At == "" {
		return false, "Invalid schedule.at: expected ISO-8601 timestamp (got empty)"
	}
	// Full ISO/absolute parsing would require a time package; for now require non-empty.
	return true, ""
}

// CronAddHandler handles "cron.add". Returns job directly (same as TS).
func CronAddHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	params := opts.Params
	name, _ := params["name"].(string)
	if name == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "name required",
		}, nil)
		return nil
	}
	description, _ := params["description"].(string)
	agentID, _ := params["agentId"].(string)
	sched := parseSchedule(params["schedule"])
	if ok, msg := validateScheduleTimestamp(sched); !ok {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: msg,
		}, nil)
		return nil
	}
	payload := parsePayload(params["payload"])
	sessionTarget := "main"
	if v, ok := params["sessionTarget"].(string); ok && v != "" {
		sessionTarget = v
	}
	wakeMode := "next-heartbeat"
	if v, ok := params["wakeMode"].(string); ok && (v == "now" || v == "next-heartbeat") {
		wakeMode = v
	}
	enabled := true
	if v, ok := params["enabled"].(bool); ok {
		enabled = v
	}
	delivery := parseDelivery(params["delivery"])
	sessionKey := ""
	if v, ok := params["sessionKey"].(string); ok {
		sessionKey = strings.TrimSpace(v)
	}
	digitalEmployeeID := ""
	if v, ok := params["digitalEmployeeId"].(string); ok {
		digitalEmployeeID = strings.TrimSpace(v)
	}
	j, err := ctx.CronService.Add(cron.JobCreate{
		Name:              name,
		Description:       description,
		AgentID:           agentID,
		Schedule:          sched,
		Payload:           payload,
		SessionTarget:     sessionTarget,
		SessionKey:        sessionKey,
		DigitalEmployeeID: digitalEmployeeID,
		WakeMode:          wakeMode,
		Enabled:           enabled,
		Delivery:          delivery,
	})
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, j, nil, nil)
	return nil
}

// CronRemoveHandler handles "cron.remove". Accepts id or jobId; returns { ok, removed } like TS.
func CronRemoveHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	jobID := getJobID(opts.Params)
	if jobID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "invalid cron.remove params: missing id/jobId",
		}, nil)
		return nil
	}

	removed := true
	if svc, ok := ctx.CronService.(interface {
		GetJob(string) (cron.CronJob, bool)
	}); ok {
		_, found := svc.GetJob(jobID)
		removed = found
	}

	err := ctx.CronService.Remove(jobID)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"ok": true, "removed": removed}, nil, nil)
	return nil
}

func parseSchedule(v interface{}) cron.CronSchedule {
	if v == nil {
		return cron.CronSchedule{}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return cron.CronSchedule{}
	}
	s := cron.CronSchedule{}
	if k, ok := m["kind"].(string); ok {
		s.Kind = k
	}
	if at, ok := m["at"].(string); ok {
		s.At = at
	}
	if every, ok := m["everyMs"].(float64); ok {
		s.EveryMs = int64(every)
	} else if every, ok := m["everyMs"].(int); ok {
		s.EveryMs = int64(every)
	}
	if anchor, ok := m["anchorMs"].(float64); ok && anchor > 0 {
		v := int64(anchor)
		s.AnchorMs = &v
	} else if anchor, ok := m["anchorMs"].(int64); ok && anchor > 0 {
		v := anchor
		s.AnchorMs = &v
	} else if anchor, ok := m["anchorMs"].(int); ok && anchor > 0 {
		v := int64(anchor)
		s.AnchorMs = &v
	}
	if expr, ok := m["expr"].(string); ok {
		s.Expr = expr
	}
	if tz, ok := m["tz"].(string); ok {
		s.Tz = tz
	}
	return s
}

// CronUpdateHandler handles "cron.update". Accepts id or jobId; returns job directly (same as TS). Patch may include schedule.
func CronUpdateHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	svc, ok := ctx.CronService.(interface {
		Update(string, cron.JobPatch) (cron.CronJob, error)
	})
	if !ok {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service does not support update",
		}, nil)
		return nil
	}
	jobID := getJobID(opts.Params)
	if jobID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "invalid cron.update params: missing id",
		}, nil)
		return nil
	}
	patch := cron.JobPatch{}
	if p, ok := opts.Params["patch"].(map[string]interface{}); ok {
		if v, ok := p["enabled"].(bool); ok {
			patch.Enabled = &v
		}
		if v, ok := p["name"].(string); ok {
			patch.Name = v
		}
		if v, ok := p["description"].(string); ok {
			desc := v
			patch.Description = &desc
		}
		if v, ok := p["agentId"].(string); ok {
			aid := v
			patch.AgentID = &aid
		}
		if s := parseSchedule(p["schedule"]); s.Kind != "" || s.At != "" || s.Expr != "" || s.EveryMs != 0 {
			if ok2, msg := validateScheduleTimestamp(s); !ok2 {
				opts.Respond(false, nil, &protocol.ErrorShape{
					Code:    protocol.ErrCodeInvalidRequest,
					Message: msg,
				}, nil)
				return nil
			}
			patch.Schedule = &s
		}
		if rawPayload, ok := p["payload"]; ok && rawPayload != nil {
			pl := parsePayload(rawPayload)
			if pl.Kind != "" {
				patch.Payload = &pl
			}
		}
		if v, ok := p["sessionTarget"].(string); ok && v != "" {
			st := v
			patch.SessionTarget = &st
		}
		if v, ok := p["wakeMode"].(string); ok && v != "" {
			wm := v
			patch.WakeMode = &wm
		}
		if d := parseDelivery(p["delivery"]); d != nil {
			patch.Delivery = d
		}
		if v, ok := p["sessionKey"].(string); ok {
			sk := strings.TrimSpace(v)
			patch.SessionKey = &sk
		}
		if v, ok := p["digitalEmployeeId"].(string); ok {
			de := strings.TrimSpace(v)
			patch.DigitalEmployeeID = &de
		} else if _, exists := p["digitalEmployeeId"]; exists && p["digitalEmployeeId"] == nil {
			// 显式 null 不修改；保持行为与其他字段一致（此处不做处理）
		}
	}
	j, err := svc.Update(jobID, patch)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, j, nil, nil)
	return nil
}

// CronRunHandler handles "cron.run". Accepts id or jobId; returns { ok, ran } like TS.
func CronRunHandler(opts HandlerOpts) error {
	ctx := opts.Context
	if ctx == nil || ctx.CronService == nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service not configured",
		}, nil)
		return nil
	}
	svc, ok := ctx.CronService.(interface {
		Run(string, string, string, string, string, string) error
	})
	if !ok {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: "cron service does not support run with context",
		}, nil)
		return nil
	}
	jobID := getJobID(opts.Params)
	if jobID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "invalid cron.run params: missing id",
		}, nil)
		return nil
	}
	mode := "force"
	if v, ok := opts.Params["mode"].(string); ok && (v == "due" || v == "force") {
		mode = v
	}
	domain, _ := opts.Params["domain"].(string)
	clusterID, _ := opts.Params["clusterId"].(string)
	component, _ := opts.Params["component"].(string)
	scenarioKey, _ := opts.Params["scenarioKey"].(string)
	err := svc.Run(jobID, mode, domain, clusterID, component, scenarioKey)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	opts.Respond(true, map[string]interface{}{"ok": true, "ran": true}, nil, nil)
	return nil
}

// cronRunLogEntry matches TS CronRunLogEntry (action=finished).
type cronRunLogEntry struct {
	Ts          int64                 `json:"ts"`
	JobID       string                `json:"jobId"`
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

func enrichInspectionCronRunLogEntry(entry *cronRunLogEntry, variant int) {
	if entry == nil || !strings.HasPrefix(entry.JobID, "job-inspect-") {
		return
	}
	if entry.Result == nil {
		runAtMs := entry.Ts
		if entry.RunAtMs != nil {
			runAtMs = *entry.RunAtMs
		}
		durationMs := int64(0)
		if entry.DurationMs != nil {
			durationMs = *entry.DurationMs
		}
		parsed := ops.ParseInspectionResult(entry.SessionID, entry.JobID, entry.Summary, entry.Status, runAtMs, durationMs)
		entry.Result = &parsed
	}
	runAtMs := entry.Ts
	if entry.RunAtMs != nil {
		runAtMs = *entry.RunAtMs
	}
	ops.EnrichInspectionCronRunResult(entry.Result, entry.JobID, runAtMs, variant)
	if entry.Result != nil && strings.TrimSpace(entry.Result.ReportMarkdown) != "" {
		entry.Summary = entry.Result.ReportMarkdown
	}
}

// resolveCronRunLogPath returns path to job run log file (same layout as TS run-log).
func resolveCronRunLogPath(storePath, jobID string) string {
	dir := filepath.Dir(storePath)
	return filepath.Join(dir, "runs", jobID+".jsonl")
}

// readCronRunLogEntries reads JSONL run log, newest first, up to limit (1–5000, default 200). Matches TS readCronRunLogEntries.
func readCronRunLogEntries(logPath string, jobID string, limit int) ([]cronRunLogEntry, error) {
	if limit < 1 {
		limit = 200
	}
	if limit > 5000 {
		limit = 5000
	}
	f, err := os.Open(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	var entries []cronRunLogEntry
	for i := len(lines) - 1; i >= 0 && len(entries) < limit; i-- {
		line := lines[i]
		if line == "" {
			continue
		}
		var raw map[string]interface{}
		if json.Unmarshal([]byte(line), &raw) != nil {
			continue
		}
		if raw["action"] != "finished" {
			continue
		}
		jid, _ := raw["jobId"].(string)
		if jid == "" {
			continue
		}
		if jobID != "" && jid != jobID {
			continue
		}
		ts, _ := raw["ts"].(float64)
		if ts == 0 {
			continue
		}
		e := cronRunLogEntry{Ts: int64(ts), JobID: jid, Action: "finished"}
		if v, ok := raw["status"].(string); ok {
			e.Status = v
		}
		if v, ok := raw["error"].(string); ok {
			e.Error = v
		}
		if v, ok := raw["summary"].(string); ok {
			e.Summary = v
		}
		if v, ok := raw["sessionId"].(string); ok {
			e.SessionID = v
		}
		if v, ok := raw["sessionKey"].(string); ok {
			e.SessionKey = v
		}
		if v, ok := raw["runAtMs"].(float64); ok {
			t := int64(v)
			e.RunAtMs = &t
		}
		if v, ok := raw["durationMs"].(float64); ok {
			t := int64(v)
			e.DurationMs = &t
		}
		if v, ok := raw["nextRunAtMs"].(float64); ok {
			t := int64(v)
			e.NextRunAtMs = &t
		}
		if v, ok := raw["domain"].(string); ok {
			e.Domain = v
		}
		if v, ok := raw["clusterId"].(string); ok {
			e.ClusterID = v
		}
		if v, ok := raw["component"].(string); ok {
			e.Component = v
		}
		if v, ok := raw["scenarioKey"].(string); ok {
			e.ScenarioKey = v
		}
		if rMap, ok := raw["result"].(map[string]interface{}); ok {
			var rObj ops.InspectionResult
			if rBytes, err := json.Marshal(rMap); err == nil {
				if json.Unmarshal(rBytes, &rObj) == nil {
					e.Result = &rObj
				}
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// CronRunsHandler handles "cron.runs". Accepts id or jobId and optional limit; returns { entries } from run log (same as TS).
func CronRunsHandler(opts HandlerOpts) error {
	jobID := getJobID(opts.Params)
	if jobID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "invalid cron.runs params: missing id",
		}, nil)
		return nil
	}
	limit := 200
	if v, ok := opts.Params["limit"].(float64); ok {
		limit = int(v)
	} else if v, ok := opts.Params["limit"].(int); ok {
		limit = v
	} else if v, ok := opts.Params["limit"].(string); ok {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	storePath := ""
	if opts.Context != nil && opts.Context.GetCronStorePath != nil {
		storePath = opts.Context.GetCronStorePath()
	}
	if storePath == "" {
		opts.Respond(true, map[string]interface{}{"entries": []interface{}{}}, nil, nil)
		return nil
	}
	logPath := resolveCronRunLogPath(storePath, jobID)
	entries, err := readCronRunLogEntries(logPath, jobID, limit)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}
	for i := range entries {
		enrichInspectionCronRunLogEntry(&entries[i], i)
	}
	// Convert to []interface{} for response
	out := make([]interface{}, len(entries))
	for i := range entries {
		out[i] = entries[i]
	}
	opts.Respond(true, map[string]interface{}{"entries": out}, nil, nil)
	return nil
}

func parsePayload(v interface{}) cron.CronPayload {
	if v == nil {
		return cron.CronPayload{}
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return cron.CronPayload{}
	}
	p := cron.CronPayload{}
	if k, ok := m["kind"].(string); ok {
		p.Kind = k
	}
	if t, ok := m["text"].(string); ok {
		p.Text = t
	}
	if msg, ok := m["message"].(string); ok {
		p.Message = msg
	}
	return p
}

// parseDelivery parses optional delivery config from params (cron.add / cron.update). Aligns with openclaw CronDelivery.
func parseDelivery(v interface{}) *cron.CronDelivery {
	if v == nil {
		return nil
	}
	m, ok := v.(map[string]interface{})
	if !ok {
		return nil
	}
	mode, _ := m["mode"].(string)
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode != "announce" && mode != "webhook" && mode != "none" {
		// legacy "deliver" -> announce
		if mode == "deliver" {
			mode = "announce"
		} else {
			mode = ""
		}
	}
	channel, _ := m["channel"].(string)
	channel = strings.TrimSpace(strings.ToLower(channel))
	if channel == "" && (mode == "announce" || mode == "") {
		channel = "last"
	}
	to, _ := m["to"].(string)
	to = strings.TrimSpace(to)
	bestEffort := false
	if b, ok := m["bestEffort"].(bool); ok {
		bestEffort = b
	}
	if mode == "" && channel == "last" && to == "" && !bestEffort {
		return nil
	}
	if mode == "" {
		mode = "announce"
	}
	return &cron.CronDelivery{
		Mode:       mode,
		Channel:    channel,
		To:         to,
		BestEffort: bestEffort,
	}
}
