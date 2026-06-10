package handlers

import (
	"encoding/json"
	"strings"

	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/rbac"
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
	if opts.Client != nil {
		legacyGateway := opts.Client.Session == nil
		if err := rbac.AuthorizeMethod(opts.Client.Session, opts.Req.Method, legacyGateway); err != nil {
			msg := err.Error()
			code := protocol.ErrCodeForbidden
			if strings.Contains(msg, "unauthorized") {
				code = protocol.ErrCodeUnauthorized
			}
			opts.Respond(false, nil, &protocol.ErrorShape{
				Code:    code,
				Message: msg,
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
