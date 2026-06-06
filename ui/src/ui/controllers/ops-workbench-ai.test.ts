import { describe, expect, it } from "vitest";
import {
  analysisContentFromAlert,
  applyWorkbenchAiResult,
  hasMeaningfulAlertAnalysis,
  resolveWorkbenchAlertDomain,
  syncWorkbenchAiFromCopilot,
  workbenchCapabilityForDomain,
  type WorkbenchAiHost,
} from "./ops-workbench-ai.ts";

function buildHost(overrides: Partial<WorkbenchAiHost> = {}): WorkbenchAiHost {
  return {
    client: null,
    connected: false,
    chatModelRef: null,
    workbenchAiSessionKey: null,
    workbenchAiRunId: null,
    workbenchAiStream: null,
    workbenchAiResult: null,
    workbenchAiStatus: "idle",
    workbenchAiError: null,
    workbenchAiObjectId: null,
    workbenchAiMode: "root-cause",
    ...overrides,
  };
}

describe("ops-workbench-ai helpers", () => {
  it("detects meaningful alert analysis and ignores placeholders", () => {
    expect(
      hasMeaningfulAlertAnalysis({
        id: "a1",
        title: "告警",
        severity: "critical",
        rootCause: "暂无根因分析",
      }),
    ).toBe(false);
    expect(
      hasMeaningfulAlertAnalysis({
        id: "a1",
        title: "告警",
        severity: "critical",
        analysisMarkdown: "## 根因\nGBase 连接未配置，导致 financial_report_daily SLA 逾期。",
      }),
    ).toBe(true);
  });

  it("prefers analysis markdown over placeholder root cause", () => {
    const content = analysisContentFromAlert({
      id: "a1",
      title: "告警",
      severity: "critical",
      rootCause: "暂无根因分析",
      analysisMarkdown: "## 结论\n根因为 GBASE_DSN 未配置。",
    });
    expect(content).toContain("GBASE_DSN");
  });

  it("syncs copilot final output back to workbench panel state", () => {
    const host = buildHost({
      workbenchAiCopilotRunId: "run-1",
      workbenchAiCopilotAlertId: "alert-1",
      workbenchAiCopilotMode: "root-cause",
    });
    const synced = syncWorkbenchAiFromCopilot(host, {
      runId: "run-1",
      sessionKey: "main",
      state: "final",
      message: {
        role: "assistant",
        content: "## 结论\n根因为 GBASE_DSN 未配置，导致 financial_report_daily 任务失败。",
      },
    });
    expect(synced).toBe(true);
    expect(host.workbenchAiStatus).toBe("done");
    expect(host.workbenchAiObjectId).toBe("alert-1");
    expect(host.workbenchAiResult).toContain("GBASE_DSN");
    expect(host.workbenchAiCopilotRunId).toBeNull();
  });

  it("resolves alert domain from record instead of all-domain filter", () => {
    expect(resolveWorkbenchAlertDomain("governance", "all")).toBe("governance");
    expect(resolveWorkbenchAlertDomain("dataapps", "all")).toBe("dataapps");
    expect(resolveWorkbenchAlertDomain("", "governance")).toBe("governance");
  });

  it("maps capability by technical domain for workbench AI", () => {
    expect(workbenchCapabilityForDomain("governance", "root-cause")).toBe("governance");
    expect(workbenchCapabilityForDomain("dataapps", "root-cause")).toBe("observability-alert");
    expect(workbenchCapabilityForDomain("hadoop", "root-cause")).toBe("observability-alert");
  });

  it("applies cached analysis into panel state", () => {
    const host = buildHost();
    applyWorkbenchAiResult(host, {
      alertId: "alert-2",
      mode: "root-cause",
      content: "## 根因\nNameNode heap 持续高于阈值。",
    });
    expect(host.workbenchAiStatus).toBe("done");
    expect(host.workbenchAiResult).toContain("NameNode");
  });
});
