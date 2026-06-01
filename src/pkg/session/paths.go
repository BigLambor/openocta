// Package session provides session and transcript path resolution.
// Mirrors src/config/sessions/paths.ts.
package session

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/openocta/openocta/pkg/paths"
)

const (
	// DefaultAgentID is the default agent identifier.
	DefaultAgentID = "main"
)

// ContextKey represents a context key.
type ContextKey string

const (
	ContextKeySessionID      = ContextKey("sessionID")
	ContextKeySessionKey     = ContextKey("sessionKey")
	ContextKeyTranscriptPath = ContextKey("transcriptPath")
	ContextKeyPrompt         = ContextKey("prompt")
)

// SafeSessionIDRe validates session IDs.
var SafeSessionIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,127}$`)

// SanitizeForSessionID converts a string (e.g. from session key parts) to a valid session ID.
// Replaces colons and other invalid chars with dashes, collapses runs, trims, and truncates.
// Returns "main" if the result would be empty or invalid.
func SanitizeForSessionID(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "main"
	}
	// Replace colons (common in keys like employee:sre:run:uuid) and other invalid chars with dash
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			b.WriteRune(r)
		} else if r == ':' || r == ' ' || r == '/' || r == '\\' {
			if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
				b.WriteByte('-')
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 128 {
		out = out[:128]
	}
	out = strings.Trim(out, "-")
	if out == "" || !SafeSessionIDRe.MatchString(out) {
		return "main"
	}
	return out
}

// ResolveAgentSessionsDir returns the sessions directory for an agent.
func ResolveAgentSessionsDir(agentID string, env func(string) string) string {
	if env == nil {
		env = os.Getenv
	}
	stateDir := paths.ResolveStateDir(env)
	id := normalizeAgentID(agentID)
	return filepath.Join(stateDir, "agents", id, "sessions")
}

// ResolveSessionTranscriptsDir returns the default agent sessions dir.
func ResolveSessionTranscriptsDir(env func(string) string) string {
	return ResolveAgentSessionsDir(DefaultAgentID, env)
}

// ResolveSessionFilePath returns the transcript file path for a session.
func ResolveSessionFilePath(sessionID string, opts *SessionPathOptions, env func(string) string) string {
	sessionsDir := ResolveSessionTranscriptsDir(env)
	if opts != nil && opts.SessionsDir != "" {
		sessionsDir = opts.SessionsDir
	}
	// Session files are stored as <sessionId>.jsonl in sessions dir
	return filepath.Join(sessionsDir, sessionID+".jsonl")
}

// SessionPathOptions holds options for resolving session paths.
type SessionPathOptions struct {
	AgentID     string
	SessionsDir string
}

// ValidateSessionID validates and returns trimmed session ID.
func ValidateSessionID(sessionID string) (string, error) {
	s := strings.TrimSpace(sessionID)
	if s == "" {
		return "", ErrEmptySessionID
	}
	if !SafeSessionIDRe.MatchString(s) {
		return "", ErrInvalidSessionID
	}
	return s, nil
}

// ErrEmptySessionID is returned when session ID is empty.
var ErrEmptySessionID = &SessionError{Msg: "session ID must not be empty"}

// ErrInvalidSessionID is returned when session ID format is invalid.
var ErrInvalidSessionID = &SessionError{Msg: "invalid session ID format"}

// SessionError is a session-related error.
type SessionError struct {
	Msg string
}

func (e *SessionError) Error() string {
	return e.Msg
}

func normalizeAgentID(id string) string {
	s := strings.TrimSpace(strings.ToLower(id))
	if s == "" {
		return DefaultAgentID
	}
	return s
}
