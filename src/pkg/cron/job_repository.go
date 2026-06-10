package cron

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

const jobKindCron = "cron"

type jobRepository struct {
	db *sql.DB
}

func newJobRepository(db *sql.DB) *jobRepository {
	if db == nil {
		return nil
	}
	return &jobRepository{db: db}
}

func (r *jobRepository) List(includeDisabled bool) ([]CronJob, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("cron job repository 未初始化")
	}
	rows, err := r.db.Query(`
		SELECT detail_json FROM jobs
		WHERE kind = ?
		ORDER BY updated_at DESC, created_at DESC
	`, jobKindCron)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []CronJob{}
	for rows.Next() {
		var detail string
		if err := rows.Scan(&detail); err != nil {
			return nil, err
		}
		var job CronJob
		if err := json.Unmarshal([]byte(detail), &job); err != nil {
			return nil, err
		}
		if job.Enabled || includeDisabled {
			out = append(out, normalizeCronJobForStorage(job))
		}
	}
	return out, rows.Err()
}

func (r *jobRepository) Get(id string) (CronJob, bool, error) {
	if r == nil || r.db == nil {
		return CronJob{}, false, fmt.Errorf("cron job repository 未初始化")
	}
	id = strings.TrimSpace(id)
	var detail string
	err := r.db.QueryRow(`
		SELECT detail_json FROM jobs WHERE id = ? AND kind = ?
	`, id, jobKindCron).Scan(&detail)
	if err == sql.ErrNoRows {
		return CronJob{}, false, nil
	}
	if err != nil {
		return CronJob{}, false, err
	}
	var job CronJob
	if err := json.Unmarshal([]byte(detail), &job); err != nil {
		return CronJob{}, false, err
	}
	return normalizeCronJobForStorage(job), true, nil
}

func (r *jobRepository) Upsert(job CronJob) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cron job repository 未初始化")
	}
	job = normalizeCronJobForStorage(job)
	if strings.TrimSpace(job.ID) == "" {
		return fmt.Errorf("cron job id 不能为空")
	}
	detail, err := json.Marshal(job)
	if err != nil {
		return err
	}
	now := time.Now().UnixMilli()
	if job.CreatedAtMs == 0 {
		job.CreatedAtMs = now
	}
	if job.UpdatedAtMs == 0 {
		job.UpdatedAtMs = now
	}

	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertCronJobTx(tx, job, string(detail)); err != nil {
		return err
	}
	if err := upsertJobScheduleTx(tx, job); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *jobRepository) Delete(id string) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cron job repository 未初始化")
	}
	id = strings.TrimSpace(id)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM job_schedules WHERE job_id = ?`, id); err != nil {
		return err
	}
	res, err := tx.Exec(`DELETE FROM jobs WHERE id = ? AND kind = ?`, id, jobKindCron)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return tx.Commit()
}

func (r *jobRepository) ImportJSON(storePath string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("cron job repository 未初始化")
	}
	storePath = strings.TrimSpace(storePath)
	if storePath == "" {
		return 0, nil
	}
	if _, err := os.Stat(storePath); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	store, err := LoadStore(storePath)
	if err != nil {
		return 0, err
	}
	if len(store.Jobs) == 0 {
		return 0, backupCronJSONStore(storePath)
	}
	imported := 0
	for _, job := range store.Jobs {
		if strings.TrimSpace(job.ID) == "" {
			continue
		}
		if err := r.Upsert(job); err != nil {
			return imported, err
		}
		imported++
	}
	if imported > 0 {
		if err := backupCronJSONStore(storePath); err != nil {
			return imported, err
		}
	}
	return imported, nil
}

func (r *jobRepository) ImportLegacyCronJobs() (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("cron job repository 未初始化")
	}
	rows, err := r.db.Query(`SELECT detail_json FROM cron_jobs ORDER BY id`)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	jobs := []CronJob{}
	for rows.Next() {
		var detail string
		if err := rows.Scan(&detail); err != nil {
			return 0, err
		}
		var job CronJob
		if err := json.Unmarshal([]byte(detail), &job); err != nil {
			return 0, err
		}
		if strings.TrimSpace(job.ID) == "" {
			continue
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(jobs) == 0 {
		return 0, nil
	}

	imported := 0
	for _, job := range jobs {
		existing, found, err := r.Get(job.ID)
		if err != nil {
			return imported, err
		}
		if found {
			continue
		}
		if err := r.Upsert(job); err != nil {
			return imported, err
		}
		imported++
		_ = existing
	}
	if imported > 0 {
		if _, err := r.db.Exec(`DELETE FROM cron_jobs`); err != nil {
			return imported, err
		}
	}
	return imported, nil
}

func upsertCronJobTx(tx *sql.Tx, job CronJob, detail string) error {
	deliveryJSON := "{}"
	if job.Delivery != nil {
		if b, err := json.Marshal(job.Delivery); err == nil {
			deliveryJSON = string(b)
		}
	}
	_, err := tx.Exec(`
		INSERT INTO jobs (
			id, kind, name, enabled, schedule_expr, agent_id, session_key,
			delivery_json, detail_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind = excluded.kind,
			name = excluded.name,
			enabled = excluded.enabled,
			schedule_expr = excluded.schedule_expr,
			agent_id = excluded.agent_id,
			session_key = excluded.session_key,
			delivery_json = excluded.delivery_json,
			detail_json = excluded.detail_json,
			updated_at = excluded.updated_at
	`, job.ID, jobKindCron, job.Name, boolToInt(job.Enabled), scheduleExpr(job.Schedule),
		strings.TrimSpace(job.AgentID), strings.TrimSpace(job.SessionKey), deliveryJSON, detail,
		job.CreatedAtMs, job.UpdatedAtMs)
	return err
}

func upsertJobScheduleTx(tx *sql.Tx, job CronJob) error {
	anchorMs := int64(0)
	if job.Schedule.AnchorMs != nil {
		anchorMs = *job.Schedule.AnchorMs
	}
	now := time.Now().UnixMilli()
	createdAt := job.CreatedAtMs
	if createdAt == 0 {
		createdAt = now
	}
	updatedAt := job.UpdatedAtMs
	if updatedAt == 0 {
		updatedAt = now
	}
	_, err := tx.Exec(`
		INSERT INTO job_schedules (
			job_id, schedule_kind, schedule_at, every_ms, anchor_ms, cron_expr, timezone, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(job_id) DO UPDATE SET
			schedule_kind = excluded.schedule_kind,
			schedule_at = excluded.schedule_at,
			every_ms = excluded.every_ms,
			anchor_ms = excluded.anchor_ms,
			cron_expr = excluded.cron_expr,
			timezone = excluded.timezone,
			updated_at = excluded.updated_at
	`, job.ID, strings.TrimSpace(job.Schedule.Kind), strings.TrimSpace(job.Schedule.At),
		job.Schedule.EveryMs, anchorMs, strings.TrimSpace(job.Schedule.Expr), strings.TrimSpace(job.Schedule.Tz),
		createdAt, updatedAt)
	return err
}

func scheduleExpr(sched CronSchedule) string {
	switch strings.TrimSpace(sched.Kind) {
	case "cron":
		return strings.TrimSpace(sched.Expr)
	case "every":
		return fmt.Sprintf("every:%d", sched.EveryMs)
	case "at":
		return "at:" + strings.TrimSpace(sched.At)
	default:
		if strings.TrimSpace(sched.Kind) == "" {
			return ""
		}
		return strings.TrimSpace(sched.Kind)
	}
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func normalizeCronJobForStorage(job CronJob) CronJob {
	job.ID = strings.TrimSpace(job.ID)
	job.AgentID = strings.TrimSpace(strings.ToLower(job.AgentID))
	job.DigitalEmployeeID = normalizeDigitalEmployeeID(job.DigitalEmployeeID)
	job.Name = strings.TrimSpace(job.Name)
	job.Description = strings.TrimSpace(job.Description)
	job.SessionTarget = strings.TrimSpace(job.SessionTarget)
	job.SessionKey = strings.TrimSpace(job.SessionKey)
	job.WakeMode = strings.TrimSpace(job.WakeMode)
	if job.SessionTarget == "" {
		job.SessionTarget = "main"
	}
	if job.WakeMode == "" {
		job.WakeMode = "next-heartbeat"
	}
	return job
}

func backupCronJSONStore(storePath string) error {
	if strings.TrimSpace(storePath) == "" {
		return nil
	}
	if _, err := os.Stat(storePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	backupPath := fmt.Sprintf("%s.bak.%d", storePath, time.Now().UnixMilli())
	return os.Rename(storePath, backupPath)
}
