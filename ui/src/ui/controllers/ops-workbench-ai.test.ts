import { describe, expect, it } from "vitest";
import {
  analysisContentFromAlert,
  applyWorkbenchAiResult,
  handleWorkbenchAiEvent,
  hasMeaningfulAlertAnalysis,
  resolveWorkbenchAlertDomain,
  syncWorkbenchAiFromCopilot,
  workbenchCapabilityForDomain,
  workbenchScenarioAiSessionKey,
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

  it("builds employee-scoped workbench scenario session keys", () => {
    expect(
      workbenchScenarioAiSessionKey({
        employeeId: "builtin-bch-inspect",
        scenarioId: "bch-flink-health",
        objectType: "flink_job",
        objectId: "job_tx_core",
      }),
    ).toBe("agent:main:employee:builtin-bch-inspect:workbench:bch-flink-health:flink_job:job_tx_core");
  });

  it("captures workbench AI events by session key when run id differs", () => {
    const host = buildHost({
      workbenchAiRunId: "local-run",
      workbenchAiSessionKey: "agent:main:employee:builtin-bch-inspect:workbench:bch-flink-health:flink_job:job_tx_core",
    });
    const captured = handleWorkbenchAiEvent(host, {
      runId: "gateway-run",
      sessionKey: "agent:main:employee:builtin-bch-inspect:workbench:bch-flink-health:flink_job:job_tx_core",
      state: "final",
      message: {
        role: "assistant",
        content: "## 结论\nFlink 作业运行健康。",
      },
    });
    expect(captured).toBe(true);
    expect(host.workbenchAiStatus).toBe("done");
    expect(host.workbenchAiResult).toContain("Flink 作业运行健康");
  });

  it("sends scenario object type and dedicated session to chat.send", async () => {
    const requests: Array<{ method: string; params: any }> = [];
    const host = buildHost({
      connected: true,
      client: {
        request: async (method: string, params: any) => {
          requests.push({ method, params });
          return {};
        },
      } as any,
    });
    const sessionKey = workbenchScenarioAiSessionKey({
      employeeId: "emp_bch_duty",
      scenarioId: "bch-flink-health",
      objectType: "flink_job",
      objectId: "job_tx_core",
    });
    const { runWorkbenchAi } = await import("./ops-workbench-ai.ts");
    await runWorkbenchAi(host, {
      domain: "hadoop",
      mode: "root-cause",
      assistantTemplate: "emp_bch_duty",
      sessionKey,
      alert: {
        id: "bch-flink-health:flink_job:job_tx_core:24h",
        title: "交易核心链路 · 实时问诊",
        severity: "info",
        domain: "hadoop",
        objectType: "flink_job",
        objectId: "job_tx_core",
      },
    });

    expect(requests[0]?.method).toBe("chat.send");
    expect(requests[0]?.params.sessionKey).toBe(sessionKey);
    expect(requests[0]?.params.context.objectType).toBe("flink_job");
    expect(requests[0]?.params.context.objectId).toBe("job_tx_core");
    expect(requests[0]?.params.deliver).toBe(false);
  });
});
