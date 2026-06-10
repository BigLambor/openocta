package ops

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/openocta/openocta/pkg/db"
)

func persistInspectionReport(report InspectionReport) error {
	sqliteDB := db.GetDB()
	if sqliteDB == nil {
		return nil
	}
	id := strings.TrimSpace(report.ID)
	if id == "" {
		return fmt.Errorf("inspection report id required")
	}
	body, err := json.Marshal(report)
	if err != nil {
		return err
	}
	requiresApproval := 0
	if report.RequiresApproval != nil && *report.RequiresApproval {
		requiresApproval = 1
	}
	now := time.Now().UnixMilli()
	if report.FinishedAt > 0 {
		now = report.FinishedAt
	}
	score := sql.NullInt64{}
	if report.Score != nil {
		score = sql.NullInt64{Int64: int64(*report.Score), Valid: true}
	}
	_, err = sqliteDB.Exec(`
		INSERT INTO inspection_reports (
			id, run_id, job_id, cluster_id, domain, scenario_key,
			score, score_status, validation_status, confidence, summary,
			requires_approval, report_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			run_id = excluded.run_id,
			job_id = excluded.job_id,
			cluster_id = excluded.cluster_id,
			domain = excluded.domain,
			scenario_key = excluded.scenario_key,
			score = excluded.score,
			score_status = excluded.score_status,
			validation_status = excluded.validation_status,
			confidence = excluded.confidence,
			summary = excluded.summary,
			requires_approval = excluded.requires_approval,
			report_json = excluded.report_json,
			updated_at = excluded.updated_at
	`,
		id,
		runIDFromInspectionReport(report),
		strings.TrimSpace(report.JobID),
		strings.TrimSpace(report.ClusterID),
		strings.TrimSpace(report.Domain),
		ScenarioKeyForInspection(report),
		score,
		strings.TrimSpace(report.ScoreStatus),
		strings.TrimSpace(report.ValidationStatus),
		strings.TrimSpace(report.Confidence),
		strings.TrimSpace(report.Summary),
		requiresApproval,
		string(body),
		report.StartedAt,
		now,
	)
	return err
}

// ListInspectionReportsByRunIDs loads reports whose run_id or id matches child L2 runs.
func ListInspectionReportsByRunIDs(runIDs []string) (map[string]InspectionReport, error) {
	sqliteDB := db.GetDB()
	out := map[string]InspectionReport{}
	if sqliteDB == nil || len(runIDs) == 0 {
		return out, nil
	}
	keys := make([]string, 0, len(runIDs)*2)
	for _, id := range runIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		keys = append(keys, id, "inspection-"+id)
	}
	if len(keys) == 0 {
		return out, nil
	}
	placeholders := strings.TrimRight(strings.Repeat("?,", len(keys)), ",")
	args := make([]interface{}, len(keys))
	for i, k := range keys {
		args[i] = k
	}
	rows, err := sqliteDB.Query(`
		SELECT report_json, run_id, id FROM inspection_reports
		WHERE run_id IN (`+placeholders+`) OR id IN (`+placeholders+`)
	`, append(args, args...)...)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var body, runID, id string
		if err := rows.Scan(&body, &runID, &id); err != nil {
			return out, err
		}
		var report InspectionReport
		if err := json.Unmarshal([]byte(body), &report); err != nil {
			continue
		}
		for _, raw := range runIDs {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if id == raw || runID == raw || runID == "inspection-"+raw || id == "inspection-"+raw {
				out[raw] = report
			}
		}
	}
	return out, rows.Err()
}
