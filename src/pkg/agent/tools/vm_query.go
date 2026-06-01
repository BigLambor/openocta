package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
)

// VMQueryTool queries VictoriaMetrics or Prometheus using standard PromQL.
type VMQueryTool struct{}

// Name returns the tool name.
func (VMQueryTool) Name() string {
	return "query_vm_metrics"
}

// Description returns the tool description.
func (VMQueryTool) Description() string {
	return "Query VictoriaMetrics or Prometheus using PromQL. Supports instant and range queries."
}

// Schema returns the parameters schema.
func (VMQueryTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"query": map[string]interface{}{
				"type":        "string",
				"description": "The PromQL query string, e.g. sum(rate(node_cpu_seconds_total{mode!='idle'}[5m])) by (instance)",
			},
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Optional VictoriaMetrics/Prometheus API base URL. Defaults to env VICTORIAMETRICS_URL or PROMETHEUS_URL.",
			},
			"time": map[string]interface{}{
				"type":        "string",
				"description": "Optional evaluation timestamp (RFC3339 or Unix timestamp). For instant queries.",
			},
			"start": map[string]interface{}{
				"type":        "string",
				"description": "Optional start time (RFC3339 or Unix timestamp) to trigger a range query.",
			},
			"end": map[string]interface{}{
				"type":        "string",
				"description": "Optional end time (RFC3339 or Unix timestamp) for range query.",
			},
			"step": map[string]interface{}{
				"type":        "string",
				"description": "Optional query resolution step width (e.g. 15s, 1m) for range query.",
			},
		},
		Required: []string{"query"},
	}
}

// Execute runs the PromQL query.
func (VMQueryTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	query, _ := params["query"].(string)
	if query == "" {
		return &tool.ToolResult{Success: false, Output: "query is required"}, nil
	}

	// Resolve target URL
	targetURL, _ := params["url"].(string)
	targetURL = strings.TrimSpace(targetURL)
	if targetURL == "" {
		targetURL = os.Getenv("VICTORIAMETRICS_URL")
	}
	if targetURL == "" {
		targetURL = os.Getenv("PROMETHEUS_URL")
	}
	if targetURL == "" {
		// Default VictoriaMetrics single-node address
		targetURL = "http://localhost:8428"
	}

	// Ensure no trailing slash
	targetURL = strings.TrimSuffix(targetURL, "/")

	// Determine query endpoint
	isRange := params["start"] != nil && strings.TrimSpace(fmt.Sprint(params["start"])) != ""
	endpoint := "/api/v1/query"
	if isRange {
		endpoint = "/api/v1/query_range"
	}

	apiURL, err := url.Parse(targetURL + endpoint)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("invalid base url: %v", err)}, nil
	}

	// Build query parameters
	q := apiURL.Query()
	q.Set("query", query)

	if isRange {
		q.Set("start", strings.TrimSpace(fmt.Sprint(params["start"])))
		if params["end"] != nil {
			q.Set("end", strings.TrimSpace(fmt.Sprint(params["end"])))
		} else {
			q.Set("end", fmt.Sprint(time.Now().Unix()))
		}
		if params["step"] != nil {
			q.Set("step", strings.TrimSpace(fmt.Sprint(params["step"])))
		} else {
			q.Set("step", "1m")
		}
	} else {
		if params["time"] != nil {
			q.Set("time", strings.TrimSpace(fmt.Sprint(params["time"])))
		}
	}

	apiURL.RawQuery = q.Encode()

	// Make HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL.String(), nil)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("failed to build request: %v", err)}, nil
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("http request failed: %v", err)}, nil
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("failed to read response body: %v", err)}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return &tool.ToolResult{
			Success: false,
			Output:  fmt.Sprintf("API returned status %d: %s", resp.StatusCode, string(bodyBytes)),
		}, nil
	}

	return &tool.ToolResult{
		Success: true,
		Output:  string(bodyBytes),
	}, nil
}

var _ tool.Tool = VMQueryTool{}
