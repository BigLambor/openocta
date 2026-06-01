package ops

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/openocta/openocta/pkg/session"
)

// ToolRunReport structures the execution output of an individual tool during an inspection.
type ToolRunReport struct {
	ToolName string `json:"toolName"`
	Success  bool   `json:"success"`
	Output   string `json:"output,omitempty"`
	Error    string `json:"error,omitempty"`
}

// InspectionResult is the structured report representing a single inspection run.
type InspectionResult struct {
	ID              string                 `json:"id"`
	JobID           string                 `json:"jobId"`
	Domain          string                 `json:"domain"`
	ClusterID       string                 `json:"clusterId,omitempty"`
	Component       string                 `json:"component,omitempty"`
	Score           *int                   `json:"score,omitempty"`
	ScoreStatus     string                 `json:"scoreStatus"` // "ok", "warning", "critical", "unknown", "degraded"
	ToolRuns        []ToolRunReport        `json:"toolRuns,omitempty"`
	MetricsEvidence map[string]interface{} `json:"metricsEvidence,omitempty"`
	Errors          []string               `json:"errors,omitempty"`
	ReportMarkdown  string                 `json:"reportMarkdown,omitempty"`
	StartedAt       int64                  `json:"startedAt"`
	FinishedAt      int64                  `json:"finishedAt"`
}

// ParseInspectionResult extracts the score, status, tool runs, and errors for a completed session.
func ParseInspectionResult(sessionID string, jobID string, summary string, status string, runAtMs int64, durationMs int64) InspectionResult {
	res := InspectionResult{
		ID:         sessionID,
		JobID:      jobID,
		Domain:     DomainFromInspectJobID(jobID),
		StartedAt:  runAtMs,
		FinishedAt: runAtMs + durationMs,
	}

	// 1. Parse Score & ScoreStatus
	scoreMatch := regexp.MustCompile(`(?i)(?:健康得分|健康度|Score)\s*[：:]\s*(\d+)`).FindStringSubmatch(summary)
	if len(scoreMatch) > 1 {
		var s int
		if _, err := fmt.Sscanf(scoreMatch[1], "%d", &s); err == nil {
			res.Score = &s
			if s >= 90 {
				res.ScoreStatus = "ok"
			} else if s >= 75 {
				res.ScoreStatus = "warning"
			} else {
				res.ScoreStatus = "critical"
			}
		}
	}

	// 2. Parse ToolRuns from transcript
	transcriptPath := session.ResolveSessionFilePath(sessionID, nil, os.Getenv)
	if msgs, err := session.ReadTranscriptMessages(transcriptPath, 0); err == nil {
		toolRunsMap := make(map[string]*ToolRunReport)
		var toolOrder []string

		for _, m := range msgs {
			// Check Tool Call
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
			// Check Tool Result
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
						res.Errors = append(res.Errors, fmt.Sprintf("工具 %s 执行失败: %s", report.ToolName, resultText))
					}
				}
			}
		}

		for _, id := range toolOrder {
			if report, ok := toolRunsMap[id]; ok {
				res.ToolRuns = append(res.ToolRuns, *report)
			}
		}
	}

	// 3. Fallback for ScoreStatus and Errors if no score is generated
	if res.Score == nil {
		if len(res.Errors) > 0 || status == "error" {
			res.ScoreStatus = "degraded"
		} else {
			res.ScoreStatus = "unknown"
		}
	}

	if status == "error" {
		res.Errors = append(res.Errors, "巡检执行遇到系统错误")
	}

	res.ReportMarkdown = summary
	return res
}
