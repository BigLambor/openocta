import type { GatewayBrowserClient } from "../gateway.ts";
import { effectiveOpsDomain, normalizeOpsDomain, type OpsDomainKey } from "../components/domain-filter.ts";
import { extractText } from "../chat/message-extract.ts";
import { canonicalGatewaySessionKey, gatewaySessionKeysEqual } from "../sessions/session-key-utils.js";
import { generateUUID } from "../uuid.ts";

type ConcreteOpsDomain = Exclude<OpsDomainKey, "all">;

/** 事件中心 AI 分析必须使用告警自身技术域，避免在「全部技术域」视图下误用 BCH 员工与会话。 */
export function resolveWorkbenchAlertDomain(
  alertDomain: string | undefined,
  selectedDomain: string,
): ConcreteOpsDomain {
  const fromAlert = normalizeOpsDomain(alertDomain);
  if (fromAlert !== "all") {
    return fromAlert;
  }
  return effectiveOpsDomain(selectedDomain);
}

export function workbenchCapabilityForDomain(domain: string, mode: WorkbenchAiMode): string {
  const normalized = resolveWorkbenchAlertDomain(domain, domain);
  if (mode === "action") {
    return "incident";
  }
  switch (normalized) {
    case "governance":
      return "governance";
    case "gbase":
      return "diagnosis";
    case "fi":
      return "inspection";
    default:
      return "observability-alert";
  }
}

const ANALYSIS_PLACEHOLDERS = new Set([
  "",
  "暂无根因分析",
  "Agent 正在分析合并告警…",
  "—",
  "当前告警缺少明确根因，点击对应 AI 操作发起实时分析。",
]);

export function hasMeaningfulAlertAnalysis(alert: WorkbenchAiAlert): boolean {
  const md = alert.analysisMarkdown?.trim() ?? "";
  if (md.length > 30 && !ANALYSIS_PLACEHOLDERS.has(md)) {
    return true;
  }
  const rc = alert.rootCause?.trim() ?? "";
  return rc.length > 30 && !ANALYSIS_PLACEHOLDERS.has(rc);
}

export function analysisContentFromAlert(alert: WorkbenchAiAlert, mode: WorkbenchAiMode = "root-cause"): string {
  if (mode !== "root-cause") {
    return "";
  }
  const md = alert.analysisMarkdown?.trim() ?? "";
  if (md.length > 0 && !ANALYSIS_PLACEHOLDERS.has(md)) {
    return md;
  }
  const rc = alert.rootCause?.trim() ?? "";
  if (rc.length > 0 && !ANALYSIS_PLACEHOLDERS.has(rc)) {
    return rc;
  }
  return "";
}

function isMeaningfulDiagnosis(text: string | null | undefined, minLength = 30): text is string {
  const trimmed = text?.trim() ?? "";
  return trimmed.length >= minLength && !ANALYSIS_PLACEHOLDERS.has(trimmed);
}

/** 会话历史复用需为完整报告，避免把「我需要分析…」等流式片段当成最终结果。 */
function isCompleteDiagnosisReport(text: string | null | undefined): text is string {
  const trimmed = text?.trim() ?? "";
  if (!isMeaningfulDiagnosis(trimmed, 50)) {
    return false;
  }
  const hasStructure = /##\s*(结论|根因|影响)|判断结论|根因分析|证据链|处置建议|排查步骤|影响面/i.test(trimmed);
  if (hasStructure) {
    return true;
  }
  const looksLikeOpeningOnly =
    /^(我需要分析|让我先|首先[，,]|正在分析)/.test(trimmed) && trimmed.length < 200;
  return !looksLikeOpeningOnly && trimmed.length >= 180;
}

function workbenchAiSessionKey(domain: string): string {
  return `agent:main:ops:${resolveWorkbenchAlertDomain(domain, domain)}:workbench-ai`;
}

function sessionSegment(value: string | undefined | null): string {
  const trimmed = (value ?? "").trim().toLowerCase();
  return trimmed
    .replace(/[^a-z0-9_-]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 96) || "workbench";
}

export function workbenchScenarioAiSessionKey(params: {
  employeeId: string;
  scenarioId: string;
  objectType?: string;
  objectId?: string;
}): string {
  const employeeId = sessionSegment(params.employeeId || "main");
  const scenarioId = sessionSegment(params.scenarioId);
  const objectType = sessionSegment(params.objectType || "object");
  const objectId = sessionSegment(params.objectId || "all");
  return `agent:main:employee:${employeeId}:workbench:${scenarioId}:${objectType}:${objectId}`;
}

function messageMatchesAlert(
  message: unknown,
  alertId: string,
  alertTitle: string,
): boolean {
  const text = extractText(message)?.trim() ?? "";
  if (!text) {
    return false;
  }
  if (alertTitle && text.includes(alertTitle)) {
    return true;
  }
  if (alertId && text.includes(alertId)) {
    return true;
  }
  const context = (message as { context?: { objectId?: string } })?.context;
  return Boolean(context?.objectId && context.objectId === alertId);
}

export async function loadDiagnosisFromSession(
  host: WorkbenchAiHost,
  sessionKey: string,
): Promise<string | null> {
  if (!host.client || !host.connected || !sessionKey.trim()) {
    return null;
  }
  try {
    const res = await host.client.request<{ messages?: unknown[] }>("chat.history", {
      sessionKey: canonicalGatewaySessionKey(sessionKey),
      limit: 80,
    });
    const messages = Array.isArray(res.messages) ? res.messages : [];
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      const message = messages[i] as { role?: string };
      if (message.role !== "assistant") {
        continue;
      }
      const text = extractText(message);
      if (isCompleteDiagnosisReport(text)) {
        return text.trim();
      }
    }
  } catch {
    return null;
  }
  return null;
}

export async function loadWorkbenchAiHistoryForAlert(
  host: WorkbenchAiHost,
  params: { domain: string; alertId: string; alertTitle: string },
): Promise<string | null> {
  if (!host.client || !host.connected) {
    return null;
  }
  try {
    const res = await host.client.request<{ messages?: unknown[] }>("chat.history", {
      sessionKey: canonicalGatewaySessionKey(workbenchAiSessionKey(params.domain)),
      limit: 120,
    });
    const messages = Array.isArray(res.messages) ? res.messages : [];
    for (let i = messages.length - 1; i >= 0; i -= 1) {
      const message = messages[i] as { role?: string };
      if (message.role !== "user" || !messageMatchesAlert(message, params.alertId, params.alertTitle)) {
        continue;
      }
      for (let j = i + 1; j < messages.length; j += 1) {
        const reply = messages[j] as { role?: string };
        if (reply.role !== "assistant") {
          continue;
        }
        const text = extractText(reply);
        if (isCompleteDiagnosisReport(text)) {
          return text.trim();
        }
        break;
      }
    }
  } catch {
    return null;
  }
  return null;
}

export function syncWorkbenchAiFromCopilot(host: WorkbenchAiHost, payload?: WorkbenchAiEventPayload): boolean {
  if (!payload || payload.state !== "final" || !host.workbenchAiCopilotRunId) {
    return false;
  }
  if (payload.runId !== host.workbenchAiCopilotRunId || !host.workbenchAiCopilotAlertId) {
    return false;
  }
  const text = extractText(payload.message);
  if (isMeaningfulDiagnosis(text, 20)) {
    applyWorkbenchAiResult(host, {
      alertId: host.workbenchAiCopilotAlertId,
      mode: host.workbenchAiCopilotMode ?? "root-cause",
      content: text.trim(),
    });
  }
  host.workbenchAiCopilotRunId = null;
  host.workbenchAiCopilotAlertId = null;
  host.workbenchAiCopilotMode = null;
  return true;
}

export function applyWorkbenchAiResult(
  host: WorkbenchAiHost,
  params: { alertId: string; mode: WorkbenchAiMode; content: string },
): void {
  host.workbenchAiObjectId = params.alertId;
  host.workbenchAiMode = params.mode;
  host.workbenchAiRunId = null;
  host.workbenchAiStream = null;
  host.workbenchAiError = null;
  host.workbenchAiResult = params.content;
  host.workbenchAiStatus = "done";
}

export async function openWorkbenchAiPanel(
  host: WorkbenchAiHost,
  params: {
    domain: string;
    mode: WorkbenchAiMode;
    alert: WorkbenchAiAlert;
    assistantTemplate: string;
    sessionKey?: string;
    force?: boolean;
  },
): Promise<void> {
  const { domain, mode, alert, assistantTemplate, sessionKey, force = false } = params;
  const alertDomain = resolveWorkbenchAlertDomain(alert.domain, domain);
  host.workbenchAiObjectId = alert.id;
  host.workbenchAiMode = mode;
  (host as WorkbenchAiHost & { workbenchAiDomain?: string }).workbenchAiDomain = alertDomain;

  if (!force && mode === "root-cause") {
    const cached = analysisContentFromAlert(alert, mode);
    if (isMeaningfulDiagnosis(cached)) {
      applyWorkbenchAiResult(host, { alertId: alert.id, mode, content: cached });
      return;
    }

    if (sessionKey?.trim()) {
      const fromAlertSession = await loadDiagnosisFromSession(host, sessionKey);
      if (isCompleteDiagnosisReport(fromAlertSession)) {
        applyWorkbenchAiResult(host, { alertId: alert.id, mode, content: fromAlertSession });
        return;
      }
    }

    const fromWorkbenchSession = await loadWorkbenchAiHistoryForAlert(host, {
      domain: alertDomain,
      alertId: alert.id,
      alertTitle: alert.title,
    });
    if (isCompleteDiagnosisReport(fromWorkbenchSession)) {
      applyWorkbenchAiResult(host, { alertId: alert.id, mode, content: fromWorkbenchSession });
      return;
    }
  }

  await runWorkbenchAi(host, {
    domain: alertDomain,
    mode,
    alert: { ...alert, domain: alertDomain },
    assistantTemplate,
    sessionKey: alert.sessionKey || `agent:main:${alert.id}`,
  });
}

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
  sessionKey?: string;
  domain?: string;
  objectType?: string;
  objectId?: string;
  scenarioTitle?: string;
  scenarioSummary?: string;
  evidence?: string[];
  expectedOutputs?: string[];
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
  workbenchAiDomain?: string;
  workbenchAiCopilotRunId?: string | null;
  workbenchAiCopilotAlertId?: string | null;
  workbenchAiCopilotMode?: WorkbenchAiMode | null;
  loadOpsDomainAlerts?: (domain: string) => Promise<void>;
};

export type WorkbenchAiEventPayload = {
  runId: string;
  sessionKey: string;
  state: "delta" | "final" | "aborted" | "error";
  message?: unknown;
  errorMessage?: string;
};

function questionForMode(mode: WorkbenchAiMode, alert: WorkbenchAiAlert): string {
  if (alert.scenarioTitle) {
    switch (mode) {
      case "action":
        return `请基于当前运维专项给出处置建议，区分只读排查、需要审批的变更和高风险操作，并按步骤说明：${alert.scenarioTitle}`;
      case "similar":
        return `请分析当前运维专项是否存在相似对象、相似风险或可批量治理机会，并给出聚合逻辑：${alert.scenarioTitle}`;
      case "root-cause":
      default:
        return `请基于当前运维专项分析风险根因、证据链、影响面和验证步骤：${alert.scenarioTitle}`;
    }
  }
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
    alert.scenarioTitle ? `专项场景: ${alert.scenarioTitle}` : "",
    alert.scenarioSummary ? `场景说明: ${alert.scenarioSummary}` : "",
    alert.objectType ? `对象类型: ${alert.objectType}` : "",
    alert.objectId ? `对象范围: ${alert.objectId}` : "",
    typeof alert.originalCount === "number" ? `原始告警数: ${alert.originalCount}` : "",
    typeof alert.reducedTo === "number" ? `降噪后: ${alert.reducedTo}` : "",
    alert.rootCause ? `根因候选: ${alert.rootCause}` : "",
    alert.impact ? `影响范围: ${alert.impact}` : "",
    alert.analysisMarkdown ? `已有分析材料: ${alert.analysisMarkdown}` : "",
    alert.evidence?.length ? `输入证据: ${alert.evidence.join(" / ")}` : "",
    alert.expectedOutputs?.length ? `期望输出: ${alert.expectedOutputs.join(" / ")}` : "",
  ]
    .filter(Boolean)
    .join("\n");
}

export async function runWorkbenchAi(
  host: WorkbenchAiHost,
  params: { domain: string; mode: WorkbenchAiMode; alert: WorkbenchAiAlert; assistantTemplate: string; sessionKey?: string },
): Promise<void> {
  const { domain, mode, alert, assistantTemplate } = params;
  const alertDomain = resolveWorkbenchAlertDomain(alert.domain, domain);
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

  const sessionKey = params.sessionKey?.trim() || alert.sessionKey?.trim() || `agent:main:${alert.id}`;
  const runId = generateUUID();
  const anyHost = host as any;
  host.workbenchAiSessionKey = sessionKey;
  host.workbenchAiRunId = runId;
  host.workbenchAiMode = mode;
  host.workbenchAiObjectId = alert.id;
  host.workbenchAiStream = "";
  host.workbenchAiResult = null;
  host.workbenchAiError = null;
  host.workbenchAiStatus = "loading";
  anyHost._workbenchAiAccumulated = "";
  anyHost._workbenchAiMsgId = null;

  try {
    await host.client.request("chat.send", {
      sessionKey: canonicalGatewaySessionKey(sessionKey),
      message: questionForMode(mode, alert),
      deliver: false,
      idempotencyKey: runId,
      modelRef: host.chatModelRef ?? undefined,
      assistantTemplate: assistantTemplate || undefined,
      context: {
        domain: alertDomain,
        capability: workbenchCapabilityForDomain(alertDomain, mode),
        workflowType: mode === "action" ? "incident" : "diagnosis",
        objectType: alert.objectType || "alert",
        objectId: alert.objectId || alert.id,
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
  if (
    payload.runId !== host.workbenchAiRunId &&
    !gatewaySessionKeysEqual(payload.sessionKey, host.workbenchAiSessionKey)
  ) {
    return false;
  }

  const anyHost = host as any;
  const msgId = payload.message && typeof payload.message === "object" ? (payload.message as any).id : null;

  if (payload.state === "delta") {
    const next = extractText(payload.message);
    if (typeof next === "string") {
      if (msgId && anyHost._workbenchAiMsgId !== msgId) {
        if (anyHost._workbenchAiMsgId && host.workbenchAiStream) {
          anyHost._workbenchAiAccumulated = (anyHost._workbenchAiAccumulated || "") + host.workbenchAiStream + "\n\n";
        }
        anyHost._workbenchAiMsgId = msgId;
      }
      host.workbenchAiStream = next;
    }
    host.workbenchAiStatus = "streaming";
  } else if (payload.state === "final") {
    const finalText = extractText(payload.message);
    if (typeof finalText === "string") {
      if (msgId && anyHost._workbenchAiMsgId !== msgId) {
        if (anyHost._workbenchAiMsgId && host.workbenchAiStream) {
          anyHost._workbenchAiAccumulated = (anyHost._workbenchAiAccumulated || "") + host.workbenchAiStream + "\n\n";
        }
        anyHost._workbenchAiMsgId = msgId;
      }
    }
    const lastPart = typeof finalText === "string" ? finalText : (host.workbenchAiStream || "");
    host.workbenchAiResult = (anyHost._workbenchAiAccumulated || "") + lastPart;
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = "done";
    anyHost._workbenchAiAccumulated = "";
    anyHost._workbenchAiMsgId = null;
    if (host.workbenchAiDomain && host.loadOpsDomainAlerts) {
      void host.loadOpsDomainAlerts(host.workbenchAiDomain);
    }
  } else if (payload.state === "aborted") {
    host.workbenchAiResult = (anyHost._workbenchAiAccumulated || "") + (host.workbenchAiStream || "");
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = host.workbenchAiResult ? "done" : "idle";
    anyHost._workbenchAiAccumulated = "";
    anyHost._workbenchAiMsgId = null;
  } else if (payload.state === "error") {
    host.workbenchAiStream = null;
    host.workbenchAiRunId = null;
    host.workbenchAiStatus = "error";
    host.workbenchAiError = payload.errorMessage ?? "AI 分析失败";
    anyHost._workbenchAiAccumulated = "";
    anyHost._workbenchAiMsgId = null;
  }
  return true;
}
