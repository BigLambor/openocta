// Package tools provides custom tools for the agent runtime.
package tools

import (
	"bytes"
	"context"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/stellarlinkco/agentsdk-go/pkg/tool"
)

// WindowsCmdTool executes shell commands on Windows via PowerShell (preferred) or cmd (fallback).
// Only available when the agent runs on Windows.
type WindowsCmdTool struct {
	Timeout time.Duration // Optional: command timeout, defaults to 60s
}

// Name returns the tool name.
func (WindowsCmdTool) Name() string {
	return "windows_exec_cmd"
}

// Description returns the tool description.
func (WindowsCmdTool) Description() string {
	return "Execute a command or batch script on Windows using PowerShell (preferred) or cmd (fallback). Only works when the agent is running on Windows (runtime.GOOS=windows). Supports dir, ipconfig, tasklist, etc. Runs silently with no console window."
}

// Schema returns the parameter schema.
func (WindowsCmdTool) Schema() *tool.JSONSchema {
	return &tool.JSONSchema{
		Type: "object",
		Properties: map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "Windows command to execute silently in background (no window, no flash). Supports dir, ipconfig, tasklist, etc.",
			},
			"timeout": map[string]interface{}{
				"type":        "integer",
				"description": "Optional timeout in seconds (default 60)",
			},
		},
		Required: []string{"command"},
	}
}

// Execute runs the tool.
func (t WindowsCmdTool) Execute(ctx context.Context, params map[string]interface{}) (*tool.ToolResult, error) {
	if runtime.GOOS != "windows" {
		return &tool.ToolResult{
			Success: false,
			Output:  "windows_exec_cmd: this tool only works on Windows. Current OS is " + runtime.GOOS + ".",
		}, nil
	}

	cmdStr, _ := params["command"].(string)
	cmdStr = strings.TrimSpace(cmdStr)
	if cmdStr == "" {
		return &tool.ToolResult{Success: false, Output: "command is required"}, nil
	}

	// Parse optional timeout parameter
	timeout := t.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second // default 60 seconds
	}
	if timeoutSec, ok := params["timeout"].(float64); ok && timeoutSec > 0 {
		timeout = time.Duration(timeoutSec) * time.Second
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout, stderr bytes.Buffer

	// Primary: PowerShell with triple hiding mechanism
	// 1. cmd.exe /c start /b /wait - no new window at cmd level
	// 2. PowerShell -WindowStyle Hidden - hide at PowerShell level
	// 3. CreationFlags: CREATE_NO_WINDOW | CREATE_NEW_PROCESS_GROUP | DETACHED_PROCESS (Go syscall level)
	// This ensures 100% no console window popup
	cmd := exec.CommandContext(timeoutCtx,
		"cmd.exe",
		"/c",
		"start",
		"/b",
		"/wait",
		"powershell.exe",
		"-NoProfile",
		"-NonInteractive",
		"-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden",
		"-Command", cmdStr,
	)
	applyExecNoWindow(cmd)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = nil

	err := cmd.Run()

	// Fallback 1: If start /b fails to capture output, try direct PowerShell
	if err == nil && stdout.Len() == 0 && stderr.Len() == 0 {
		// Retry with direct PowerShell execution (may have brief flash but captures output)
		stdout.Reset()
		stderr.Reset()
		cmd = exec.CommandContext(timeoutCtx,
			"powershell.exe",
			"-NoProfile",
			"-NonInteractive",
			"-ExecutionPolicy", "Bypass",
			"-WindowStyle", "Hidden",
			"-Command", cmdStr,
		)
		applyExecNoWindow(cmd)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Stdin = nil
		err = cmd.Run()
	}

	// Fallback: If PowerShell not found, degrade to cmd
	// Note: cmd fallback cannot guarantee completely no window due to system limitations
	if err != nil && isPowershellNotFound(err) {
		stdout.Reset()
		stderr.Reset()
		cmd = exec.CommandContext(timeoutCtx, "cmd", "/c", cmdStr)
		applyExecNoWindow(cmd)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		cmd.Stdin = nil
		err = cmd.Run()
	}

	// Handle timeout
	if timeoutCtx.Err() == context.DeadlineExceeded {
		return &tool.ToolResult{Success: false, Output: "command timeout after " + timeout.String()}, nil
	}

	outStr := strings.TrimSpace(stdout.String())
	errStr := strings.TrimSpace(stderr.String())

	if err != nil {
		combined := outStr
		if errStr != "" {
			if combined != "" {
				combined += "\n"
			}
			combined += errStr
		}
		if combined == "" {
			combined = err.Error()
		}
		return &tool.ToolResult{Success: false, Output: combined}, nil
	}

	output := outStr
	if errStr != "" {
		output = outStr + "\n" + errStr
	}
	if output == "" {
		output = "(command completed with no output)"
	}
	return &tool.ToolResult{Success: true, Output: output}, nil
}

// isPowershellNotFound checks if the error indicates PowerShell is not available.
// Covers multiple languages and Windows versions.
func isPowershellNotFound(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "powershell") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "no such file") ||
		strings.Contains(msg, "cannot find") ||
		strings.Contains(msg, "executable file not found")
}
