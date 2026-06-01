package ops

import "fmt"

// FormatInspectionAlertCard is IM card body for low-score inspection (P2-C1).
func FormatInspectionAlertCard(jobName string, score int, summary, deepLink string) string {
	body := fmt.Sprintf("健康得分：%d 分\n任务：%s\n\n%s", score, jobName, truncateRunes(summary, 400))
	if deepLink != "" {
		body += "\n\n查看巡检报告：\n" + deepLink
	}
	return body
}

// FormatAlertQueuedCard is IM card body when alert batch is queued for analysis.
func FormatAlertQueuedCard(g AlertGroup, deepLink string) string {
	body := fmt.Sprintf("已合并 %d 条告警\n来源：%s\n标题：%s", g.OriginalCount, g.Source, g.Title)
	if deepLink != "" {
		body += "\n\n打开告警详情：\n" + deepLink
	}
	return body
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}
