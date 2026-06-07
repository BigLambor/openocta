package ops

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/openocta/openocta/pkg/agent/tools"
	"github.com/openocta/openocta/pkg/session"
)

var (
	alertsMu      sync.RWMutex
	alertsPath    string
	alertGroups   []AlertGroup
	stateDirRoot  string
)

func loadAlertsLocked() error {
	if alertsPath == "" {
		return fmt.Errorf("ops alerts store 未初始化")
	}
	store, err := loadAlertsStore(alertsPath)
	if err != nil {
		return err
	}
	alertGroups = store.Groups
	return nil
}

func persistAlertsLocked() error {
	if alertsPath == "" {
		return fmt.Errorf("ops alerts store 未初始化")
	}
	return saveAlertsStore(alertsPath, &alertsStoreFile{Version: 1, Groups: alertGroups})
}

// InitAlertsStore loads alert groups from stateDir/ops/alerts.json.
func InitAlertsStore(stateDir string) error {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	stateDirRoot = stateDir
	alertsPath = filepath.Join(stateDir, "ops", "alerts.json")
	err := loadAlertsLocked()
	if err != nil {
		return err
	}

	if len(alertGroups) == 0 && flag.Lookup("test.v") == nil {
		now := nowMs()
		alertGroups = []AlertGroup{
			{
				ID:            "alert-group-hadoop-01",
				Source:        "prometheus",
				Domain:        DomainHadoop,
				Title:         "HDFS NameNode 内存占用过高",
				Severity:      "warning",
				Status:        AlertStatusActive,
				OriginalCount: 3,
				ReducedTo:     1,
				CreatedAtMs:   now - 3600000,
				UpdatedAtMs:   now - 3600000,
				Alertname:     "HDFSNameNodeHeapUsageHigh",
				Service:       "HDFS",
				Instance:      "nn-prod-01",
				ClusterID:     "cluster-bch-prod-a",
				Component:     "NameNode",
				Timeline: []AlertTimelineEvent{
					{Type: "created", Operator: "system", TimestampMs: now - 3600000, Message: "告警组已创建"},
				},
			},
			{
				ID:            "alert-group-gbase-01",
				Source:        "gbase-monitor",
				Domain:        DomainGBase,
				Title:         "GBase 数据库连接池占满告警",
				Severity:      "critical",
				Status:        AlertStatusActive,
				OriginalCount: 5,
				ReducedTo:     1,
				CreatedAtMs:   now - 1800000,
				UpdatedAtMs:   now - 1800000,
				Alertname:     "GBaseConnectionPoolExhausted",
				Service:       "GBase",
				Instance:      "gbase-prod-01",
				ClusterID:     "cluster-gbase-prod",
				Component:     "ConnectionPool",
				Timeline: []AlertTimelineEvent{
					{Type: "created", Operator: "system", TimestampMs: now - 1800000, Message: "告警组已创建"},
				},
			},
			{
				ID:            "alert-group-fi-01",
				Source:        "fi-manager",
				Domain:        DomainFI,
				Title:         "FusionInsight YARN root.default 队列资源耗尽",
				Severity:      "warning",
				Status:        AlertStatusActive,
				OriginalCount: 2,
				ReducedTo:     1,
				CreatedAtMs:   now - 900000,
				UpdatedAtMs:   now - 900000,
				Alertname:     "FIYarnQueueExhausted",
				Service:       "YARN",
				Instance:      "fi-yarn-rm-01",
				ClusterID:     "cluster-fi-prod",
				Component:     "ResourceManager",
				Timeline: []AlertTimelineEvent{
					{Type: "created", Operator: "system", TimestampMs: now - 900000, Message: "告警组已创建"},
				},
			},
			{
				ID:            "alert-group-gov-01",
				Source:        "governance-lineage",
				Domain:        DomainGovernance,
				Title:         "开发治理平台数据血缘解析中断异常",
				Severity:      "warning",
				Status:        AlertStatusActive,
				OriginalCount: 1,
				ReducedTo:     1,
				CreatedAtMs:   now - 600000,
				UpdatedAtMs:   now - 600000,
				Alertname:     "LineageParsingInterrupted",
				Service:       "Metadata",
				Instance:      "gov-lineage-01",
				ClusterID:     "cluster-gov-platform",
				Component:     "LineageEngine",
				Timeline: []AlertTimelineEvent{
					{Type: "created", Operator: "system", TimestampMs: now - 600000, Message: "告警组已创建"},
				},
			},
			{
				ID:            "alert-group-dataapps-01",
				Source:        "scheduler-dataapp",
				Domain:        DomainDataApps,
				Title:         "核心数据 App (financial_report_daily) 出现 SLA 逾期告警",
				Severity:      "critical",
				Status:        AlertStatusActive,
				OriginalCount: 1,
				ReducedTo:     1,
				CreatedAtMs:   now - 300000,
				UpdatedAtMs:   now - 300000,
				Alertname:     "DataAppSLABreach",
				Service:       "Scheduler",
				Instance:      "scheduler-prod-01",
				ClusterID:     "cluster-dataapp-scheduler",
				Component:     "SLA-Monitor",
				Timeline: []AlertTimelineEvent{
					{Type: "created", Operator: "system", TimestampMs: now - 300000, Message: "告警组已创建"},
				},
			},
		}
		_ = persistAlertsLocked()
	}
	return nil
}

// MergedAlertInput is one alert in a merged batch (from hooks).
type MergedAlertInput struct {
	AlertID   string
	Title     string
	Message   string
	Severity  string
	Alertname string
	Service   string
	Instance  string
	ClusterID string
	Component string
}

// RecordMergedAlertGroup persists a batch when sliding-window merge triggers analysis.
func RecordMergedAlertGroup(source, sessionKey, runID string, raw []MergedAlertInput) (AlertGroup, error) {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	now := nowMs()
	events := make([]AlertEvent, 0, len(raw))
	for _, item := range raw {
		events = append(events, AlertEvent{
			AlertID:    item.AlertID,
			Title:      item.Title,
			Message:    item.Message,
			Severity:   item.Severity,
			ReceivedAt: now,
			Alertname:  item.Alertname,
			Service:    item.Service,
			Instance:   item.Instance,
			ClusterID:  item.ClusterID,
			Component:  item.Component,
		})
	}
	if len(events) == 0 {
		return AlertGroup{}, fmt.Errorf("告警批次为空")
	}

	src := strings.TrimSpace(source)
	if src == "" {
		src = "default"
	}

	var gAlertname, gService, gInstance, gClusterID, gComponent string
	if len(events) > 0 {
		gAlertname = events[0].Alertname
		gService = events[0].Service
		gInstance = events[0].Instance
		gClusterID = events[0].ClusterID
		gComponent = events[0].Component
	}

	g := AlertGroup{
		ID:            "alert-group-" + uuid.New().String(),
		Source:        src,
		Domain:        inferDomainFromSource(src),
		Title:         pickGroupTitle(events),
		Severity:      pickGroupSeverity(events),
		Status:        AlertStatusAnalyzing,
		OriginalCount: len(events),
		ReducedTo:     1,
		SessionKey:    sessionKey,
		RunID:         runID,
		Events:        events,
		CreatedAtMs:   now,
		UpdatedAtMs:   now,
		Alertname:     gAlertname,
		Service:       gService,
		Instance:      gInstance,
		ClusterID:     gClusterID,
		Component:     gComponent,
		Timeline: []AlertTimelineEvent{
			{
				Type:        "created",
				Operator:    "system",
				TimestampMs: now,
				Message:     fmt.Sprintf("告警组已创建，包含来自 %s 的 %d 条原始告警事件", src, len(events)),
			},
		},
		DiagnosticStatus: "analyzing",
	}
	alertGroups = append([]AlertGroup{g}, alertGroups...)
	const maxGroups = 500
	if len(alertGroups) > maxGroups {
		alertGroups = alertGroups[:maxGroups]
	}
	if err := persistAlertsLocked(); err != nil {
		alertGroups = alertGroups[1:]
		return AlertGroup{}, err
	}
	return g, nil
}

// ListAlertGroups returns groups filtered by domain and status.
func ListAlertGroups(domain, status string) AlertGroupsListResponse {
	alertsMu.RLock()
	defer alertsMu.RUnlock()

	domain = strings.TrimSpace(strings.ToLower(domain))
	status = strings.TrimSpace(strings.ToLower(status))

	out := make([]AlertGroup, 0)
	var originalTotal int
	pendingActive := 0

	for _, g := range alertGroups {
		if domain != "" && g.Domain != domain {
			continue
		}
		if status != "" && g.Status != status {
			continue
		}
		enriched := enrichAlertGroupFromSession(g)
		out = append(out, enriched)
		originalTotal += g.OriginalCount
		if g.Status == AlertStatusActive || g.Status == AlertStatusAnalyzing {
			pendingActive++
		}
	}

	merged := len(out)
	var rate float64
	if originalTotal > 0 && merged > 0 {
		rate = (1 - float64(merged)/float64(originalTotal)) * 100
		if rate < 0 {
			rate = 0
		}
	}

	return AlertGroupsListResponse{
		Groups:        out,
		Total:         merged,
		OriginalTotal: originalTotal,
		MergedTotal:   merged,
		ReductionRate: rate,
		PendingActive: pendingActive,
	}
}

// CountPendingAlerts returns active + analyzing groups across all domains.
func CountPendingAlerts() int {
	alertsMu.RLock()
	defer alertsMu.RUnlock()

	n := 0
	for _, g := range alertGroups {
		if g.Status == AlertStatusActive || g.Status == AlertStatusAnalyzing {
			n++
		}
	}
	return n
}

// GetAlertGroup returns one group by ID with session analysis merged in.
func GetAlertGroup(id string) (AlertGroup, error) {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	id = strings.TrimSpace(id)
	for i, g := range alertGroups {
		if g.ID != id {
			continue
		}
		enriched := enrichAlertGroupFromSession(g)
		if enriched.RootCauseMarkdown != "" && enriched.Status == AlertStatusAnalyzing {
			enriched.Status = AlertStatusActive
			enriched.UpdatedAtMs = nowMs()
			alertGroups[i] = enriched
			_ = persistAlertsLocked()
		}
		return enriched, nil
	}
	return AlertGroup{}, fmt.Errorf("告警组不存在: %s", id)
}

// PatchAlertGroup updates group status (e.g. resolved).
func PatchAlertGroup(id string, patch AlertGroupPatch, operator string) (AlertGroup, error) {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	id = strings.TrimSpace(id)
	for i, g := range alertGroups {
		if g.ID != id {
			continue
		}

		now := nowMs()
		if operator == "" {
			operator = "system"
		}

		if patch.Status != nil {
			st := strings.TrimSpace(strings.ToLower(*patch.Status))
			switch st {
			case AlertStatusActive, AlertStatusAnalyzing, AlertStatusResolved:
				if st == AlertStatusResolved {
					var note, reason string
					if patch.AckNote != nil {
						note = strings.TrimSpace(*patch.AckNote)
					}
					if patch.ResolvedReason != nil {
						reason = strings.TrimSpace(*patch.ResolvedReason)
					}
					if note == "" && reason == "" {
						return AlertGroup{}, fmt.Errorf("确认或关闭告警时，必须填写处理备注或关闭原因")
					}
				}

				if g.Status != st {
					g.Status = st
					g.Timeline = append(g.Timeline, AlertTimelineEvent{
						Type:        "status_change",
						Operator:    operator,
						TimestampMs: now,
						Message:     fmt.Sprintf("告警组状态变更为 [%s]", st),
					})
				}
			default:
				return AlertGroup{}, fmt.Errorf("无效的状态: %s", st)
			}
		}

		if patch.Assignee != nil {
			val := strings.TrimSpace(*patch.Assignee)
			if g.Assignee != val {
				g.Assignee = val
				g.Timeline = append(g.Timeline, AlertTimelineEvent{
					Type:        "assignee_change",
					Operator:    operator,
					TimestampMs: now,
					Message:     fmt.Sprintf("指派负责人为: %s", val),
				})
			}
		}

		if patch.AckNote != nil {
			val := strings.TrimSpace(*patch.AckNote)
			if val != "" {
				g.AckNote = val
				g.Timeline = append(g.Timeline, AlertTimelineEvent{
					Type:        "ack_note",
					Operator:    operator,
					TimestampMs: now,
					Message:     fmt.Sprintf("添加确认备注: %s", val),
				})
			}
		}

		if patch.ResolvedReason != nil {
			val := strings.TrimSpace(*patch.ResolvedReason)
			if val != "" {
				g.ResolvedReason = val
				g.Timeline = append(g.Timeline, AlertTimelineEvent{
					Type:        "resolved_reason",
					Operator:    operator,
					TimestampMs: now,
					Message:     fmt.Sprintf("添加解决原因: %s", val),
				})
			}
		}

		g.UpdatedAtMs = now
		alertGroups[i] = g
		if err := persistAlertsLocked(); err != nil {
			return AlertGroup{}, err
		}
		return enrichAlertGroupFromSession(g), nil
	}
	return AlertGroup{}, fmt.Errorf("告警组不存在: %s", id)
}

func enrichAlertGroupFromSession(g AlertGroup) AlertGroup {
	if strings.TrimSpace(g.SessionKey) == "" {
		return g
	}

	sessionID := tools.SessionIDFromSessionKey(g.SessionKey)
	md := readAssistantMarkdown(g.SessionKey)

	if md != "" {
		md = parseReportMarkdownFromText(md)
		md = strings.ReplaceAll(md, "\\n", "\n")
		md = strings.ReplaceAll(md, "\\t", "\t")

		g.RootCauseMarkdown = md
		g.DiagnosticStatus = "completed"

		if g.ImpactMarkdown == "" {
			g.ImpactMarkdown = extractSection(md, []string{"## 影响", "## impact", "### 影响", "业务受损", "impact assessment", "影响范围", "影响面"})
		}
		g.ImpactAnalysis = g.ImpactMarkdown

		if g.RootCauseSummary == "" {
			g.RootCauseSummary = extractSection(md, []string{"## 根因", "## root cause", "### 根因", "根因分析", "原因分析", "## 根因分析", "判断结论", "根因候选", "结论", "根因锁定"})
			if g.RootCauseSummary == "" {
				lines := strings.Split(md, "\n")
				for _, line := range lines {
					trimmed := strings.TrimSpace(line)
					if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
						if len(trimmed) > 200 {
							g.RootCauseSummary = trimmed[:200] + "..."
						} else {
							g.RootCauseSummary = trimmed
						}
						break
					}
				}
			}
		}

		if g.SuggestedActions == "" {
			g.SuggestedActions = extractSection(md, []string{"## 建议动作", "## suggested actions", "### 建议动作", "## 建议", "## 处置建议", "## 处置步骤", "## 排查步骤", "## 修复步骤", "remediation"})
		}
	} else {
		if g.DiagnosticStatus == "" || g.DiagnosticStatus == "completed" {
			g.DiagnosticStatus = "analyzing"
		}
	}

	if g.Evidence == nil || len(g.Evidence) == 0 {
		toolRuns := parseAlertGroupToolRuns(sessionID)
		if len(toolRuns) > 0 {
			evidenceMap := make(map[string]interface{})
			evidenceMap["toolRuns"] = toolRuns
			g.Evidence = evidenceMap
		}
	}

	return g
}

func parseReportMarkdownFromText(text string) string {
	startMarker := "```json"
	endMarker := "```"
	startIdx := strings.Index(text, startMarker)
	if startIdx >= 0 {
		rest := text[startIdx+len(startMarker):]
		endIdx := strings.Index(rest, endMarker)
		if endIdx >= 0 {
			jsonStr := strings.TrimSpace(rest[:endIdx])
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
				if reportMD, ok := parsed["reportMarkdown"].(string); ok && strings.TrimSpace(reportMD) != "" {
					return strings.TrimSpace(reportMD)
				}
				if reportMD, ok := parsed["report_markdown"].(string); ok && strings.TrimSpace(reportMD) != "" {
					return strings.TrimSpace(reportMD)
				}
			}
		}
	}

	startMarker = "```"
	startIdx = strings.Index(text, startMarker)
	if startIdx >= 0 {
		rest := text[startIdx+len(startMarker):]
		endIdx := strings.Index(rest, endMarker)
		if endIdx >= 0 {
			jsonStr := strings.TrimSpace(rest[:endIdx])
			if strings.HasPrefix(jsonStr, "{") && strings.HasSuffix(jsonStr, "}") {
				var parsed map[string]interface{}
				if err := json.Unmarshal([]byte(jsonStr), &parsed); err == nil {
					if reportMD, ok := parsed["reportMarkdown"].(string); ok && strings.TrimSpace(reportMD) != "" {
						return strings.TrimSpace(reportMD)
					}
					if reportMD, ok := parsed["report_markdown"].(string); ok && strings.TrimSpace(reportMD) != "" {
						return strings.TrimSpace(reportMD)
					}
				}
			}
		}
	}
	return text
}

func parseAlertGroupToolRuns(sessionID string) []ToolRunReport {
	var runs []ToolRunReport
	env := func(k string) string { return os.Getenv(k) }
	transcriptPath := session.ResolveSessionFilePath(sessionID, &session.SessionPathOptions{AgentID: "main"}, env)
	msgs, err := session.ReadTranscriptMessages(transcriptPath, 0)
	if err != nil {
		return nil
	}
	toolRunsMap := make(map[string]*ToolRunReport)
	var toolOrder []string

	for _, m := range msgs {
		for _, block := range m.Content {
			if strings.EqualFold(block.Type, "toolCall") || strings.EqualFold(block.Type, "tool_file") || strings.EqualFold(block.Type, "tool_use") {
				id := block.ID
				name := block.Name
				if id != "" && name != "" {
					toolRunsMap[id] = &ToolRunReport{
						ToolName: name,
					}
					toolOrder = append(toolOrder, id)
				}
			}
		}
		if strings.EqualFold(m.Role, "toolResult") || strings.EqualFold(m.Role, "tool") {
			id := m.ToolCallID
			if report, ok := toolRunsMap[id]; ok {
				var resultText string
				for _, block := range m.Content {
					if strings.EqualFold(block.Type, "text") {
						resultText = block.Text
						break
					}
				}
				report.Success = !m.IsError
				if report.Success {
					report.Output = resultText
				} else {
					report.Error = resultText
				}
			}
		}
	}

	for _, id := range toolOrder {
		if report, ok := toolRunsMap[id]; ok {
			runs = append(runs, *report)
		}
	}
	return runs
}

func extractSection(md string, headings []string) string {
	lower := strings.ToLower(md)
	for _, heading := range headings {
		idx := strings.Index(lower, strings.ToLower(heading))
		if idx < 0 {
			continue
		}
		rest := md[idx+len(heading):]

		// Determine if the heading is a markdown header line (e.g., starts with #).
		// We look back from idx to find the start of the line.
		lineStart := 0
		for i := idx; i >= 0; i-- {
			if md[i] == '\n' {
				lineStart = i + 1
				break
			}
		}
		linePrefix := strings.TrimSpace(md[lineStart:idx])
		isHeaderLine := strings.HasPrefix(linePrefix, "#")

		if isHeaderLine {
			if nl := strings.Index(rest, "\n"); nl >= 0 {
				rest = rest[nl+1:]
			} else {
				rest = ""
			}
		} else {
			rest = strings.TrimLeft(rest, " ：:*`）)")
		}

		nextHeadingIdx := -1
		for _, nextMarker := range []string{"\n#", "\n##", "\n###"} {
			if pos := strings.Index(rest, nextMarker); pos >= 0 {
				if nextHeadingIdx == -1 || pos < nextHeadingIdx {
					nextHeadingIdx = pos
				}
			}
		}
		if nextHeadingIdx > 0 {
			return strings.TrimSpace(rest[:nextHeadingIdx])
		}
		return strings.TrimSpace(rest)
	}
	return ""
}

func readAssistantMarkdown(sessionKey string) string {
	if stateDirRoot == "" {
		return ""
	}
	sessionID := tools.SessionIDFromSessionKey(sessionKey)
	env := func(k string) string { return os.Getenv(k) }
	transcript := session.ResolveSessionFilePath(sessionID, &session.SessionPathOptions{AgentID: "main"}, env)
	if transcript == "" {
		return ""
	}
	items := session.ReadSessionPreviewItems(transcript, 20, 8000)
	var lastAssistant string
	for _, it := range items {
		if strings.EqualFold(it.Role, "assistant") && strings.TrimSpace(it.Text) != "" {
			lastAssistant = it.Text
		}
	}
	return strings.TrimSpace(lastAssistant)
}

// UpdateAlertGroupSessionKey updates the session key of a recorded alert group.
func UpdateAlertGroupSessionKey(id string, sessionKey string) error {
	alertsMu.Lock()
	defer alertsMu.Unlock()

	id = strings.TrimSpace(id)
	for i, g := range alertGroups {
		if g.ID == id {
			alertGroups[i].SessionKey = sessionKey
			alertGroups[i].UpdatedAtMs = nowMs()
			return persistAlertsLocked()
		}
	}
	return fmt.Errorf("告警组不存在: %s", id)
}
