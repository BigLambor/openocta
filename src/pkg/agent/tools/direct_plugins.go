package tools

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
)

// GBaseSlowSqlTool queries slow SQL logs from GBase database.
type GBaseSlowSqlTool struct{}

type gbaseSlowSQLEvidence struct {
	Type         string                   `json:"type"`
	Status       string                   `json:"status"`
	SlowSQLCount int                      `json:"slowSqlCount"`
	Rows         []map[string]interface{} `json:"rows,omitempty"`
	Error        string                   `json:"error,omitempty"`
}

// Name returns the tool name.
func (GBaseSlowSqlTool) Name() string {
	return "query_gbase_slow_sql"
}

// Description returns the tool description.
func (GBaseSlowSqlTool) Description() string {
	return "Query slow SQL logs from GBase database. Requires GBASE_DSN (or db_url). Returns SQL text, execution duration, and timestamp."
}

// Schema returns the parameters schema.
func (GBaseSlowSqlTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"limit": map[string]interface{}{
				"type":        "number",
				"description": "Optional limit of slow SQL logs to return. Defaults to 5.",
			},
			"db_url": map[string]interface{}{
				"type":        "string",
				"description": "Optional database DSN. Defaults to env GBASE_DSN.",
			},
		},
	}
}

// Execute runs the slow SQL query against a real DSN when configured.
func (GBaseSlowSqlTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := params["limit"].(int); ok {
		limit = l
	}
	if limit < 1 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	dsn, _ := params["db_url"].(string)
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		opsCtx := ParseOpsContext(ctx)
		if opsCtx != nil && opsCtx.ClusterID != "" && opsCtx.ClusterID != "all" {
			if GetClusterConfig != nil {
				c, err := GetClusterConfig(opsCtx.ClusterID)
				if err == nil && c.GBaseDsnRef != "" {
					dsn = c.GBaseDsnRef
				}
			}
		}
	}
	if dsn == "" {
		dsn = strings.TrimSpace(os.Getenv("GBASE_DSN"))
	}
	if dsn == "" {
		return gbaseSlowSQLToolResult(false, nil, "GBASE_DSN 未配置：请在环境变量中设置 GBase 连接串，或通过 db_url 参数传入。不会返回模拟慢 SQL 数据。"), nil
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return gbaseSlowSQLToolResult(false, nil, fmt.Sprintf("无法连接 GBase: %v", err)), nil
	}
	defer db.Close()

	qctx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()
	if err := db.PingContext(qctx); err != nil {
		return gbaseSlowSQLToolResult(false, nil, fmt.Sprintf("GBase 连接失败: %v", err)), nil
	}

	// Generic slow-query probe; adjust table/view via GBASE_SLOW_SQL_QUERY if needed.
	querySQL := strings.TrimSpace(os.Getenv("GBASE_SLOW_SQL_QUERY"))
	if querySQL == "" {
		querySQL = `SELECT sql_text, exec_time_sec, start_time, client_ip
FROM information_schema.slow_query_log
ORDER BY exec_time_sec DESC
LIMIT ?`
	}

	rows, err := db.QueryContext(qctx, querySQL, limit)
	if err != nil {
		return gbaseSlowSQLToolResult(false, nil, fmt.Sprintf("慢 SQL 查询失败: %v（可通过 GBASE_SLOW_SQL_QUERY 自定义 SQL）", err)), nil
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	out := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := map[string]interface{}{}
		for i, c := range cols {
			switch v := vals[i].(type) {
			case []byte:
				row[c] = string(v)
			default:
				row[c] = v
			}
		}
		out = append(out, row)
	}
	return gbaseSlowSQLToolResult(true, out, ""), nil
}

func gbaseSlowSQLToolResult(success bool, rows []map[string]interface{}, errText string) *tool.ToolResult {
	status := "healthy"
	if !success {
		status = "critical"
	} else if len(rows) > 0 {
		status = "warning"
	}
	evidence := gbaseSlowSQLEvidence{
		Type:         "gbase_sql",
		Status:       status,
		SlowSQLCount: len(rows),
		Rows:         rows,
		Error:        strings.TrimSpace(errText),
	}
	out, err := json.Marshal(evidence)
	if err != nil {
		out = []byte(errText)
	}
	return &tool.ToolResult{
		Success: success,
		Output:  string(out),
		Data:    evidence,
	}
}

var _ tool.Tool = GBaseSlowSqlTool{}

// GovernanceLineageTool queries metadata lineage and quality alerts.
type GovernanceLineageTool struct{}

// Name returns the tool name.
func (GovernanceLineageTool) Name() string {
	return "query_governance_lineage"
}

// Description returns the tool description.
func (GovernanceLineageTool) Description() string {
	return "Query data lineage and quality alerts from the governance platform API. Requires GOVERNANCE_API_URL (or api_url)."
}

// Schema returns the parameters schema.
func (GovernanceLineageTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"domain": map[string]interface{}{
				"type":        "string",
				"description": "Optional tech domain or project name to filter by.",
			},
			"api_url": map[string]interface{}{
				"type":        "string",
				"description": "Optional API base URL. Defaults to env GOVERNANCE_API_URL.",
			},
		},
	}
}

// Execute runs the governance HTTP query when API URL is configured.
func (GovernanceLineageTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	apiURL, _ := params["api_url"].(string)
	apiURL = strings.TrimSpace(apiURL)
	if apiURL == "" {
		apiURL = strings.TrimSpace(os.Getenv("GOVERNANCE_API_URL"))
	}
	if apiURL == "" {
		return &tool.ToolResult{
			Success: false,
			Output:  "GOVERNANCE_API_URL 未配置：请设置治理平台 API 地址，或通过 api_url 参数传入。不会返回模拟血缘/质量数据。",
		}, nil
	}

	domain, _ := params["domain"].(string)
	path := strings.TrimSpace(os.Getenv("GOVERNANCE_LINEAGE_PATH"))
	if path == "" {
		path = "/api/v1/quality/alerts"
	}
	target := strings.TrimSuffix(apiURL, "/") + path
	if domain != "" {
		target += "?domain=" + urlQueryEscape(domain)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if token := strings.TrimSpace(os.Getenv("GOVERNANCE_API_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("治理 API 请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &tool.ToolResult{
			Success: false,
			Output:  fmt.Sprintf("治理 API 返回 %d: %s", resp.StatusCode, truncateToolOutput(string(body), 500)),
		}, nil
	}
	return &tool.ToolResult{Success: true, Output: string(body)}, nil
}

func urlQueryEscape(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "%20")
}

func truncateToolOutput(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

var _ tool.Tool = GovernanceLineageTool{}
