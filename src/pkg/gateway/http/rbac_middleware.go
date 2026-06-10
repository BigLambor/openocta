package http

import (
	"context"
	"net/http"

	"github.com/openocta/openocta/pkg/rbac"
)

type contextKey string

const UserSessionKey contextKey = "user_session"

func authenticateRequest(r *http.Request) (*rbac.UserSession, error) {
	return validateRBACSession(r)
}

// requirePermission restricts access to users with a specific permission.
// Admin role bypasses all permission checks.
func (s *Server) requirePermission(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		session, err := authenticateRequest(r)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"未登录或会话已过期，请重新登录","code":"unauthorized"}`))
			return
		}

		if session.RoleName == "admin" {
			ctx := context.WithValue(r.Context(), UserSessionKey, session)
			next(w, r.WithContext(ctx))
			return
		}

		if permission != "" {
			hasPermission := false
			for _, p := range session.Permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}
			if !hasPermission {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error":"您的账号无权访问此模块","code":"forbidden"}`))
				return
			}
		}

		ctx := context.WithValue(r.Context(), UserSessionKey, session)
		next(w, r.WithContext(ctx))
	}
}

// GetUserSession retrieves the validated user session from context.
func GetUserSession(r *http.Request) *rbac.UserSession {
	if val := r.Context().Value(UserSessionKey); val != nil {
		if session, ok := val.(*rbac.UserSession); ok {
			return session
		}
	}
	return nil
}

// requireRbacOrGatewayToken allows access if either a valid RBAC session (with permission) is present,
// or the legacy Gateway Token is verified.
func (s *Server) requireRbacOrGatewayToken(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		if session, err := authenticateRequest(r); err == nil {
			if session.RoleName == "admin" {
				ctx := context.WithValue(r.Context(), UserSessionKey, session)
				next(w, r.WithContext(ctx))
				return
			}
			if permission != "" {
				hasPermission := false
				for _, p := range session.Permissions {
					if p == permission {
						hasPermission = true
						break
					}
				}
				if !hasPermission {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"您的账号无权访问此模块","code":"forbidden"}`))
					return
				}
			}
			ctx := context.WithValue(r.Context(), UserSessionKey, session)
			next(w, r.WithContext(ctx))
			return
		}

		if s.validateGatewayToken(r) {
			next(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"未授权的请求，请登录或提供有效的 Gateway Token","code":"unauthorized"}`))
	}
}
