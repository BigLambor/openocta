package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/openocta/openocta/pkg/cron"
	"github.com/openocta/openocta/pkg/ops"
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
	var isDegraded bool
	var scoreText string = "未知"
	var res ops.InspectionResult

	if isInspectionJob {
		res = ops.ParseInspectionResult("", jobID, summary, status, 0, 0)
		if res.Score != nil {
			scoreVal := *res.Score
			scoreText = fmt.Sprintf("%d", scoreVal)
			if scoreVal < 85 && res.ScoreStatus != "degraded" && res.ScoreStatus != "unknown" {
				isCritical = true
			}
		} 
		
		// If score is absent or degraded, it's not a normal low-score critical alert,
		// but rather a degraded exception.
		if res.ScoreStatus == "degraded" || res.ScoreStatus == "unknown" || len(res.MissingSources) > 0 {
			isDegraded = true
		} else if status != "ok" || len(res.Errors) > 0 {
			// Other fatal errors
			isCritical = true
		}

		if (isCritical || isDegraded) && (job.Delivery == nil || job.Delivery.Mode == "none" || job.Delivery.Mode == "") {
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
				header := "深度巡检告警 · " + job.Name
				var link string
				if domain := ops.DomainFromInspectJobID(jobID); domain != "" {
					link = ops.BuildUIDeepLink(domain + "?opsSubTab=inspections")
				}
				var alertMessage string
				
				if isDegraded {
					header = "巡检降级告警 · " + job.Name
					alertMessage = fmt.Sprintf("由于缺失核心数据源，巡检任务【%s】触发降级状态 (ScoreStatus: %s)。\n缺失来源: %s\n详情查看: %s", job.Name, res.ScoreStatus, strings.Join(res.MissingSources, ", "), link)
				} else if res.Score != nil {
					alertMessage = ops.FormatInspectionAlertCard(job.Name, *res.Score, summary, link)
				} else {
					alertMessage = fmt.Sprintf("巡检任务【%s】执行异常，未生成健康度得分。\n错误详情：%s\n查看报告：%s", job.Name, strings.Join(res.Errors, ", "), link)
				}
				params := map[string]interface{}{
					"channel": targetChannel,
					"to":      "last",
					"message": alertMessage,
					"header":  header,
				}
				_, _, _ = ctx.InvokeMethod("send", params)
			} else if isCritical || isDegraded {
				slog.Warn("inspection alert skipped: no IM channel enabled", "jobId", jobID, "score", scoreText, "isDegraded", isDegraded)
			}
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
