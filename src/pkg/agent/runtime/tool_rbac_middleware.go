package runtime

import (
	"context"

	"github.com/openocta/openocta/pkg/rbac"
	"github.com/stellarlinkco/agentsdk-go/pkg/middleware"
	"github.com/stellarlinkco/agentsdk-go/pkg/model"
)

func toolRBACMiddleware(session *rbac.UserSession) middleware.Middleware {
	return middleware.Funcs{
		Identifier: "openocta-tool-rbac",
		OnBeforeTool: func(_ context.Context, st *middleware.State) error {
			call, ok := st.ToolCall.(model.ToolCall)
			if !ok {
				return nil
			}
			return rbac.CheckToolExecution(session, call.Name)
		},
	}
}
