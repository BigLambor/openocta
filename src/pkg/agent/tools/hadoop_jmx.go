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

// HadoopJMXTool queries Hadoop JMX HTTP endpoint (NameNode / ResourceManager).
type HadoopJMXTool struct{}

func (HadoopJMXTool) Name() string {
	return "query_hadoop_jmx"
}

func (HadoopJMXTool) Description() string {
	return "Query Hadoop JMX metrics via HTTP (NameNode, YARN RM, etc.). Requires HADOOP_JMX_URL."
}

func (HadoopJMXTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"url": map[string]interface{}{
				"type":        "string",
				"description": "Optional full JMX URL. Defaults to env HADOOP_JMX_URL.",
			},
			"qry": map[string]interface{}{
				"type":        "string",
				"description": "Optional JMX bean query, e.g. Hadoop:service=NameNode,name=NameNodeInfo",
			},
		},
	}
}

func (HadoopJMXTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	target, _ := params["url"].(string)
	target = strings.TrimSpace(target)
	if target == "" {
		opsCtx := ParseOpsContext(ctx)
		if opsCtx != nil && opsCtx.ClusterID != "" && opsCtx.ClusterID != "all" {
			if GetClusterConfig != nil {
				c, err := GetClusterConfig(opsCtx.ClusterID)
				if err == nil && c.JMXUrl != "" {
					target = c.JMXUrl
				}
			}
		}
	}
	if target == "" {
		target = strings.TrimSpace(os.Getenv("HADOOP_JMX_URL"))
	}
	if target == "" {
		return &tool.ToolResult{
			Success: false,
			Output:  "HADOOP_JMX_URL 未配置：请设置 NameNode/RM 的 JMX HTTP 地址（如 http://nn:50070/jmx），或通过 url 参数传入。",
		}, nil
	}
	if qry, _ := params["qry"].(string); strings.TrimSpace(qry) != "" && !strings.Contains(target, "qry=") {
		sep := "?"
		if strings.Contains(target, "?") {
			sep = "&"
		}
		target = target + sep + "qry=" + strings.TrimSpace(qry)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: fmt.Sprintf("JMX 请求失败: %v", err)}, nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &tool.ToolResult{Success: false, Output: err.Error()}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return &tool.ToolResult{
			Success: false,
			Output:  fmt.Sprintf("JMX API %d: %s", resp.StatusCode, truncateToolOutput(string(body), 500)),
		}, nil
	}
	return &tool.ToolResult{Success: true, Output: string(body)}, nil
}

var _ tool.Tool = HadoopJMXTool{}
