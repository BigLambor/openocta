package http

import (
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/rbac"
)

const sessionCookieName = rbac.SessionCookieName

const sessionCookieMaxAge = 24 * 60 * 60

func readRBACSessionToken(r *http.Request) string {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		if token := strings.TrimSpace(c.Value); token != "" {
			return token
		}
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func validateRBACSession(r *http.Request) (*rbac.UserSession, error) {
	token := readRBACSessionToken(r)
	if token == "" {
		return nil, rbac.ErrNoSessionToken
	}
	return rbac.ValidateToken(token)
}

func setSessionCookie(w http.ResponseWriter, r *http.Request, token string) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   sessionCookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if isSecureRequest(r) {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)
}

func clearSessionCookie(w http.ResponseWriter, r *http.Request) {
	cookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
	if isSecureRequest(r) {
		cookie.Secure = true
	}
	http.SetCookie(w, cookie)
}

func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	if strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return true
	}
	return false
}

func writeSessionResponse(w http.ResponseWriter, r *http.Request, token string, session *rbac.UserSession) {
	setSessionCookie(w, r, token)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"user": session,
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := strings.TrimSpace(r.Header.Get("X-Real-IP")); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func writeLoginLocked(w http.ResponseWriter, status rbac.LoginLockStatus) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Retry-After", retryAfterSeconds(status.LockedUntil))
	w.WriteHeader(http.StatusTooManyRequests)
	payload := map[string]interface{}{
		"error": status.Reason,
		"code":  "login_locked",
	}
	if !status.LockedUntil.IsZero() {
		payload["lockedUntil"] = status.LockedUntil.UTC().Format(time.RFC3339)
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func retryAfterSeconds(until time.Time) string {
	sec := int(time.Until(until).Seconds())
	if sec < 1 {
		return "1"
	}
	return strconv.Itoa(sec)
}
