package rbac

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/audit"
)

const (
	defaultLoginMaxAttempts    = 5
	defaultLoginLockoutDuration  = 15 * time.Minute
	defaultLoginAttemptWindow    = 15 * time.Minute
	envLoginMaxAttempts          = "OPENOCTA_LOGIN_MAX_ATTEMPTS"
	envLoginLockoutMinutes       = "OPENOCTA_LOGIN_LOCKOUT_MINUTES"
	envLoginAttemptWindowMinutes = "OPENOCTA_LOGIN_ATTEMPT_WINDOW_MINUTES"
)

// LoginLockStatus describes whether a login attempt is currently allowed.
type LoginLockStatus struct {
	Allowed     bool
	LockedUntil time.Time
	Reason      string
}

func loginGuardDB() (*sql.DB, error) {
	if sqlDB == nil {
		return nil, fmt.Errorf("RBAC 数据库未初始化")
	}
	return sqlDB, nil
}

func loginMaxAttempts() int {
	if v := strings.TrimSpace(os.Getenv(envLoginMaxAttempts)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultLoginMaxAttempts
}

func loginLockoutDuration() time.Duration {
	if v := strings.TrimSpace(os.Getenv(envLoginLockoutMinutes)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return defaultLoginLockoutDuration
}

func loginAttemptWindow() time.Duration {
	if v := strings.TrimSpace(os.Getenv(envLoginAttemptWindowMinutes)); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return time.Duration(n) * time.Minute
		}
	}
	return defaultLoginAttemptWindow
}

// CheckLoginAllowed reports whether login should proceed for the given IP and username.
func CheckLoginAllowed(ip, username string) (LoginLockStatus, error) {
	db, err := loginGuardDB()
	if err != nil {
		return LoginLockStatus{}, err
	}
	now := time.Now()
	for _, item := range []struct {
		scope string
		key   string
	}{
		{"ip", normalizeLoginKey(ip)},
		{"username", normalizeLoginKey(username)},
	} {
		if item.key == "" {
			continue
		}
		status, err := lookupLockout(db, item.scope, item.key, now)
		if err != nil {
			return LoginLockStatus{}, err
		}
		if !status.Allowed {
			return status, nil
		}
	}
	return LoginLockStatus{Allowed: true}, nil
}

// RecordLoginFailure increments failure counters and may lock the IP/username.
func RecordLoginFailure(ip, username string) (LoginLockStatus, error) {
	db, err := loginGuardDB()
	if err != nil {
		return LoginLockStatus{}, err
	}
	now := time.Now()
	maxAttempts := loginMaxAttempts()
	lockout := loginLockoutDuration()
	window := loginAttemptWindow()

	lockedUntil := time.Time{}
	failCount := 0
	for _, item := range []struct {
		scope string
		key   string
	}{
		{"ip", normalizeLoginKey(ip)},
		{"username", normalizeLoginKey(username)},
	} {
		if item.key == "" {
			continue
		}
		count, until, err := bumpLockout(db, item.scope, item.key, now, maxAttempts, lockout, window)
		if err != nil {
			return LoginLockStatus{}, err
		}
		if count > failCount {
			failCount = count
		}
		if until.After(lockedUntil) {
			lockedUntil = until
		}
	}

	_ = audit.Record(audit.Entry{
		ActorID:    normalizeLoginKey(username),
		Action:     "auth.login_failed",
		ObjectType: "user",
		ObjectID:   normalizeLoginKey(username),
		Summary:    "登录失败",
		Metadata: map[string]interface{}{
			"ip":        normalizeLoginKey(ip),
			"failCount": failCount,
			"locked":    !lockedUntil.IsZero() && lockedUntil.After(now),
		},
	})

	if !lockedUntil.IsZero() && lockedUntil.After(now) {
		_ = audit.Record(audit.Entry{
			ActorID:    normalizeLoginKey(username),
			Action:     "auth.login_locked",
			ObjectType: "user",
			ObjectID:   normalizeLoginKey(username),
			Summary:    "登录失败次数过多，账号或来源 IP 已被临时锁定",
			Metadata: map[string]interface{}{
				"ip":          normalizeLoginKey(ip),
				"lockedUntil": lockedUntil.UTC().Format(time.RFC3339),
				"failCount":   failCount,
			},
		})
		return LoginLockStatus{
			Allowed:     false,
			LockedUntil: lockedUntil,
			Reason:      "登录失败次数过多，请稍后再试",
		}, nil
	}

	return LoginLockStatus{Allowed: true}, nil
}

// RecordLoginSuccess clears lockout counters after a successful login.
func RecordLoginSuccess(ip, username string) error {
	db, err := loginGuardDB()
	if err != nil {
		return err
	}
	for _, item := range []struct {
		scope string
		key   string
	}{
		{"ip", normalizeLoginKey(ip)},
		{"username", normalizeLoginKey(username)},
	} {
		if item.key == "" {
			continue
		}
		if err := clearLockout(db, item.scope, item.key); err != nil {
			return err
		}
	}
	_ = audit.Record(audit.Entry{
		ActorID:    normalizeLoginKey(username),
		Action:     "auth.login_success",
		ObjectType: "user",
		ObjectID:   normalizeLoginKey(username),
		Summary:    "登录成功",
		Metadata: map[string]interface{}{
			"ip": normalizeLoginKey(ip),
		},
	})
	return nil
}

func normalizeLoginKey(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

func lookupLockout(db *sql.DB, scope, key string, now time.Time) (LoginLockStatus, error) {
	var lockedUntilMs int64
	err := db.QueryRow(`
		SELECT locked_until FROM login_lockouts WHERE scope = ? AND key = ?
	`, scope, key).Scan(&lockedUntilMs)
	if err == sql.ErrNoRows {
		return LoginLockStatus{Allowed: true}, nil
	}
	if err != nil {
		return LoginLockStatus{}, err
	}
	if lockedUntilMs <= 0 {
		return LoginLockStatus{Allowed: true}, nil
	}
	lockedUntil := time.UnixMilli(lockedUntilMs)
	if now.Before(lockedUntil) {
		return LoginLockStatus{
			Allowed:     false,
			LockedUntil: lockedUntil,
			Reason:      "登录失败次数过多，请稍后再试",
		}, nil
	}
	return LoginLockStatus{Allowed: true}, nil
}

func bumpLockout(db *sql.DB, scope, key string, now time.Time, maxAttempts int, lockout, window time.Duration) (int, time.Time, error) {
	nowMs := now.UnixMilli()
	windowStartMs := now.Add(-window).UnixMilli()

	var failCount int
	var lockedUntilMs int64
	var updatedAtMs int64
	err := db.QueryRow(`
		SELECT fail_count, locked_until, updated_at FROM login_lockouts WHERE scope = ? AND key = ?
	`, scope, key).Scan(&failCount, &lockedUntilMs, &updatedAtMs)
	if err != nil && err != sql.ErrNoRows {
		return 0, time.Time{}, err
	}
	if err == sql.ErrNoRows || updatedAtMs < windowStartMs {
		failCount = 0
	}

	failCount++
	lockedUntil := time.Time{}
	if failCount >= maxAttempts {
		lockedUntil = now.Add(lockout)
		lockedUntilMs = lockedUntil.UnixMilli()
	} else {
		lockedUntilMs = 0
	}

	_, err = db.Exec(`
		INSERT INTO login_lockouts (scope, key, fail_count, locked_until, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(scope, key) DO UPDATE SET
			fail_count = excluded.fail_count,
			locked_until = excluded.locked_until,
			updated_at = excluded.updated_at
	`, scope, key, failCount, lockedUntilMs, nowMs)
	if err != nil {
		return 0, time.Time{}, err
	}
	return failCount, lockedUntil, nil
}

func clearLockout(db *sql.DB, scope, key string) error {
	_, err := db.Exec(`DELETE FROM login_lockouts WHERE scope = ? AND key = ?`, scope, key)
	return err
}
