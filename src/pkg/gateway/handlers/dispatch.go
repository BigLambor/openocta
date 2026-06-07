package handlers

import (
	"encoding/json"

	"github.com/openocta/openocta/pkg/gateway/protocol"
)

// Dispatch dispatches a request to the appropriate handler.
func (r Registry) Dispatch(opts HandlerOpts) error {
	desc, exists := r[opts.Req.Method]
	if !exists {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeMethodNotFound,
			Message: "method not found: " + opts.Req.Method,
		}, nil)
		return nil
	}

	// Permission checks (only for external client calls, internal calls bypass)
	if desc.RequiredPermission != "" && opts.Client != nil {
		allowed := false
		if opts.Client.Session == nil {
			// Legacy token auth bypass (e.g. gateway token client)
			// NOTE: This is a transitional compatibility design for backward compatibility.
			// It should be deprecated in future versions as part of migrating all clients to use session-based RBAC.
			allowed = true
		} else {
			session := opts.Client.Session
			if session.RoleName == "admin" {
				allowed = true
			} else {
				for _, p := range session.Permissions {
					if p == desc.RequiredPermission {
						allowed = true
						break
					}
				}
			}
		}

		if !allowed {
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    protocol.ErrCodeForbidden,
				Message: "forbidden: requires permission " + desc.RequiredPermission,
			}, nil)
			return nil
		}
	}

	opts.Params = unwrapParams(opts.Req)
	return desc.Handler(opts)
}

func unwrapParams(req protocol.RequestFrame) map[string]interface{} {
	if req.Params == nil {
		return nil
	}
	if m, ok := req.Params.(map[string]interface{}); ok {
		return m
	}
	b, err := json.Marshal(req.Params)
	if err != nil {
		return nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}
