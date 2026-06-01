package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/openocta/openocta/pkg/rbac"
)

type contextKey string

const UserSessionKey contextKey = "user_session"

// requirePermission restricts access to users with a specific permission.
// Admin role bypasses all permission checks.
func (s *Server) requirePermission(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS preflight requests bypass auth
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"未提供认证Token","code":"unauthorized"}`))
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"认证Token格式错误，必须为 Bearer <token>","code":"unauthorized"}`))
			return
		}

		token := parts[1]
		session, err := rbac.ValidateToken(token)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"登录会话无效或已过期，请重新登录","code":"unauthorized"}`))
			return
		}

		// Admin bypasses all checks
		if session.RoleName == "admin" {
			ctx := context.WithValue(r.Context(), UserSessionKey, session)
			next(w, r.WithContext(ctx))
			return
		}

		// Enforce permission checks if required
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
		// CORS preflight
		if r.Method == http.MethodOptions {
			next(w, r)
			return
		}

		// 1. Try RBAC auth first
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" && strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 {
				token := parts[1]
				session, err := rbac.ValidateToken(token)
				if err == nil {
					// Admin gets full pass
					if session.RoleName == "admin" {
						ctx := context.WithValue(r.Context(), UserSessionKey, session)
						next(w, r.WithContext(ctx))
						return
					}
					// Normal permission check
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
			}
		}

		// 2. Fallback to legacy Gateway Token
		if s.validateGatewayToken(r) {
			next(w, r)
			return
		}

		// Both failed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"未授权的请求，请提供有效的认证Token","code":"unauthorized"}`))
	}
}
