package http

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/openocta/openocta/pkg/channels"
	"github.com/openocta/openocta/pkg/employees"
	"github.com/openocta/openocta/pkg/gateway/handlers"
	"github.com/openocta/openocta/pkg/ops"
)

// hooksAgentSink 将 RuntimeChannel 的 InboundMessage 转换为 HooksAgent 调用。
type hooksAgentSink struct {
	ctx *handlers.Context
}

func normalizeEmployeeID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	lower := strings.ToLower(raw)
	// 兼容 /api/v1/employees 返回的本地前缀：local:<id>
	if strings.HasPrefix(lower, "local:") {
		return strings.TrimSpace(raw[len("local:"):])
	}
	return raw
}

func (s *hooksAgentSink) Deliver(_ context.Context, msg *channels.InboundMessage) error {
	if s == nil || s.ctx == nil || s.ctx.HooksAgent == nil || msg == nil {
		return nil
	}
	text := strings.TrimSpace(msg.Content)
	if text == "" {
		return nil
	}

	channelID := strings.TrimSpace(msg.Channel)
	if channelID == "" {
		channelID = "feishu"
	}

	to := msg.ChatID
	if to == "" {
		to = msg.SenderID
	}

	if cmd, args, ok := ops.ParseChatOpsCommand(text); ok {
		result := ops.HandleChatOpsCommand(cmd, args)
		if result.Handled {
			s.replyIM(channelID, to, result.Reply)
			return nil
		}
		if result.AgentMessage != "" {
			text = result.AgentMessage
		}
	}

	// 默认 sessionKey：agent:main:channel:<channelId>:<peerId>
	// 当该渠道配置了 digitalEmployeeId 且员工未被删除时，使用数字员工会话：
	// agent:main:employee:<employeeId>
	// 这样会复用员工的 prompt/skills/mcp，并将所有入口统一进该员工的会话（不再按渠道/peer 拆分 sessionKey）。
	sessionKey := fmt.Sprintf("agent:main:channel:%s:%s", channelID, to)
	if s.ctx != nil && s.ctx.Config != nil && s.ctx.Config.Channels != nil {
		if chCfg := s.ctx.Config.Channels.GetChannelConfig(channelID); chCfg != nil {
			if rawEmp, ok := chCfg["digitalEmployeeId"]; ok {
				empID, _ := rawEmp.(string)
				empID = normalizeEmployeeID(empID)
				if empID != "" {
					// 检查员工是否存在（被删除则回退到普通会话 key，等同“走之前逻辑新建会话”）
					if _, err := employees.LoadManifest(empID, os.Getenv); err == nil {
						sessionKey = fmt.Sprintf("agent:main:employee:%s", empID)
					}
				}
			}
		}
	}

	params := handlers.HooksAgentParams{
		Message:    text,
		MessageID:  msg.ID,
		Name:       channelID,
		WakeMode:   "now",
		SessionKey: sessionKey,
		Deliver:    true,
		Channel:    channelID,
		To:         to,
		ChatType:   strings.TrimSpace(msg.ChatType),
	}
	_ = s.ctx.HooksAgent(params)
	return nil
}

func (s *hooksAgentSink) replyIM(channelID, to, message string) {
	if s == nil || s.ctx == nil || s.ctx.InvokeMethod == nil || strings.TrimSpace(message) == "" {
		return
	}
	if strings.TrimSpace(to) == "" {
		return
	}
	_, _, _ = s.ctx.InvokeMethod("send", map[string]interface{}{
		"channel": channelID,
		"to":      to,
		"message": message,
	})
}
