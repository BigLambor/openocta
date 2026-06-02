package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// SyncError describes an error encountered during import of a single CMDB record.
type SyncError struct {
	RowIndex int    `json:"rowIndex"`
	Name     string `json:"name"`
	Error    string `json:"error"`
}

// CMDBSyncResult summarizes a CMDB import run (P1-2).
type CMDBSyncResult struct {
	Created  int         `json:"created"`
	Updated  int         `json:"updated"`
	Skipped  int         `json:"skipped"`
	Total    int         `json:"total"`
	Source   string      `json:"source"`
	Strategy string      `json:"strategy"`
	DryRun   bool        `json:"dryRun,omitempty"`
	Errors   []SyncError `json:"errors,omitempty"`
}

// CMDBMapping defines external field mappings for CMDB import.
type CMDBMapping struct {
	Name        string `json:"name"`
	Domain      string `json:"domain"`
	Region      string `json:"region"`
	NodeCount   string `json:"nodeCount"`
	Components  string `json:"components"`
	Owner       string `json:"owner"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

// DefaultMapping is the default fallback mapping configuration.
var DefaultMapping = CMDBMapping{
	Name:        "name",
	Domain:      "domain",
	Region:      "region",
	NodeCount:   "nodeCount",
	Components:  "components",
	Owner:       "owner",
	Status:      "status",
	Description: "description",
}

// LoadCMDBMappingFromEnv loads mapping properties from individual environment overrides or JSON.
func LoadCMDBMappingFromEnv() CMDBMapping {
	m := DefaultMapping
	if env := os.Getenv("OPS_CMDB_MAPPING"); env != "" {
		_ = json.Unmarshal([]byte(env), &m)
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_NAME"); v != "" {
		m.Name = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_DOMAIN"); v != "" {
		m.Domain = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_REGION"); v != "" {
		m.Region = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_NODE_COUNT"); v != "" {
		m.NodeCount = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_COMPONENTS"); v != "" {
		m.Components = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_OWNER"); v != "" {
		m.Owner = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_STATUS"); v != "" {
		m.Status = v
	}
	if v := os.Getenv("OPS_CMDB_MAPPING_DESCRIPTION"); v != "" {
		m.Description = v
	}
	return m
}

// CMDBClusterImport is one row from CMDB webhook or manual POST body.
type CMDBClusterImport struct {
	Name        string          `json:"name"`
	Domain      string          `json:"domain"`
	Region      string          `json:"region"`
	NodeCount   int             `json:"nodeCount"`
	Components  json.RawMessage `json:"components"`
	Owner       string          `json:"owner"`
	Status      string          `json:"status"`
	Description string          `json:"description"`
}

func parseComponentsField(raw json.RawMessage) []string {
	if len(raw) == 0 || string(raw) == "null" {
		return []string{}
	}
	var arr []string
	if err := json.Unmarshal(raw, &arr); err == nil {
		return normalizeComponents(arr)
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return normalizeComponents(strings.Split(single, ","))
	}
	return []string{}
}

func (r CMDBClusterImport) toCreate() (ClusterCreate, error) {
	name := strings.TrimSpace(r.Name)
	if name == "" {
		return ClusterCreate{}, fmt.Errorf("CMDB 行缺少 name")
	}
	return ClusterCreate{
		Name:        name,
		Domain:      r.Domain,
		Region:      r.Region,
		NodeCount:   r.NodeCount,
		Components:  parseComponentsField(r.Components),
		Owner:       r.Owner,
		Status:      r.Status,
		Description: r.Description,
	}, nil
}

func applyCMDBMapping(raw map[string]interface{}, mapping CMDBMapping) CMDBClusterImport {
	var row CMDBClusterImport

	getStr := func(fieldKey string, defaultKey string) string {
		key := defaultKey
		if fieldKey != "" {
			key = fieldKey
		}
		if val, ok := raw[key]; ok {
			if s, ok := val.(string); ok {
				return s
			}
			return fmt.Sprintf("%v", val)
		}
		return ""
	}

	row.Name = getStr(mapping.Name, "name")
	row.Domain = getStr(mapping.Domain, "domain")
	row.Region = getStr(mapping.Region, "region")
	row.Owner = getStr(mapping.Owner, "owner")
	row.Status = getStr(mapping.Status, "status")
	row.Description = getStr(mapping.Description, "description")

	nodeKey := "nodeCount"
	if mapping.NodeCount != "" {
		nodeKey = mapping.NodeCount
	}
	if val, ok := raw[nodeKey]; ok {
		switch v := val.(type) {
		case float64:
			row.NodeCount = int(v)
		case int:
			row.NodeCount = v
		case int64:
			row.NodeCount = int(v)
		case string:
			var n int
			fmt.Sscanf(v, "%d", &n)
			row.NodeCount = n
		}
	}

	compKey := "components"
	if mapping.Components != "" {
		compKey = mapping.Components
	}
	if val, ok := raw[compKey]; ok {
		if b, err := json.Marshal(val); err == nil {
			row.Components = b
		}
	}

	return row
}

// SyncClustersFromCMDB imports clusters from inline rows or OPS_CMDB_SYNC_URL (GET JSON).
// Supports upsert, dry-run, mark-inactive, delete strategies and custom mapping fields.
func SyncClustersFromCMDB(ctx context.Context, rawClusters []map[string]interface{}, strategy string, mapping *CMDBMapping) (CMDBSyncResult, error) {
	source := "body"
	var rawRows []map[string]interface{}

	if len(rawClusters) == 0 {
		url := strings.TrimSpace(os.Getenv("OPS_CMDB_SYNC_URL"))
		if url == "" {
			return CMDBSyncResult{}, fmt.Errorf("未配置 OPS_CMDB_SYNC_URL，且请求体未包含 clusters")
		}
		fetched, err := fetchCMDBRawRows(ctx, url)
		if err != nil {
			return CMDBSyncResult{}, err
		}
		rawRows = fetched
		source = "webhook"
	} else {
		rawRows = rawClusters
	}

	// Determine strategy
	strat := strings.TrimSpace(strings.ToLower(strategy))
	if strat == "" {
		strat = strings.TrimSpace(strings.ToLower(os.Getenv("OPS_CMDB_SYNC_STRATEGY")))
	}
	if strat != "upsert" && strat != "dry-run" && strat != "mark-inactive" && strat != "delete" {
		strat = "upsert"
	}

	// Apply mapping
	m := DefaultMapping
	if mapping != nil {
		m = *mapping
	} else {
		m = LoadCMDBMappingFromEnv()
	}

	mappedRows := make([]CMDBClusterImport, 0, len(rawRows))
	for _, raw := range rawRows {
		mappedRows = append(mappedRows, applyCMDBMapping(raw, m))
	}

	result := CMDBSyncResult{Source: source, Total: len(mappedRows), Strategy: strat, DryRun: strat == "dry-run"}
	processedKeys := make(map[string]bool)

	for idx, row := range mappedRows {
		in, err := row.toCreate()
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, SyncError{
				RowIndex: idx,
				Name:     row.Name,
				Error:    err.Error(),
			})
			continue
		}
		if err := validateClusterCreate(in); err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, SyncError{
				RowIndex: idx,
				Name:     row.Name,
				Error:    err.Error(),
			})
			continue
		}
		domainKey := strings.TrimSpace(strings.ToLower(row.Domain))
		nameKey := strings.TrimSpace(strings.ToLower(row.Name))
		processedKeys[domainKey+"/"+nameKey] = true

		if strat == "dry-run" {
			if _, ok := findClusterByDomainName(in.Domain, in.Name); ok {
				result.Updated++
			} else {
				result.Created++
			}
			continue
		}

		created, err := upsertCluster(in)
		if err != nil {
			result.Skipped++
			result.Errors = append(result.Errors, SyncError{
				RowIndex: idx,
				Name:     row.Name,
				Error:    err.Error(),
			})
			continue
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	// Apply post-sync strategy for local clusters not found in the CMDB feed
	if strat == "delete" || strat == "mark-inactive" {
		localClusters, err := ListClusters("")
		if err == nil {
			for _, lc := range localClusters {
				domainKey := strings.TrimSpace(strings.ToLower(lc.Domain))
				nameKey := strings.TrimSpace(strings.ToLower(lc.Name))
				if !processedKeys[domainKey+"/"+nameKey] {
					if strat == "delete" {
						_ = DeleteCluster(lc.ID)
					} else if strat == "mark-inactive" {
						statusInactive := "inactive"
						_, _ = PatchCluster(lc.ID, ClusterPatch{Status: &statusInactive})
					}
				}
			}
		}
	}

	return result, nil
}

func fetchCMDBRawRows(ctx context.Context, url string) ([]map[string]interface{}, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if token := strings.TrimSpace(os.Getenv("OPS_CMDB_SYNC_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("拉取 CMDB 失败: %w", err)
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("CMDB 返回 HTTP %d", res.StatusCode)
	}
	return decodeCMDBRawPayload(body)
}

func decodeCMDBRawPayload(body []byte) ([]map[string]interface{}, error) {
	var wrap struct {
		Clusters []map[string]interface{} `json:"clusters"`
	}
	if err := json.Unmarshal(body, &wrap); err == nil && len(wrap.Clusters) > 0 {
		return wrap.Clusters, nil
	}
	var rows []map[string]interface{}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, fmt.Errorf("无法解析 CMDB JSON: %w", err)
	}
	return rows, nil
}

func upsertCluster(in ClusterCreate) (created bool, err error) {
	domain, err := NormalizeDomain(in.Domain)
	if err != nil {
		return false, err
	}
	name := strings.TrimSpace(in.Name)

	existing, ok := findClusterByDomainName(domain, name)
	existingID := existing.ID

	if ok {
		comps := normalizeComponents(in.Components)
		patch := ClusterPatch{
			Region:         strPtr(strings.TrimSpace(in.Region)),
			NodeCount:      &in.NodeCount,
			Components:     &comps,
			Owner:          strPtr(strings.TrimSpace(in.Owner)),
			Description:    strPtr(strings.TrimSpace(in.Description)),
			MonitorLabels:  strPtr(strings.TrimSpace(in.MonitorLabels)),
			VMUrlRef:       strPtr(strings.TrimSpace(in.VMUrlRef)),
			MetricsBaseUrl: strPtr(strings.TrimSpace(in.MetricsBaseUrl)),
			JMXUrl:         strPtr(strings.TrimSpace(in.JMXUrl)),
			FIManagerUrl:   strPtr(strings.TrimSpace(in.FIManagerUrl)),
			GBaseDsnRef:    strPtr(strings.TrimSpace(in.GBaseDsnRef)),
			CredentialsRef: strPtr(strings.TrimSpace(in.CredentialsRef)),
		}
		if strings.TrimSpace(in.Status) != "" {
			status, serr := NormalizeStatus(in.Status)
			if serr != nil {
				return false, serr
			}
			patch.Status = &status
		}
		_, err := PatchCluster(existingID, patch)
		return false, err
	}
	_, err = CreateCluster(in)
	return true, err
}

func validateClusterCreate(in ClusterCreate) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("集群名称不能为空")
	}
	if _, err := NormalizeDomain(in.Domain); err != nil {
		return err
	}
	if _, err := NormalizeStatus(in.Status); err != nil {
		return err
	}
	if in.NodeCount < 0 {
		return fmt.Errorf("节点数不能为负数")
	}
	return nil
}

func findClusterByDomainName(domain, name string) (Cluster, bool) {
	domain = strings.TrimSpace(strings.ToLower(domain))
	name = strings.TrimSpace(name)

	serviceMu.RLock()
	defer serviceMu.RUnlock()
	for _, c := range clusters {
		if c.Domain == domain && strings.EqualFold(c.Name, name) {
			return c, true
		}
	}
	return Cluster{}, false
}
