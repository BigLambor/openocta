package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/openocta/openocta/pkg/cron"
)

// cronJobIDFromSessionKey extracts job ID from cron session key "agent:<agentId>:cron:<jobId>".
func cronJobIDFromSessionKey(sessionKey string) string {
	rawKey := strings.TrimSpace(sessionKey)
	parts := strings.Split(strings.ToLower(rawKey), ":")
	if len(parts) >= 4 && parts[0] == "agent" && parts[2] == "cron" {
		rawParts := strings.Split(rawKey, ":")
		if len(rawParts) >= 4 {
			return rawParts[3]
		}
	}
	return ""
}

// DeliverCronResultIfNeeded runs after a cron session completes (chat.send for agent:main:cron:<jobId>).
// If the job has delivery mode "announce" or "webhook", it sends the summary to the configured channel or webhook.
func DeliverCronResultIfNeeded(ctx *Context, sessionKey, summary, status string) {
	if ctx == nil || strings.TrimSpace(summary) == "" {
		return
	}
	jobID := cronJobIDFromSessionKey(sessionKey)
	if jobID == "" {
		return
	}
	// Get job: use List and find by ID (CronService may not expose GetJob in interface).
	list, err := ctx.CronService.List(true)
	if err != nil {
		return
	}
	var job *cron.CronJob
	for i := range list {
		if list[i].ID == jobID {
			job = &list[i]
			break
		}
	}
	if job == nil {
		return
	}

	// Fallback/Automatic routing for inspection jobs
	isInspectionJob := strings.HasPrefix(jobID, "job-inspect-")
	var isCritical bool
	var score int = 100
	if isInspectionJob {
		// Parse score
		scoreMatch := regexp.MustCompile(`(?i)(?:健康得分|健康度|Score)\s*[：:]\s*(\d+)`).FindStringSubmatch(summary)
		if len(scoreMatch) > 1 {
			_, _ = fmt.Sscanf(scoreMatch[1], "%d", &score)
		}
		if score < 85 || strings.Contains(strings.ToUpper(summary), "CRITICAL") || strings.Contains(strings.ToUpper(summary), "ERROR") || status != "ok" {
			isCritical = true
		}
	}

	if isCritical && (job.Delivery == nil || job.Delivery.Mode == "none" || job.Delivery.Mode == "") {
		// Try to find an enabled channel to send the alert
		var targetChannel string
		if ctx.Config != nil && ctx.Config.Channels != nil {
			if f := ctx.Config.Channels.GetChannelConfig("feishu"); f != nil {
				if enabled, _ := f["enabled"].(bool); enabled {
					targetChannel = "feishu"
				}
			}
			if targetChannel == "" {
				if d := ctx.Config.Channels.GetChannelConfig("dingtalk"); d != nil {
					if enabled, _ := d["enabled"].(bool); enabled {
						targetChannel = "dingtalk"
					}
				}
			}
		}
		if targetChannel != "" && ctx.InvokeMethod != nil {
			header := "⚠️ 深度巡检告警: " + job.Name
			alertMessage := fmt.Sprintf("### ⚠️ %s 巡检异常报警\n- **健康得分**：%d 分\n- **异常诊断**：检测到潜在的核心指标异常，请立即登录系统处理！\n\n%s", job.Name, score, summary)
			if len(alertMessage) > 1500 {
				alertMessage = alertMessage[:1497] + "..."
			}
			params := map[string]interface{}{
				"channel": targetChannel,
				"to":      "last", // fallback to last active group/user chat
				"message": alertMessage,
				"header":  header,
			}
			_, _, _ = ctx.InvokeMethod("send", params)
		}
	}

	if job.Delivery == nil {
		return
	}

	d := job.Delivery
	mode := strings.TrimSpace(strings.ToLower(d.Mode))
	if mode != "announce" && mode != "webhook" {
		return
	}
	if mode == "announce" {
		channel := strings.TrimSpace(strings.ToLower(d.Channel))
		if channel == "" {
			channel = "last"
		}
		to := strings.TrimSpace(d.To)
		if to == "" && channel == "last" {
			return // cannot resolve "last" without to
		}
		if ctx.InvokeMethod == nil {
			return
		}
		header := "定时任务: " + job.Name
		if len(header) > 50 {
			header = header[:47] + "......"
		}
		params := map[string]interface{}{
			"channel": channel,
			"to":      to,
			"message": summary,
			"header":  header,
		}
		_, _, _ = ctx.InvokeMethod("send", params)
		return
	}
	// webhook
	url := strings.TrimSpace(d.To)
	if url == "" {
		return
	}
	if !strings.HasPrefix(strings.ToLower(url), "http://") && !strings.HasPrefix(strings.ToLower(url), "https://") {
		return
	}
	body := map[string]interface{}{
		"jobId":      jobID,
		"sessionKey": sessionKey,
		"summary":    summary,
		"status":     status,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	_ = resp.Body.Close()
}
