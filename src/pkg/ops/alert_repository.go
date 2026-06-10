package ops

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type alertRepository struct {
	db *sql.DB
}

func newAlertRepository(db *sql.DB) *alertRepository {
	if db == nil {
		return nil
	}
	return &alertRepository{db: db}
}

func (r *alertRepository) List() ([]AlertGroup, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("alert repository 未初始化")
	}
	rows, err := r.db.Query(`SELECT detail_json FROM alert_groups ORDER BY updated_at DESC, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AlertGroup{}
	for rows.Next() {
		var detail string
		if err := rows.Scan(&detail); err != nil {
			return nil, err
		}
		var g AlertGroup
		if err := json.Unmarshal([]byte(detail), &g); err != nil {
			return nil, err
		}
		out = append(out, normalizeAlertGroupForStorage(g))
	}
	return out, rows.Err()
}

func (r *alertRepository) Get(id string) (AlertGroup, error) {
	if r == nil || r.db == nil {
		return AlertGroup{}, fmt.Errorf("alert repository 未初始化")
	}
	var detail string
	err := r.db.QueryRow(`SELECT detail_json FROM alert_groups WHERE id = ?`, strings.TrimSpace(id)).Scan(&detail)
	if err == sql.ErrNoRows {
		return AlertGroup{}, fmt.Errorf("告警组不存在: %s", id)
	}
	if err != nil {
		return AlertGroup{}, err
	}
	var g AlertGroup
	if err := json.Unmarshal([]byte(detail), &g); err != nil {
		return AlertGroup{}, err
	}
	return normalizeAlertGroupForStorage(g), nil
}

func (r *alertRepository) Upsert(g AlertGroup) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("alert repository 未初始化")
	}
	g = normalizeAlertGroupForStorage(g)
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertAlertGroupTx(tx, g); err != nil {
		return err
	}
	if err := replaceAlertEventsTx(tx, g); err != nil {
		return err
	}
	if err := replaceAlertTimelineTx(tx, g); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *alertRepository) ReplaceAll(groups []AlertGroup) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("alert repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM incident_timeline WHERE subject_type = 'alert_group'`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM alert_events`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM alert_groups`); err != nil {
		return err
	}
	for _, g := range groups {
		g = normalizeAlertGroupForStorage(g)
		if err := upsertAlertGroupTx(tx, g); err != nil {
			return err
		}
		if err := replaceAlertEventsTx(tx, g); err != nil {
			return err
		}
		if err := replaceAlertTimelineTx(tx, g); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *alertRepository) ImportJSON(path string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("alert repository 未初始化")
	}
	if strings.TrimSpace(path) == "" {
		return 0, nil
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	store, err := loadAlertsStore(path)
	if err != nil {
		return 0, err
	}
	imported := 0
	for _, g := range store.Groups {
		if strings.TrimSpace(g.ID) == "" {
			continue
		}
		if err := r.Upsert(g); err != nil {
			return imported, err
		}
		imported++
	}
	if err := backupJSONStore(path); err != nil {
		return imported, err
	}
	return imported, nil
}

func upsertAlertGroupTx(tx *sql.Tx, g AlertGroup) error {
	detail, err := json.Marshal(g)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO alert_groups (
			id, source, domain, title, severity, status, original_count, reduced_to,
			session_key, run_id, alertname, service, instance, cluster_id, component,
			review_status, suppression_category, suppression_detail, created_at, updated_at, detail_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source = excluded.source,
			domain = excluded.domain,
			title = excluded.title,
			severity = excluded.severity,
			status = excluded.status,
			original_count = excluded.original_count,
			reduced_to = excluded.reduced_to,
			session_key = excluded.session_key,
			run_id = excluded.run_id,
			alertname = excluded.alertname,
			service = excluded.service,
			instance = excluded.instance,
			cluster_id = excluded.cluster_id,
			component = excluded.component,
			review_status = excluded.review_status,
			suppression_category = excluded.suppression_category,
			suppression_detail = excluded.suppression_detail,
			updated_at = excluded.updated_at,
			detail_json = excluded.detail_json
	`, g.ID, g.Source, g.Domain, g.Title, g.Severity, g.Status, g.OriginalCount, g.ReducedTo,
		g.SessionKey, g.RunID, g.Alertname, g.Service, g.Instance, g.ClusterID, g.Component,
		g.ReviewStatus, g.SuppressionCategory, g.SuppressionDetail, g.CreatedAtMs, g.UpdatedAtMs, string(detail))
	return err
}

func replaceAlertEventsTx(tx *sql.Tx, g AlertGroup) error {
	if _, err := tx.Exec(`DELETE FROM alert_events WHERE group_id = ?`, g.ID); err != nil {
		return err
	}
	for i, ev := range g.Events {
		id := strings.TrimSpace(ev.AlertID)
		if id == "" {
			id = fmt.Sprintf("%s-event-%03d", g.ID, i)
		}
		raw, err := json.Marshal(ev)
		if err != nil {
			raw = []byte("{}")
		}
		received := ev.ReceivedAt
		if received == 0 {
			received = g.CreatedAtMs
		}
		if _, err := tx.Exec(`
			INSERT INTO alert_events (
				id, group_id, source, severity, title, message, alertname, service, instance,
				cluster_id, component, raw_json, received_at, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, id, g.ID, g.Source, normalizeSeverity(ev.Severity), ev.Title, ev.Message, ev.Alertname, ev.Service, ev.Instance,
			ev.ClusterID, ev.Component, string(raw), received, received, g.UpdatedAtMs); err != nil {
			return err
		}
	}
	return nil
}

func replaceAlertTimelineTx(tx *sql.Tx, g AlertGroup) error {
	if _, err := tx.Exec(`DELETE FROM incident_timeline WHERE subject_type = 'alert_group' AND subject_id = ?`, g.ID); err != nil {
		return err
	}
	for i, item := range g.Timeline {
		evidence, err := json.Marshal(map[string]interface{}{
			"type":     item.Type,
			"operator": item.Operator,
		})
		if err != nil {
			evidence = []byte("{}")
		}
		ts := item.TimestampMs
		if ts == 0 {
			ts = g.UpdatedAtMs
		}
		runID := strings.TrimSpace(item.RunID)
		if _, err := tx.Exec(`
			INSERT INTO incident_timeline (
				id, subject_type, subject_id, event_type, operator_id, run_id, message,
				evidence_json, created_at, updated_at
			) VALUES (?, 'alert_group', ?, ?, ?, ?, ?, ?, ?, ?)
		`, fmt.Sprintf("%s-timeline-%03d", g.ID, i), g.ID, item.Type, item.Operator, runID, item.Message, string(evidence), ts, ts); err != nil {
			return err
		}
	}
	return nil
}

func normalizeAlertGroupForStorage(g AlertGroup) AlertGroup {
	g.ID = strings.TrimSpace(g.ID)
	g.Source = strings.TrimSpace(g.Source)
	if g.Source == "" {
		g.Source = "default"
	}
	g.Domain = strings.TrimSpace(strings.ToLower(g.Domain))
	g.Title = strings.TrimSpace(g.Title)
	if g.Title == "" {
		g.Title = "合并告警组"
	}
	g.Severity = normalizeSeverity(g.Severity)
	g.Status = strings.TrimSpace(strings.ToLower(g.Status))
	if g.Status == "" {
		g.Status = AlertStatusActive
	}
	if g.ReducedTo == 0 {
		g.ReducedTo = 1
	}
	if g.OriginalCount == 0 && len(g.Events) > 0 {
		g.OriginalCount = len(g.Events)
	}
	if strings.TrimSpace(g.SuppressionCategory) == "" {
		g.SuppressionCategory = "none"
	}
	if strings.TrimSpace(g.ReviewStatus) == "" {
		g.ReviewStatus = "pending"
	}
	now := nowMs()
	if g.CreatedAtMs == 0 {
		g.CreatedAtMs = now
	}
	if g.UpdatedAtMs == 0 {
		g.UpdatedAtMs = g.CreatedAtMs
	}
	return g
}
