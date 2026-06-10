package handlers

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/config"
	"github.com/openocta/openocta/pkg/gateway/protocol"
	"github.com/openocta/openocta/pkg/paths"
	octasecurity "github.com/openocta/openocta/pkg/security"
)

// approvalRecord mirrors agentsdk-go security.ApprovalRecord JSON fields.
type approvalRecord struct {
	ID           string     `json:"id"`
	SessionID    string     `json:"session_id"`
	Command      string     `json:"command"`
	Paths        []string   `json:"paths"`
	State        string     `json:"state"` // pending/approved/denied
	RequestedAt  time.Time  `json:"requested_at"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	Approver     string     `json:"approver,omitempty"`
	Reason       string     `json:"reason,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	AutoApproved bool       `json:"auto_approved"`
}

func resolveApprovalStoreFile(cfg *config.OpenOctaConfig, env func(string) string) (string, int) {
	timeoutSeconds := 300
	var (
		queueCfg      *config.SandboxApprovalQueue
		approvalStore *string
	)
	if cfg != nil && cfg.Security != nil {
		queueCfg = cfg.Security.ApprovalQueue
		if cfg.Security.Sandbox != nil {
			approvalStore = cfg.Security.Sandbox.ApprovalStore
		}
	}
	if queueCfg != nil && queueCfg.TimeoutSeconds != nil && *queueCfg.TimeoutSeconds > 0 {
		timeoutSeconds = *queueCfg.TimeoutSeconds
	}
	if approvalStore != nil && strings.TrimSpace(*approvalStore) != "" {
		p := strings.TrimSpace(*approvalStore)
		if !strings.HasSuffix(strings.ToLower(p), ".json") {
			return filepath.Join(p, "approvals.json"), timeoutSeconds
		}
		return p, timeoutSeconds
	}
	stateDir := paths.ResolveStateDir(env)
	return filepath.Join(stateDir, "agents", "approvals", "approvals.json"), timeoutSeconds
}

func toListEntry(rec approvalRecord, now time.Time, timeoutSeconds int) map[string]interface{} {
	createdAt := rec.RequestedAt
	timeoutAt := createdAt.Add(time.Duration(timeoutSeconds) * time.Second)
	status := rec.State
	if status == string(octasecurity.ApprovalPending) && timeoutSeconds > 0 && now.After(timeoutAt) {
		status = "expired"
	}

	out := map[string]interface{}{
		"id":        rec.ID,
		"sessionId": rec.SessionID,
		"command":   rec.Command,
		"paths":     rec.Paths,
		"status":    status,
		"createdAt": createdAt.UnixMilli(),
		"timeoutAt": timeoutAt.UnixMilli(),
	}
	if rec.ApprovedAt != nil {
		out["approvedAt"] = rec.ApprovedAt.UnixMilli()
	}
	if rec.Approver != "" {
		out["approver"] = rec.Approver
	}
	if rec.Reason != "" {
		out["reason"] = rec.Reason
	}
	if rec.AutoApproved {
		out["autoApproved"] = true
	}

	var expiresAt time.Time
	switch status {
	case string(octasecurity.ApprovalPending):
		expiresAt = timeoutAt
	case string(octasecurity.ApprovalApproved):
		if rec.ExpiresAt != nil {
			expiresAt = *rec.ExpiresAt
		}
	default:
		expiresAt = time.Time{}
	}
	if !expiresAt.IsZero() {
		out["expiresAt"] = expiresAt.UnixMilli()
		ttl := int(expiresAt.Sub(now).Seconds())
		out["ttlSeconds"] = ttl
		if ttl < 0 {
			out["expired"] = true
		}
	}
	return out
}

func toWhitelistEntry(sessionID string, expiresAt time.Time, now time.Time) map[string]interface{} {
	out := map[string]interface{}{
		"sessionId": sessionID,
		"status":    "whitelisted",
	}
	if !expiresAt.IsZero() {
		out["expiresAt"] = expiresAt.UnixMilli()
		ttl := int(expiresAt.Sub(now).Seconds())
		out["ttlSeconds"] = ttl
		if ttl < 0 {
			out["expired"] = true
			out["status"] = "whitelist_expired"
		}
	} else {
		out["ttlSeconds"] = -1
		out["expiresAt"] = int64(0)
	}
	return out
}

func approvalRecordToListEntry(rec *octasecurity.ApprovalRecord, now time.Time, timeoutSeconds int) map[string]interface{} {
	if rec == nil {
		return nil
	}
	wrapped := approvalRecord{
		ID:           rec.ID,
		SessionID:    rec.SessionID,
		Command:      rec.Command,
		Paths:        rec.Paths,
		State:        string(rec.State),
		RequestedAt:  rec.RequestedAt,
		ApprovedAt:   rec.ApprovedAt,
		Approver:     rec.Approver,
		Reason:       rec.Reason,
		ExpiresAt:    rec.ExpiresAt,
		AutoApproved: rec.AutoApproved,
	}
	return toListEntry(wrapped, now, timeoutSeconds)
}

// ApprovalsListHandler handles "approvals.list".
// Returns approved, pending, denied records and session whitelist, each with status, expiresAt, ttlSeconds.
func ApprovalsListHandler(opts HandlerOpts) error {
	cfg := loadConfigFromContext(opts.Context)
	env := func(k string) string { return os.Getenv(k) }
	storeFile, timeoutSeconds := resolveApprovalStoreFile(cfg, env)

	q, err := octasecurity.GetApprovalQueue(storeFile)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}

	now := time.Now()
	var approved, pendingActive, pendingExpired, denied []map[string]interface{}
	for _, rec := range q.ListAll() {
		entry := approvalRecordToListEntry(rec, now, timeoutSeconds)
		switch rec.State {
		case octasecurity.ApprovalApproved:
			approved = append(approved, entry)
		case octasecurity.ApprovalPending:
			status, _ := entry["status"].(string)
			if status == "expired" {
				pendingExpired = append(pendingExpired, entry)
			} else {
				pendingActive = append(pendingActive, entry)
			}
		case octasecurity.ApprovalDenied:
			denied = append(denied, entry)
		default:
			status, _ := entry["status"].(string)
			if status == "expired" {
				pendingExpired = append(pendingExpired, entry)
			} else {
				pendingActive = append(pendingActive, entry)
			}
		}
	}

	whitelisted := make([]map[string]interface{}, 0)
	for sessionID, expiresAt := range q.WhitelistSnapshot() {
		whitelisted = append(whitelisted, toWhitelistEntry(sessionID, expiresAt, now))
	}

	entries := append([]map[string]interface{}{}, approved...)
	entries = append(entries, pendingActive...)
	entries = append(entries, pendingExpired...)
	entries = append(entries, denied...)

	opts.Respond(true, map[string]interface{}{
		"storePath":      storeFile,
		"storeBackend":   q.StoreBackend(),
		"approved":       approved,
		"pending":        pendingActive,
		"pendingExpired": pendingExpired,
		"denied":         denied,
		"whitelisted":    whitelisted,
		"entries":        entries,
	}, nil, nil)
	return nil
}

func ensureApprovalPendingAndNotExpired(q *octasecurity.ApprovalQueue, requestID string, timeoutSeconds int) (*octasecurity.ApprovalRecord, *protocol.ErrorShape) {
	rec, ok := q.GetRecord(requestID)
	if !ok {
		return nil, &protocol.ErrorShape{Code: protocol.ErrCodeNotFound, Message: "approval request not found"}
	}
	if rec.State != octasecurity.ApprovalPending {
		return nil, &protocol.ErrorShape{Code: protocol.ErrCodeInvalidRequest, Message: "approval not pending"}
	}
	if timeoutSeconds > 0 && time.Now().After(rec.RequestedAt.Add(time.Duration(timeoutSeconds)*time.Second)) {
		return nil, &protocol.ErrorShape{Code: protocol.ErrCodeInvalidRequest, Message: "approval request expired"}
	}
	return rec, nil
}

// ApprovalsApproveHandler handles "approvals.approve".
// This approves the request WITHOUT adding to whitelist (ttl=0).
func ApprovalsApproveHandler(opts HandlerOpts) error {
	requestID, _ := opts.Params["requestId"].(string)
	approverID, _ := opts.Params["approverId"].(string)
	if requestID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "requestId required",
		}, nil)
		return nil
	}

	cfg := loadConfigFromContext(opts.Context)
	env := func(k string) string { return os.Getenv(k) }
	storeFile, timeoutSeconds := resolveApprovalStoreFile(cfg, env)
	q, err := octasecurity.GetApprovalQueue(storeFile)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{Code: protocol.ErrCodeInternal, Message: err.Error()}, nil)
		return nil
	}
	if _, errShape := ensureApprovalPendingAndNotExpired(q, requestID, timeoutSeconds); errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}

	var recordTTL time.Duration
	if cfg != nil && cfg.Security != nil && cfg.Security.ApprovalQueue != nil && cfg.Security.ApprovalQueue.TimeoutSeconds != nil {
		recordTTL = time.Duration(int64(*cfg.Security.ApprovalQueue.TimeoutSeconds)) * time.Second
	} else {
		recordTTL = time.Minute * 5
	}
	if _, err := q.Approve(requestID, strings.TrimSpace(approverID), recordTTL); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"requestId": requestID, "status": "approved"}, nil, nil)
	return nil
}

// ApprovalsDenyHandler handles "approvals.deny".
func ApprovalsDenyHandler(opts HandlerOpts) error {
	requestID, _ := opts.Params["requestId"].(string)
	approverID, _ := opts.Params["approverId"].(string)
	reason, _ := opts.Params["reason"].(string)
	if requestID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "requestId required",
		}, nil)
		return nil
	}

	cfg := loadConfigFromContext(opts.Context)
	env := func(k string) string { return os.Getenv(k) }
	storeFile, timeoutSeconds := resolveApprovalStoreFile(cfg, env)
	q, err := octasecurity.GetApprovalQueue(storeFile)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{Code: protocol.ErrCodeInternal, Message: err.Error()}, nil)
		return nil
	}
	if _, errShape := ensureApprovalPendingAndNotExpired(q, requestID, timeoutSeconds); errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}
	if _, err := q.Deny(requestID, strings.TrimSpace(approverID), strings.TrimSpace(reason)); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeNotFound,
			Message: err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"requestId": requestID, "status": "denied"}, nil, nil)
	return nil
}

// ApprovalsWhitelistSessionHandler handles "approvals.whitelistSession".
// It adds the session of the given request to the whitelist (indefinitely)
// and marks this specific approval as approved.
func ApprovalsWhitelistSessionHandler(opts HandlerOpts) error {
	requestID, _ := opts.Params["requestId"].(string)
	approverID, _ := opts.Params["approverId"].(string)
	if requestID == "" {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInvalidRequest,
			Message: "requestId required",
		}, nil)
		return nil
	}

	cfg := loadConfigFromContext(opts.Context)
	env := func(k string) string { return os.Getenv(k) }
	storeFile, timeoutSeconds := resolveApprovalStoreFile(cfg, env)
	q, err := octasecurity.GetApprovalQueue(storeFile)
	if err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{Code: protocol.ErrCodeInternal, Message: err.Error()}, nil)
		return nil
	}
	rec, errShape := ensureApprovalPendingAndNotExpired(q, requestID, timeoutSeconds)
	if errShape != nil {
		opts.Respond(false, nil, errShape, nil)
		return nil
	}

	var ttl time.Duration
	if cfg != nil && cfg.Security != nil && cfg.Security.ApprovalQueue != nil && cfg.Security.ApprovalQueue.TimeoutSeconds != nil {
		ttl = time.Duration(int64(*cfg.Security.ApprovalQueue.TimeoutSeconds)) * time.Second
	} else {
		ttl = time.Minute * 5
	}

	if err := q.AddSessionToWhitelist(rec.SessionID, ttl); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}

	if _, err := q.Approve(requestID, strings.TrimSpace(approverID), ttl); err != nil {
		opts.Respond(false, nil, &protocol.ErrorShape{
			Code:    protocol.ErrCodeInternal,
			Message: err.Error(),
		}, nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"requestId": requestID,
		"sessionId": rec.SessionID,
		"status":    "whitelisted",
	}, nil, nil)
	return nil
}
