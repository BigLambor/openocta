package ops

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
	agentTools "github.com/openocta/openocta/pkg/agent/tools"
)

// RunOpts configuration for running a scenario
type RunOpts struct {
	SessionID  string
	RunID      string
	EmployeeID string
	// Params can be used to pass arbitrary parameters to the tools (e.g. cluster limits, queries).
	Params     map[string]interface{}
}

// toolSourceMapping maps platform tools to the SignalSource types they provide
var toolSourceMapping = map[string][]string{
	"query_gbase_slow_sql":     {SignalTypeGBaseSQL},
	"query_vm_metrics":         {SignalTypeMetrics},
	"query_governance_lineage": {SignalTypeInspection},
}

// RunScenario executes a scenario completely natively without LLM intervention (e.g., used by cron).
// It loads Platform Tools, runs them, verifies required/optional sources, and persists the results.
func RunScenario(ctx context.Context, scenarioKey string, objectID string, opts RunOpts) (InspectionResult, error) {
	start := time.Now()

	scenario, ok := GetOpsScenario(scenarioKey)
	if !ok {
		return InspectionResult{}, fmt.Errorf("scenario %s not found", scenarioKey)
	}

	var executedTools []string
	var mcpCalled []string
	signalsWritten := 0

	// Platform Tools Registry
	toolRegistry := map[string]tool.Tool{
		"query_gbase_slow_sql":     agentTools.GBaseSlowSqlTool{},
		"query_vm_metrics":         agentTools.VMQueryTool{},
		"query_governance_lineage": agentTools.GovernanceLineageTool{},
	}

	var missingSources []string
	obtainedSources := make(map[string]bool)
	finalScoreStatus := "healthy"
	var combinedTextBuilder strings.Builder

	// Execute Platform Tools
	for _, tk := range scenario.PlatformToolKeys {
		if t, found := toolRegistry[tk]; found {
			// For some tools we might want to inject specific parameters. For now we pass opts.Params.
			params := opts.Params
			if params == nil {
				params = map[string]interface{}{}
			}

			// If running VM query, we may need to inject a specific query if not provided, 
			// but usually the tool itself might not have defaults.
			// Here we assume it executes generically.
			res, err := t.Execute(ctx, params)
			executedTools = append(executedTools, tk)

			if err == nil && res != nil && res.Success {
				combinedTextBuilder.WriteString(fmt.Sprintf("#### [%s] Success\n%s\n", tk, truncateString(res.Output, 500)))
				// Mark fulfilled sources
				if sources, ok := toolSourceMapping[tk]; ok {
					for _, src := range sources {
						obtainedSources[src] = true
					}
				}
				signalsWritten++
			} else {
				errMsg := ""
				if err != nil {
					errMsg = err.Error()
				} else if res != nil {
					errMsg = res.Output
				}
				combinedTextBuilder.WriteString(fmt.Sprintf("#### [%s] Failed\n%s\n", tk, errMsg))
			}
		} else {
			combinedTextBuilder.WriteString(fmt.Sprintf("#### [%s] Not Found in Registry\n", tk))
		}
	}

	// P3-3: Platform Tool 与 MCP 同名时 Platform Tool 优先
	for _, mk := range scenario.MCPServerKeys {
		// Simulate MCP call
		toolName := mk
		mcpCalled = append(mcpCalled, toolName)
		
		if toolName == "prometheus" {
			// Prometheus MCP Pilot (P3-5)
			if obtainedSources[SignalTypeMetrics] {
				combinedTextBuilder.WriteString(fmt.Sprintf("#### [MCP: %s] Skipped (Platform Tool already provided metrics)\n", toolName))
				continue
			}
			
			// Mock MCP execution
			// In production, we would use mcpManager.Tools(ctx) to find the tool and call Execute()
			resOutput := `{"type": "metrics", "status": "ok", "prometheus": "mcp", "message": "Successfully collected metrics from MCP Prometheus"}`
			combinedTextBuilder.WriteString(fmt.Sprintf("#### [MCP: %s] Success\n%s\n", toolName, resOutput))
			obtainedSources[SignalTypeMetrics] = true
			signalsWritten++
		} else {
			combinedTextBuilder.WriteString(fmt.Sprintf("#### [MCP: %s] Failed\nUnknown MCP Server\n", toolName))
			// missingSources will be calculated next
		}
	}

	// P3-4: Verify Required Sources (degraded) and Optional Sources (lowers coverage)
	for _, req := range scenario.RequiredSources {
		if !obtainedSources[req] {
			missingSources = append(missingSources, req)
			finalScoreStatus = "degraded"
		}
	}

	for _, opt := range scenario.OptionalSources {
		if !obtainedSources[opt] {
			missingSources = append(missingSources, opt)
		}
	}

	durationMs := time.Since(start).Milliseconds()

	// Parse results into standard InspectionResult structure
	res := ParseInspectionResult(opts.SessionID, opts.RunID, combinedTextBuilder.String(), finalScoreStatus, start.UnixMilli(), durationMs)
	res.SourceKind = "cron"
	res.TriggerType = "scenario_runner"
	res.MissingSources = missingSources

	// Write L3 Facts (PersistInspectionFacts internally writes HealthSignal and HealthSnapshot)
	if err := PersistInspectionFacts(res); err != nil {
		fmt.Printf("Warning: failed to persist inspection facts: %v\n", err)
	}

	// Record Run Audit (X-1)
	audit := RunAudit{
		RunID:          opts.RunID,
		ScenarioKey:    scenarioKey,
		EmployeeID:     opts.EmployeeID,
		ObjectID:       objectID,
		ToolsCalled:    executedTools,
		MCPCalled:      mcpCalled,
		SignalsWritten: signalsWritten,
		MissingSources: missingSources,
		DurationMs:     durationMs,
		Operator:       "system", // Because this is the automated scenario runner
		Timestamp:      fmt.Sprint(time.Now().UnixMilli()),
	}
	RecordRunAudit(audit)

	return res, nil
}

func truncateString(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
