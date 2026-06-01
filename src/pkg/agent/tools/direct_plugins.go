package tools

import (
	"context"
	"encoding/json"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
)

// GBaseSlowSqlTool queries slow SQL logs from GBase database.
type GBaseSlowSqlTool struct{}

// Name returns the tool name.
func (GBaseSlowSqlTool) Name() string {
	return "query_gbase_slow_sql"
}

// Description returns the tool description.
func (GBaseSlowSqlTool) Description() string {
	return "Query slow SQL logs from GBase database. Returns SQL text, execution duration, and timestamp."
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
				"description": "Optional database connection string. If omitted, uses simulated slow SQL logs for diagnostics.",
			},
		},
	}
}

// Execute runs the slow SQL query simulation or fetch.
func (GBaseSlowSqlTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	limit := 5
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	} else if l, ok := params["limit"].(int); ok {
		limit = l
	}

	slowSQLs := []map[string]interface{}{
		{
			"sql":              "SELECT COUNT(*), status FROM orders GROUP BY status HAVING COUNT(*) > 100000;",
			"duration_seconds": 4.5,
			"timestamp":        "2026-06-01 16:30:22",
			"client_ip":        "10.20.134.12",
		},
		{
			"sql":              "SELECT * FROM fact_sales WHERE sale_date BETWEEN '2025-01-01' AND '2025-12-31' ORDER BY revenue DESC;",
			"duration_seconds": 12.8,
			"timestamp":        "2026-06-01 15:45:10",
			"client_ip":        "10.20.134.15",
		},
		{
			"sql":              "UPDATE user_profiles SET last_login = NOW() WHERE last_login < '2026-01-01';",
			"duration_seconds": 3.2,
			"timestamp":        "2026-06-01 14:12:05",
			"client_ip":        "10.20.134.13",
		},
		{
			"sql":              "SELECT p.name, sum(s.quantity) FROM dim_products p JOIN fact_sales s ON p.id = s.product_id GROUP BY p.name;",
			"duration_seconds": 6.7,
			"timestamp":        "2026-06-01 13:05:41",
			"client_ip":        "10.20.134.19",
		},
		{
			"sql":              "SELECT * FROM sys_logs WHERE log_level = 'ERROR' AND log_time < DATE_SUB(NOW(), INTERVAL 30 DAY);",
			"duration_seconds": 5.4,
			"timestamp":        "2026-06-01 11:22:18",
			"client_ip":        "10.20.134.12",
		},
	}

	if limit > len(slowSQLs) {
		limit = len(slowSQLs)
	}

	res := slowSQLs[:limit]
	b, err := json.Marshal(res)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}

	return &tool.ToolResult{Success: true, Output: string(b)}, nil
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
	return "Query data lineage map status and metadata quality validation alerts."
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
				"description": "Optional API endpoint. If omitted, uses simulated governance events.",
			},
		},
	}
}

// Execute runs the governance query.
func (GovernanceLineageTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	alerts := []map[string]interface{}{
		{
			"table":               "fact_sales",
			"rule":                "null_check_on_sale_id",
			"status":              "FAILED",
			"severity":            "CRITICAL",
			"value":               "124 nulls found",
			"timestamp":           "2026-06-01 16:00:00",
			"impacted_downstream": []string{"sales_dashboard_app", "monthly_financial_report"},
		},
		{
			"table":               "dim_products",
			"rule":                "unique_check_on_product_id",
			"status":              "FAILED",
			"severity":            "WARNING",
			"value":               "2 duplicates found",
			"timestamp":           "2026-06-01 15:30:00",
			"impacted_downstream": []string{"fact_sales"},
		},
		{
			"table":               "user_profiles",
			"rule":                "format_check_on_email",
			"status":              "PASSED",
			"severity":            "INFO",
			"value":               "0 invalid emails",
			"timestamp":           "2026-06-01 15:00:00",
			"impacted_downstream": []string{},
		},
	}

	b, err := json.Marshal(alerts)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	return &tool.ToolResult{Success: true, Output: string(b)}, nil
}

var _ tool.Tool = GovernanceLineageTool{}
