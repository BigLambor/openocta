package security

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	openoctadb "github.com/openocta/openocta/pkg/db"
)

const (
	approvalSubjectToolExecution = "tool_execution"
	approvalRiskHigh             = "high"
)

type approvalRepository struct {
	db *sql.DB
}

var sharedApprovalRepo *approvalRepository

func initApprovalRepository() {
	if sharedApprovalRepo != nil {
		return
	}
	db := openoctadb.GetDB()
	if db == nil {
		return
	}
	sharedApprovalRepo = &approvalRepository{db: db}
}

func defaultApprovalRepository() *approvalRepository {
	initApprovalRepository()
	return sharedApprovalRepo
}

func (r *approvalRepository) Load() (map[string]*ApprovalRecord, map[string]time.Time, error) {
	records := make(map[string]*ApprovalRecord)
	whitelist := make(map[string]time.Time)
	if r == nil || r.db == nil {
		return records, whitelist, nil
	}

	rows, err := r.db.Query(`
		SELECT id, subject_id, status, reason, request_json, result_json, expires_at, created_at, updated_at
		FROM approvals
		WHERE subject_type = ?
		ORDER BY created_at ASC
	`, approvalSubjectToolExecution)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	for rows.Next() {
		rec, err := scanApprovalRow(rows)
		if err != nil {
			return nil, nil, err
		}
		records[rec.ID] = rec
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	wRows, err := r.db.Query(`
		SELECT session_id, expires_at
		FROM approval_session_whitelist
		WHERE tenant_id = 'default' AND workspace_id = 'default'
	`)
	if err != nil {
		return nil, nil, err
	}
	defer wRows.Close()
	for wRows.Next() {
		var sessionID string
		var expiresAtMs int64
		if err := wRows.Scan(&sessionID, &expiresAtMs); err != nil {
			return nil, nil, err
		}
		whitelist[sessionID] = time.UnixMilli(expiresAtMs).UTC()
	}
	return records, whitelist, wRows.Err()
}

func (r *approvalRepository) Persist(records map[string]*ApprovalRecord, whitelist map[string]time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("approval repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM approvals WHERE subject_type = ?`, approvalSubjectToolExecution); err != nil {
		return err
	}
	for _, rec := range records {
		if err := upsertApprovalTx(tx, rec); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM approval_session_whitelist WHERE tenant_id = 'default' AND workspace_id = 'default'`); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for sessionID, expiry := range whitelist {
		if strings.TrimSpace(sessionID) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO approval_session_whitelist (session_id, tenant_id, workspace_id, expires_at, created_at, updated_at)
			VALUES (?, 'default', 'default', ?, ?, ?)
		`, sessionID, expiry.UnixMilli(), now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *approvalRepository) SaveRecord(rec *ApprovalRecord) error {
	if r == nil || r.db == nil || rec == nil {
		return fmt.Errorf("approval repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := upsertApprovalTx(tx, rec); err != nil {
		return err
	}
	if rec.State == ApprovalDenied || (rec.State == ApprovalApproved && !rec.AutoApproved) {
		if err := insertApprovalStepTx(tx, rec); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *approvalRepository) SaveWhitelist(whitelist map[string]time.Time) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("approval repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM approval_session_whitelist WHERE tenant_id = 'default' AND workspace_id = 'default'`); err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	for sessionID, expiry := range whitelist {
		if strings.TrimSpace(sessionID) == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO approval_session_whitelist (session_id, tenant_id, workspace_id, expires_at, created_at, updated_at)
			VALUES (?, 'default', 'default', ?, ?, ?)
		`, sessionID, expiry.UnixMilli(), now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *approvalRepository) ImportFromJSONIfEmpty(jsonPath string) error {
	if r == nil || r.db == nil {
		return nil
	}
	var count int
	if err := r.db.QueryRow(`SELECT COUNT(*) FROM approvals WHERE subject_type = ?`, approvalSubjectToolExecution).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	jsonPath = strings.TrimSpace(jsonPath)
	if jsonPath == "" {
		return nil
	}
	store := &jsonApprovalStore{path: jsonPath}
	records, whitelist, err := store.Load()
	if err != nil {
		return err
	}
	if len(records) == 0 && len(whitelist) == 0 {
		return nil
	}
	if err := r.Persist(records, whitelist); err != nil {
		return err
	}
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%d", jsonPath, time.Now().UnixMilli())
	if err := os.WriteFile(backupPath, data, 0o600); err != nil {
		return err
	}
	return os.Remove(jsonPath)
}

func upsertApprovalTx(tx *sql.Tx, rec *ApprovalRecord) error {
	if rec == nil {
		return nil
	}
	requestJSON, resultJSON, expiresAtMs, createdAtMs, updatedAtMs := approvalRecordToDB(rec)
	now := time.Now().UnixMilli()
	if createdAtMs <= 0 {
		createdAtMs = now
	}
	if updatedAtMs <= 0 {
		updatedAtMs = now
	}
	_, err := tx.Exec(`
		INSERT INTO approvals (
			id, tenant_id, workspace_id, subject_type, subject_id, requester_id,
			status, risk_level, reason, request_json, result_json, expires_at, created_at, updated_at
		) VALUES (?, 'default', 'default', ?, ?, '', ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			subject_id = excluded.subject_id,
			status = excluded.status,
			reason = excluded.reason,
			request_json = excluded.request_json,
			result_json = excluded.result_json,
			expires_at = excluded.expires_at,
			updated_at = excluded.updated_at
	`, rec.ID, approvalSubjectToolExecution, rec.SessionID, string(rec.State), approvalRiskHigh, rec.Reason,
		requestJSON, resultJSON, expiresAtMs, createdAtMs, updatedAtMs)
	return err
}

func insertApprovalStepTx(tx *sql.Tx, rec *ApprovalRecord) error {
	now := time.Now().UnixMilli()
	decidedAt := now
	if rec.ApprovedAt != nil {
		decidedAt = rec.ApprovedAt.UnixMilli()
	}
	_, err := tx.Exec(`
		INSERT INTO approval_steps (
			id, tenant_id, workspace_id, approval_id, step_order, approver_id,
			status, comment, decided_at, created_at, updated_at
		) VALUES (?, 'default', 'default', ?, 1, ?, ?, ?, ?, ?, ?)
	`, "step-"+uuid.New().String(), rec.ID, rec.Approver, string(rec.State), rec.Reason, decidedAt, now, now)
	return err
}

func scanApprovalRow(rows *sql.Rows) (*ApprovalRecord, error) {
	var (
		id, sessionID, status, reason, requestJSON, resultJSON string
		expiresAtMs, createdAtMs, updatedAtMs                  int64
	)
	if err := rows.Scan(&id, &sessionID, &status, &reason, &requestJSON, &resultJSON, &expiresAtMs, &createdAtMs, &updatedAtMs); err != nil {
		return nil, err
	}
	rec := &ApprovalRecord{
		ID:        id,
		SessionID: sessionID,
		State:     ApprovalState(status),
		Reason:    reason,
	}
	if payload, err := decodeApprovalPayload(requestJSON, resultJSON); err == nil {
		if payload.Command != "" {
			rec.Command = payload.Command
		}
		if len(payload.Paths) > 0 {
			rec.Paths = append([]string{}, payload.Paths...)
		}
		rec.AutoApproved = payload.AutoApproved
		if payload.RequestedAt != nil {
			rec.RequestedAt = *payload.RequestedAt
		} else if createdAtMs > 0 {
			rec.RequestedAt = time.UnixMilli(createdAtMs).UTC()
		}
		if payload.ApprovedAt != nil {
			rec.ApprovedAt = payload.ApprovedAt
		}
		if payload.Approver != "" {
			rec.Approver = payload.Approver
		}
		if payload.ExpiresAt != nil {
			rec.ExpiresAt = payload.ExpiresAt
		} else if expiresAtMs > 0 {
			t := time.UnixMilli(expiresAtMs).UTC()
			rec.ExpiresAt = &t
		}
	}
	if rec.RequestedAt.IsZero() && createdAtMs > 0 {
		rec.RequestedAt = time.UnixMilli(createdAtMs).UTC()
	}
	return rec, nil
}

type approvalPayload struct {
	Command      string     `json:"command"`
	Paths        []string   `json:"paths"`
	AutoApproved bool       `json:"auto_approved"`
	RequestedAt  *time.Time `json:"requested_at,omitempty"`
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	Approver     string     `json:"approver,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

func approvalRecordToDB(rec *ApprovalRecord) (requestJSON, resultJSON string, expiresAtMs, createdAtMs, updatedAtMs int64) {
	payload := approvalPayload{
		Command:      rec.Command,
		Paths:        rec.Paths,
		AutoApproved: rec.AutoApproved,
		RequestedAt:  &rec.RequestedAt,
		ApprovedAt:   rec.ApprovedAt,
		Approver:     rec.Approver,
		ExpiresAt:    rec.ExpiresAt,
	}
	reqBytes, _ := json.Marshal(payload)
	requestJSON = string(reqBytes)
	resultJSON = "{}"
	if rec.ApprovedAt != nil || rec.Approver != "" || rec.ExpiresAt != nil {
		res := map[string]interface{}{}
		if rec.ApprovedAt != nil {
			res["approved_at"] = rec.ApprovedAt
		}
		if rec.Approver != "" {
			res["approver"] = rec.Approver
		}
		if rec.ExpiresAt != nil {
			res["expires_at"] = rec.ExpiresAt
		}
		resBytes, _ := json.Marshal(res)
		resultJSON = string(resBytes)
	}
	if rec.ExpiresAt != nil {
		expiresAtMs = rec.ExpiresAt.UnixMilli()
	}
	createdAtMs = rec.RequestedAt.UnixMilli()
	updatedAtMs = time.Now().UnixMilli()
	return requestJSON, resultJSON, expiresAtMs, createdAtMs, updatedAtMs
}

func decodeApprovalPayload(requestJSON, resultJSON string) (approvalPayload, error) {
	var payload approvalPayload
	if strings.TrimSpace(requestJSON) != "" {
		_ = json.Unmarshal([]byte(requestJSON), &payload)
	}
	if strings.TrimSpace(resultJSON) != "" && resultJSON != "{}" {
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(resultJSON), &result); err == nil {
			if v, ok := result["approver"].(string); ok && payload.Approver == "" {
				payload.Approver = v
			}
		}
	}
	return payload, nil
}
