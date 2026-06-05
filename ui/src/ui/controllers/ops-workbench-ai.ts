import type { GatewayBrowserClient } from "../gateway.ts";
import { extractText } from "../chat/message-extract.ts";
import { canonicalGatewaySessionKey } from "../sessions/session-key-utils.js";
import { generateUUID } from "../uuid.ts";

export type WorkbenchAiMode = "root-cause" | "similar" | "action";

export type WorkbenchAiStatus = "idle" | "loading" | "streaming" | "done" | "error";

export type WorkbenchAiAlert = {
  id: string;
  title: string;
  severity: string;
  originalCount?: number;
  reducedTo?: number;
  rootCause?: string;
  impact?: string;
  analysisMarkdown?: string;
  domain?: string;
};

/**
 * Runs the workbench scenario AI analysis on a dedicated session so the
 * streaming output renders inside the workbench side panel without disturbing
 * the global Copilot (/message) session or its chat history.
 */
export type WorkbenchAiHost = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  chatModelRef: string | null;
  workbenchAiSessionKey: string | null;
  workbenchAiRunId: string | null;
  workbenchAiStream: string | null;
  workbenchAiResult: string | null;
  workbenchAiStatus: WorkbenchAiStatus;
  workbenchAiError: string | null;
  workbenchAiObjectId: string | null;
  workbenchAiMode: WorkbenchAiMode;
};

export type WorkbenchAiEventPayload = {
  runId: string;
  sessionKey: string;
  state: "delta" | "final" | "aborted" | "error";
  message?: unknown;
  errorMessage?: string;
};

function questionForMode(mode: WorkbenchAiMode, alert: WorkbenchAiAlert): string {
  switch (mode) {
    case "action":
      return `请基于当前告警组给出处置建议，区分只读排查、需要审批的变更和高风险操作，并按步骤说明：${alert.title}`;
    case "similar":
      return `请分析当前告警组的相似告警聚合逻辑，判断是否还有可以继续降噪或合并的相似告警：${alert.title}`;
    case "root-cause":
    default:
      return `请基于当前告警组分析根因候选，给出证据链、影响面判断和可验证的排查步骤：${alert.title}`;
  }
}

function buildSummary(alert: WorkbenchAiAlert): string {
  return [
    alert.title,
    typeof alert.originalCount === "number" ? `原始告警数: ${alert.originalCount}` : "",
    typeof alert.reducedTo === "number" ? `降噪后: ${alert.reducedTo}` : "",
    alert.rootCause ? `根因候选: ${alert.rootCause}` : "",
    alert.impact ? `影响范围: ${alert.impact}` : "",
    alert.analysisMarkdown ? `已有分析材料: ${alert.analysisMarkdown}` : "",
  ]
    .filter(Boolean)
    .join("\n");
}

export async function runWorkbenchAi(
  host: WorkbenchAiHost,
  params: { domain: string; mode: WorkbenchAiMode; alert: WorkbenchAiAlert; assistantTemplate: string },
): Promise<void> {
  const { domain, mode, alert, assistantTemplate } = params;
  if (!host.client || !host.connected) {
    host.workbenchAiSessionKey = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStream = null;
    host.workbenchAiResult = null;
    host.workbenchAiObjectId = alert.id;
    host.workbenchAiMode = mode;
    host.workbenchAiStatus = "error";
    host.workbenchAiError = "未连接到网关，无法发起 AI 分析。";
    return;
  }

  const sessionKey = `agent:main:ops:${domain || "hadoop"}:workbench-ai`;
  const runId = generateUUID();
  host.workbenchAiSessionKey = sessionKey;
  host.workbenchAiRunId = runId;
  host.workbenchAiMode = mode;
  host.workbenchAiObjectId = alert.id;
  host.workbenchAiStream = "";
  host.workbenchAiResult = null;
  host.workbenchAiError = null;
  host.workbenchAiStatus = "loading";

  try {
    await host.client.request("chat.send", {
      sessionKey: canonicalGatewaySessionKey(sessionKey),
      message: questionForMode(mode, alert),
      deliver: false,
      idempotencyKey: runId,
      modelRef: host.chatModelRef ?? undefined,
      assistantTemplate: assistantTemplate || undefined,
      context: {
        domain,
        capability: "observability-alert",
        workflowType: mode === "action" ? "incident" : "diagnosis",
        objectType: "alert",
        objectId: alert.id,
        severity: alert.severity,
        summary: buildSummary(alert),
      },
    });
  } catch (err) {
    if (host.workbenchAiRunId !== runId) {
      return;
    }
    host.workbenchAiStatus = "error";
    host.workbenchAiError = err instanceof Error ? err.message : String(err);
    host.workbenchAiRunId = null;
    host.workbenchAiStream = null;
  }
}

/**
 * Consumes a chat gateway event for the workbench AI run. Returns true when the
 * event belonged to the active workbench run (so the caller can skip the main
 * chat handling for it).
 */
export function handleWorkbenchAiEvent(host: WorkbenchAiHost, payload?: WorkbenchAiEventPayload): boolean {
  if (!payload || !host.workbenchAiRunId) {
    return false;
  }
  if (payload.runId !== host.workbenchAiRunId) {
    return false;
  }

  if (payload.state === "delta") {
    const next = extractText(payload.message);
    if (typeof next === "string") {
      const current = host.workbenchAiStream ?? "";
      if (!current || next.length >= current.length) {
        host.workbenchAiStream = next;
      }
    }
    host.workbenchAiStatus = "streaming";
  } else if (payload.state === "final") {
    const finalText = extractText(payload.message);
    const acc = host.workbenchAiStream ?? "";
    host.workbenchAiResult =
      typeof finalText === "string" && finalText.length >= acc.length ? finalText : acc;
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = "done";
  } else if (payload.state === "aborted") {
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = host.workbenchAiResult ? "done" : "idle";
  } else if (payload.state === "error") {
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = "error";
    host.workbenchAiError = payload.errorMessage ?? "AI 分析失败";
  }
  return true;
}
