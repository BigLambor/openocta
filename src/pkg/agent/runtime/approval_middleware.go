package runtime

import (
	"context"
	"fmt"
	"strings"

	octasecurity "github.com/openocta/openocta/pkg/security"
	"github.com/stellarlinkco/agentsdk-go/pkg/middleware"
	"github.com/stellarlinkco/agentsdk-go/pkg/model"
)

func formatApprovalCommand(toolName, target string) string {
	name := strings.TrimSpace(toolName)
	if name == "" {
		name = "tool"
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return name
	}
	return fmt.Sprintf("%s(%s)", name, target)
}

func approvalTargetFromToolCall(call model.ToolCall) string {
	for _, key := range []string{"command", "path", "file_path", "target", "content"} {
		if v, ok := call.Arguments[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

// approvalQueueMiddleware blocks high-risk tool execution until the approval queue allows it.
func approvalQueueMiddleware(q *octasecurity.ApprovalQueue, blockWait bool) middleware.Middleware {
	return middleware.Funcs{
		Identifier: "openocta-approval-queue",
		OnBeforeTool: func(ctx context.Context, st *middleware.State) error {
			if q == nil || st == nil {
				return nil
			}
			call, ok := st.ToolCall.(model.ToolCall)
			if !ok {
				return nil
			}
			toolName := strings.TrimSpace(call.Name)
			if !octasecurity.ToolRequiresApproval(toolName) {
				return nil
			}
			target := approvalTargetFromToolCall(call)
			if target == "" && !strings.EqualFold(toolName, "write") && !strings.EqualFold(toolName, "edit") {
				return nil
			}
			sid, _ := st.Values["session_id"].(string)
			if strings.TrimSpace(sid) == "" {
				return fmt.Errorf("openocta: approval queue requires session_id")
			}
			line := formatApprovalCommand(toolName, target)
			rec, err := q.Request(sid, line, nil)
			if err != nil {
				return err
			}
			if rec.State == octasecurity.ApprovalApproved {
				return nil
			}
			if !blockWait {
				return fmt.Errorf("%s approval required (requestId=%s); approve via gateway", toolName, rec.ID)
			}
			resolved, err := q.Wait(ctx, rec.ID)
			if err != nil {
				return err
			}
			switch resolved.State {
			case octasecurity.ApprovalApproved:
				return nil
			case octasecurity.ApprovalDenied:
				reason := strings.TrimSpace(resolved.Reason)
				if reason == "" {
					reason = "denied"
				}
				return fmt.Errorf("%s execution denied: %s", toolName, reason)
			default:
				return fmt.Errorf("%s approval left pending", toolName)
			}
		},
	}
}
