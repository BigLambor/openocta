package ops

import (
	"fmt"
	"strings"
)

// ChatOpsResult is the outcome of an inbound IM slash command.
type ChatOpsResult struct {
	Handled      bool
	Reply        string
	AgentMessage string // non-empty → continue to agent with this text
}

// ParseChatOpsCommand parses `/cmd args` from IM text.
func ParseChatOpsCommand(text string) (cmd, args string, ok bool) {
	text = strings.TrimSpace(text)
	if text == "" || !strings.HasPrefix(text, "/") {
		return "", "", false
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return "", "", false
	}
	cmd = strings.ToLower(strings.TrimPrefix(fields[0], "/"))
	if len(fields) > 1 {
		args = strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return cmd, args, true
}

// HandleChatOpsCommand executes supported ChatOps commands (P2-C2).
func HandleChatOpsCommand(cmd, args string) ChatOpsResult {
	switch cmd {
	case "help", "h", "?":
		return ChatOpsResult{
			Handled: true,
			Reply: strings.TrimSpace(`OpenOcta ChatOps 指令：
/help — 显示本帮助
/ack <告警组ID> — 将告警组标记为已处理
/diagnose [问题] — 发起运维智能诊断（将转交 Agent）`),
		}
	case "ack":
		id := strings.TrimSpace(args)
		if id == "" {
			return ChatOpsResult{
				Handled: true,
				Reply: "用法：/ack <告警组ID>（可在 Web 告警降噪页复制 ID，形如 alert-group-…）",
			}
		}
		resolved := AlertStatusResolved
		note := "通过 ChatOps /ack 指令确认"
		if _, err := PatchAlertGroup(id, AlertGroupPatch{
			Status:  &resolved,
			AckNote: &note,
		}, "ChatOps"); err != nil {
			return ChatOpsResult{Handled: true, Reply: "确认失败：" + err.Error()}
		}
		link := AlertGroupDeepLink("", id)
		if link == "" {
			if g, err := GetAlertGroup(id); err == nil {
				link = AlertGroupDeepLink(g.Domain, id)
			}
		}
		reply := fmt.Sprintf("已确认告警组 %s 为已处理。", id)
		if link != "" {
			reply += "\n查看：" + link
		}
		return ChatOpsResult{Handled: true, Reply: reply}
	case "diagnose":
		prompt := strings.TrimSpace(args)
		if prompt == "" {
			prompt = "请根据当前运维上下文，对最近相关告警与监控指标进行快速诊断，给出根因假设与排查步骤。"
		}
		msg := fmt.Sprintf(`[ChatOps /diagnose]
你是 OpenOcta 运维助手。用户通过 IM 请求诊断：
%s

请主动调用 query_vm_metrics、query_hadoop_jmx、query_fi_manager_metrics 等可用工具收集证据，用简体中文 Markdown 回复。`, prompt)
		return ChatOpsResult{Handled: false, AgentMessage: msg}
	default:
		return ChatOpsResult{
			Handled: true,
			Reply:   "未知指令，发送 /help 查看可用命令。",
		}
	}
}
