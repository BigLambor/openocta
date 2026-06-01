package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
)

// FIManagerTool queries FusionInsight Manager REST metrics API.
type FIManagerTool struct{}

func (FIManagerTool) Name() string {
	return "query_fi_manager_metrics"
}

func (FIManagerTool) Description() string {
	return "Query FusionInsight Manager health/metrics API. Requires FI_MANAGER_URL."
}

func (FIManagerTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Optional API URL. Defaults to env FI_MANAGER_URL.",
			},
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Optional path appended to base URL, e.g. /api/v1/clusters/health",
			},
		},
	}
}

func (FIManagerTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	base, _ := params["url"].(string)
	base = strings.TrimSpace(base)
	if base == "" {
		opsCtx := ParseOpsContext(ctx)
		if opsCtx != nil && opsCtx.ClusterID != "" && opsCtx.ClusterID != "all" {
			if GetClusterConfig != nil {
				c, err := GetClusterConfig(opsCtx.ClusterID)
				if err == nil && c.FIManagerUrl != "" {
					base = c.FIManagerUrl
				}
			}
		}
	}
	if base == "" {
		base = strings.TrimSpace(os.Getenv("FI_MANAGER_URL"))
	}
	if base == "" {
		return &tool.ToolResult{
			Success: false,
			Output:  "FI_MANAGER_URL 未配置：请设置 FusionInsight Manager API 基址，或通过 url 参数传入。",
		}, nil
	}
	apiPath, _ := params["path"].(string)
	apiPath = strings.TrimSpace(apiPath)
	if apiPath == "" {
		apiPath = strings.TrimSpace(os.Getenv("FI_MANAGER_HEALTH_PATH"))
	}
	if apiPath == "" {
		apiPath = "/api/v1/clusters/health"
	}
	target := strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(apiPath, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if token := strings.TrimSpace(os.Getenv("FI_MANAGER_TOKEN")); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("FI Manager 请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &tool.ToolResult{
			Success: false,
			Output:  fmt.Sprintf("FI Manager API %d: %s", resp.StatusCode, truncateToolOutput(string(body), 500)),
		}, nil
	}
	return &tool.ToolResult{Success: true, Output: string(body)}, nil
}

var _ tool.Tool = FIManagerTool{}
