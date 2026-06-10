package ops

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type clusterRepository struct {
	db *sql.DB
}

func newClusterRepository(db *sql.DB) *clusterRepository {
	if db == nil {
		return nil
	}
	return &clusterRepository{db: db}
}

func (r *clusterRepository) List(domain string) ([]Cluster, error) {
	if r == nil || r.db == nil {
		return nil, fmt.Errorf("cluster repository 未初始化")
	}
	domain = strings.TrimSpace(strings.ToLower(domain))
	query := `
		SELECT
			a.id, a.name, a.domain, a.region, c.node_count, c.components_json,
			a.owner, a.status, COALESCE(json_extract(a.attributes_json, '$.description'), ''),
			a.created_at, a.updated_at, c.monitor_labels, c.vm_url_ref, c.metrics_base_url,
			c.jmx_url, c.fi_manager_url, c.gbase_dsn_ref, c.credentials_ref
		FROM clusters c
		JOIN assets a ON a.id = c.asset_id
		WHERE a.deleted_at = 0`
	args := []interface{}{}
	if domain != "" {
		query += ` AND a.domain = ?`
		args = append(args, domain)
	}
	query += ` ORDER BY a.updated_at DESC, a.name ASC`

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Cluster
	for rows.Next() {
		c, err := scanCluster(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []Cluster{}
	}
	return out, nil
}

func (r *clusterRepository) Get(id string) (Cluster, error) {
	if r == nil || r.db == nil {
		return Cluster{}, fmt.Errorf("cluster repository 未初始化")
	}
	row := r.db.QueryRow(`
		SELECT
			a.id, a.name, a.domain, a.region, c.node_count, c.components_json,
			a.owner, a.status, COALESCE(json_extract(a.attributes_json, '$.description'), ''),
			a.created_at, a.updated_at, c.monitor_labels, c.vm_url_ref, c.metrics_base_url,
			c.jmx_url, c.fi_manager_url, c.gbase_dsn_ref, c.credentials_ref
		FROM clusters c
		JOIN assets a ON a.id = c.asset_id
		WHERE a.deleted_at = 0 AND a.id = ?
	`, strings.TrimSpace(id))
	c, err := scanCluster(row)
	if err == sql.ErrNoRows {
		return Cluster{}, fmt.Errorf("集群不存在: %s", id)
	}
	return c, err
}

func (r *clusterRepository) FindByDomainName(domain, name string) (Cluster, bool, error) {
	if r == nil || r.db == nil {
		return Cluster{}, false, fmt.Errorf("cluster repository 未初始化")
	}
	row := r.db.QueryRow(`
		SELECT
			a.id, a.name, a.domain, a.region, c.node_count, c.components_json,
			a.owner, a.status, COALESCE(json_extract(a.attributes_json, '$.description'), ''),
			a.created_at, a.updated_at, c.monitor_labels, c.vm_url_ref, c.metrics_base_url,
			c.jmx_url, c.fi_manager_url, c.gbase_dsn_ref, c.credentials_ref
		FROM clusters c
		JOIN assets a ON a.id = c.asset_id
		WHERE a.deleted_at = 0 AND a.domain = ? AND lower(a.name) = lower(?)
	`, strings.TrimSpace(strings.ToLower(domain)), strings.TrimSpace(name))
	c, err := scanCluster(row)
	if err == sql.ErrNoRows {
		return Cluster{}, false, nil
	}
	if err != nil {
		return Cluster{}, false, err
	}
	return c, true, nil
}

func (r *clusterRepository) Create(c Cluster) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cluster repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := insertClusterTx(tx, c); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *clusterRepository) Upsert(c Cluster) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cluster repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := upsertClusterTx(tx, c); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *clusterRepository) Update(c Cluster) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cluster repository 未初始化")
	}
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`UPDATE assets SET name = ?, domain = ?, owner = ?, region = ?, status = ?, attributes_json = ?, updated_at = ? WHERE id = ? AND deleted_at = 0`,
		c.Name, c.Domain, c.Owner, c.Region, c.Status, clusterAttributesJSON(c), c.UpdatedAtMs, c.ID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("集群不存在: %s", c.ID)
	}
	if _, err := tx.Exec(`
		UPDATE clusters
		SET node_count = ?, components_json = ?, monitor_labels = ?, vm_url_ref = ?, metrics_base_url = ?,
		    jmx_url = ?, fi_manager_url = ?, gbase_dsn_ref = ?, credentials_ref = ?, updated_at = ?
		WHERE id = ?
	`, c.NodeCount, componentsJSON(c.Components), c.MonitorLabels, c.VMUrlRef, c.MetricsBaseUrl, c.JMXUrl, c.FIManagerUrl, c.GBaseDsnRef, c.CredentialsRef, c.UpdatedAtMs, c.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *clusterRepository) Delete(id string, now int64) error {
	if r == nil || r.db == nil {
		return fmt.Errorf("cluster repository 未初始化")
	}
	res, err := r.db.Exec(`UPDATE assets SET deleted_at = ?, updated_at = ? WHERE id = ? AND deleted_at = 0`, now, now, strings.TrimSpace(id))
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("集群不存在: %s", id)
	}
	return nil
}

func (r *clusterRepository) ImportJSON(storePath string) (int, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("cluster repository 未初始化")
	}
	if strings.TrimSpace(storePath) == "" {
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
	if len(store.Clusters) == 0 {
		return 0, backupJSONStore(storePath)
	}

	tx, err := r.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	imported := 0
	for _, c := range store.Clusters {
		c = normalizeClusterForStorage(c)
		if c.ID == "" {
			continue
		}
		if err := upsertClusterTx(tx, c); err != nil {
			return imported, err
		}
		imported++
	}
	if err := tx.Commit(); err != nil {
		return imported, err
	}
	if imported > 0 {
		if err := backupJSONStore(storePath); err != nil {
			return imported, err
		}
	}
	return imported, nil
}

func insertClusterTx(tx *sql.Tx, c Cluster) error {
	_, err := tx.Exec(`INSERT INTO assets (id, type, name, domain, owner, region, status, attributes_json, created_at, updated_at) VALUES (?, 'cluster', ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Domain, c.Owner, c.Region, c.Status, clusterAttributesJSON(c), c.CreatedAtMs, c.UpdatedAtMs)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO clusters (
			id, asset_id, node_count, components_json, monitor_labels, vm_url_ref, metrics_base_url,
			jmx_url, fi_manager_url, gbase_dsn_ref, credentials_ref, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, c.ID, c.ID, c.NodeCount, componentsJSON(c.Components), c.MonitorLabels, c.VMUrlRef, c.MetricsBaseUrl, c.JMXUrl, c.FIManagerUrl, c.GBaseDsnRef, c.CredentialsRef, c.CreatedAtMs, c.UpdatedAtMs)
	return err
}

func upsertClusterTx(tx *sql.Tx, c Cluster) error {
	if c.CreatedAtMs == 0 {
		c.CreatedAtMs = nowMs()
	}
	if c.UpdatedAtMs == 0 {
		c.UpdatedAtMs = c.CreatedAtMs
	}
	_, err := tx.Exec(`
		INSERT INTO assets (id, type, name, domain, owner, region, status, attributes_json, deleted_at, created_at, updated_at)
		VALUES (?, 'cluster', ?, ?, ?, ?, ?, ?, 0, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			type = 'cluster',
			name = excluded.name,
			domain = excluded.domain,
			owner = excluded.owner,
			region = excluded.region,
			status = excluded.status,
			attributes_json = excluded.attributes_json,
			deleted_at = 0,
			updated_at = excluded.updated_at
	`, c.ID, c.Name, c.Domain, c.Owner, c.Region, c.Status, clusterAttributesJSON(c), c.CreatedAtMs, c.UpdatedAtMs)
	if err != nil {
		return err
	}
	_, err = tx.Exec(`
		INSERT INTO clusters (
			id, asset_id, node_count, components_json, monitor_labels, vm_url_ref, metrics_base_url,
			jmx_url, fi_manager_url, gbase_dsn_ref, credentials_ref, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			node_count = excluded.node_count,
			components_json = excluded.components_json,
			monitor_labels = excluded.monitor_labels,
			vm_url_ref = excluded.vm_url_ref,
			metrics_base_url = excluded.metrics_base_url,
			jmx_url = excluded.jmx_url,
			fi_manager_url = excluded.fi_manager_url,
			gbase_dsn_ref = excluded.gbase_dsn_ref,
			credentials_ref = excluded.credentials_ref,
			updated_at = excluded.updated_at
	`, c.ID, c.ID, c.NodeCount, componentsJSON(c.Components), c.MonitorLabels, c.VMUrlRef, c.MetricsBaseUrl, c.JMXUrl, c.FIManagerUrl, c.GBaseDsnRef, c.CredentialsRef, c.CreatedAtMs, c.UpdatedAtMs)
	return err
}

type clusterScanner interface {
	Scan(dest ...interface{}) error
}

func scanCluster(row clusterScanner) (Cluster, error) {
	var c Cluster
	var components string
	if err := row.Scan(
		&c.ID, &c.Name, &c.Domain, &c.Region, &c.NodeCount, &components,
		&c.Owner, &c.Status, &c.Description, &c.CreatedAtMs, &c.UpdatedAtMs,
		&c.MonitorLabels, &c.VMUrlRef, &c.MetricsBaseUrl, &c.JMXUrl, &c.FIManagerUrl,
		&c.GBaseDsnRef, &c.CredentialsRef,
	); err != nil {
		return Cluster{}, err
	}
	_ = json.Unmarshal([]byte(components), &c.Components)
	if c.Components == nil {
		c.Components = []string{}
	}
	return normalizeClusterForStorage(c), nil
}

func normalizeClusterForStorage(c Cluster) Cluster {
	c.ID = strings.TrimSpace(c.ID)
	c.Name = strings.TrimSpace(c.Name)
	c.Domain = strings.TrimSpace(strings.ToLower(c.Domain))
	c.Region = strings.TrimSpace(c.Region)
	c.Components = normalizeComponents(c.Components)
	c.Owner = strings.TrimSpace(c.Owner)
	c.Status = strings.TrimSpace(strings.ToLower(c.Status))
	if c.Status == "" {
		c.Status = "unknown"
	}
	c.Description = strings.TrimSpace(c.Description)
	c.MonitorLabels = strings.TrimSpace(c.MonitorLabels)
	if c.MonitorLabels != "" {
		if normalized, err := NormalizeMonitorLabels(c.MonitorLabels); err == nil {
			c.MonitorLabels = normalized
		}
	}
	c.VMUrlRef = strings.TrimSpace(c.VMUrlRef)
	c.MetricsBaseUrl = normalizeMetricsBaseURL(c.MetricsBaseUrl)
	c.JMXUrl = strings.TrimSpace(c.JMXUrl)
	c.FIManagerUrl = strings.TrimSpace(c.FIManagerUrl)
	c.GBaseDsnRef = strings.TrimSpace(c.GBaseDsnRef)
	c.CredentialsRef = strings.TrimSpace(c.CredentialsRef)
	if c.CreatedAtMs == 0 {
		c.CreatedAtMs = nowMs()
	}
	if c.UpdatedAtMs == 0 {
		c.UpdatedAtMs = c.CreatedAtMs
	}
	return c
}

func clusterAttributesJSON(c Cluster) string {
	attrs := map[string]string{
		"description": c.Description,
	}
	b, err := json.Marshal(attrs)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func componentsJSON(parts []string) string {
	b, err := json.Marshal(normalizeComponents(parts))
	if err != nil {
		return "[]"
	}
	return string(b)
}

func backupJSONStore(path string) error {
	backupPath := fmt.Sprintf("%s.bak.%d", path, time.Now().UnixMilli())
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return err
	}
	return os.Rename(path, backupPath)
}
