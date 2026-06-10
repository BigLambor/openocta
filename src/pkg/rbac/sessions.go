package rbac

import (
	"fmt"
	"sync"
	"time"

	"github.com/openocta/openocta/pkg/audit"
)

const sessionJanitorInterval = time.Hour

var sessionJanitorOnce sync.Once

// SessionInfo is a safe summary of an active login session.
type SessionInfo struct {
	TokenHint string    `json:"tokenHint"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	Current   bool      `json:"current"`
}

// StartSessionJanitor periodically removes expired login sessions.
func StartSessionJanitor() {
	sessionJanitorOnce.Do(func() {
		if n, err := CleanupExpiredSessions(); err == nil && n > 0 {
			_ = audit.Record(audit.Entry{
				Action:     "auth.session_cleanup",
				ObjectType: "session",
				Summary:    fmt.Sprintf("启动时清理 %d 个过期会话", n),
				Metadata: map[string]interface{}{
					"removed": n,
				},
			})
		}
		go func() {
			ticker := time.NewTicker(sessionJanitorInterval)
			defer ticker.Stop()
			for range ticker.C {
				if n, err := CleanupExpiredSessions(); err == nil && n > 0 {
					_ = audit.Record(audit.Entry{
						Action:     "auth.session_cleanup",
						ObjectType: "session",
						Summary:    fmt.Sprintf("清理 %d 个过期会话", n),
						Metadata: map[string]interface{}{
							"removed": n,
						},
					})
				}
			}
		}()
	})
}

// ListUserSessions returns active sessions for a user.
func ListUserSessions(userID int, currentToken string) ([]SessionInfo, error) {
	tokens, err := requireTokenRepo()
	if err != nil {
		return nil, err
	}
	records, err := tokens.ListByUserID(userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	out := make([]SessionInfo, 0, len(records))
	for _, rec := range records {
		if now.After(rec.ExpiresAt) {
			continue
		}
		out = append(out, SessionInfo{
			TokenHint: tokenHint(rec.Token),
			ExpiresAt: rec.ExpiresAt,
			CreatedAt: rec.CreatedAt,
			Current:   rec.Token != "" && rec.Token == currentToken,
		})
	}
	return out, nil
}

// InvalidateAllSessions revokes all sessions for a user, optionally keeping one token.
func InvalidateAllSessions(userID int, exceptToken string) (int, error) {
	tokens, err := requireTokenRepo()
	if err != nil {
		return 0, err
	}
	records, err := tokens.ListByUserID(userID)
	if err != nil {
		return 0, err
	}
	now := time.Now()
	var removed int
	if exceptToken == "" {
		for _, rec := range records {
			if now.Before(rec.ExpiresAt) {
				removed++
			}
		}
		if err := tokens.DeleteByUserID(userID); err != nil {
			return 0, err
		}
	} else {
		removed, err = tokens.DeleteByUserIDExcept(userID, exceptToken)
		if err != nil {
			return 0, err
		}
	}
	_ = audit.Record(audit.Entry{
		ActorID:    fmt.Sprintf("%d", userID),
		Action:     "auth.logout_all",
		ObjectType: "user",
		ObjectID:   fmt.Sprintf("%d", userID),
		Summary:    fmt.Sprintf("吊销 %d 个登录会话", removed),
		Metadata: map[string]interface{}{
			"removed":     removed,
			"keepCurrent": exceptToken != "",
		},
	})
	return removed, nil
}

// CleanupExpiredSessions deletes expired tokens from storage.
func CleanupExpiredSessions() (int, error) {
	tokens, err := requireTokenRepo()
	if err != nil {
		return 0, err
	}
	return tokens.DeleteExpired()
}

func tokenHint(token string) string {
	if len(token) <= 8 {
		return token
	}
	return token[len(token)-8:]
}
