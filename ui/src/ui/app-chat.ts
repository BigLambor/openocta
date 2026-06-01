import type { OpenClawApp } from "./app.ts";
import type { GatewayHelloOk } from "./gateway.ts";
import type { ChatAttachment, ChatQueueItem } from "./ui-types.ts";
import {
  gatewaySessionKeysEqual,
  isCronRunSessionKey,
  isEmployeeRunSessionKey,
  isStableEmployeeWebchatSessionKey,
  parseAgentSessionKey,
} from "./sessions/session-key-utils.js";
import { scheduleChatScroll } from "./app-scroll.ts";
import { setLastActiveSessionKey, syncUrlWithSessionKey } from "./app-settings.ts";
import { resetToolStream } from "./app-tool-stream.ts";
import { abortChatRun, loadChatHistory, sendChatMessage } from "./controllers/chat.ts";
import { loadSessions } from "./controllers/sessions.ts";
import { normalizeBasePath, titleForTab, type Tab } from "./navigation.ts";
import { buildOpsContextLine, isOpsDomainTab } from "./ops/entity-config.ts";
import { generateUUID } from "./uuid.ts";

export type ChatHost = {
  connected: boolean;
  chatMessage: string;
  chatAttachments: ChatAttachment[];
  chatModelRef?: string | null;
  chatQueue: ChatQueueItem[];
  chatRunId: string | null;
  chatSending: boolean;
  sessionKey: string;
  basePath: string;
  hello: GatewayHelloOk | null;
  chatAvatarUrl: string | null;
  refreshSessionsAfterChat: Set<string>;
};

export const CHAT_SESSIONS_ACTIVE_MINUTES = 0;

/**
 * 若当前 sessionKey 已不在网关 sessions.list 中（例如已删除），则切换到可用会话并更新 URL，
 * 避免 chat.history 仍按旧 key 返回已归档内容。仅在列表已成功加载时生效。
 */
async function reconcileInvalidChatSessionFromList(host: OpenClawApp): Promise<boolean> {
  const key = host.sessionKey?.trim();
  if (!key) {
    return false;
  }
  const rows = host.sessionsResult?.sessions;
  if (!Array.isArray(rows)) {
    return false;
  }

  const inList = rows.some((row) => gatewaySessionKeysEqual(row.key, key));
  if (rows.length > 0 && inList) {
    return false;
  }
  // 新开员工会话：首条 chat.send 完成前 store 里可能还没有该 key，避免误判并切走 session。
  if (!inList && isEmployeeRunSessionKey(key)) {
    return false;
  }
  if (!inList && isStableEmployeeWebchatSessionKey(key)) {
    return false;
  }
  // Cron 单次运行会话被服务端 sessions.list 刻意排除，仍可通过 chat.history 访问。
  if (!inList && isCronRunSessionKey(key)) {
    return false;
  }
  if (rows.length === 0 && key === "agent.main.main") {
    return false;
  }

  const fallback =
    rows.length > 0
      ? (rows.find((r) => r.key && r.kind !== "global") ?? rows[0])?.key?.trim() ||
        "agent.main.main"
      : "agent.main.main";

  if (fallback === key) {
    return false;
  }

  host.sessionKey = fallback;
  host.applySettings({
    ...host.settings,
    sessionKey: fallback,
    lastActiveSessionKey: fallback,
  });
  host.chatMessage = "";
  host.chatAttachments = [];
  host.chatRunId = null;
  host.chatStream = null;
  host.chatStreamStartedAt = null;
  host.chatSending = false;
  host.resetToolStream();
  await host.loadAssistantIdentity();
  syncUrlWithSessionKey(
    host as unknown as Parameters<typeof syncUrlWithSessionKey>[0],
    fallback,
    true,
  );
  return true;
}

export function isChatBusy(host: ChatHost) {
  return host.chatSending || Boolean(host.chatRunId);
}

export function isChatStopCommand(text: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    return false;
  }
  const normalized = trimmed.toLowerCase();
  if (normalized === "/stop") {
    return true;
  }
  return (
    normalized === "stop" ||
    normalized === "esc" ||
    normalized === "abort" ||
    normalized === "wait" ||
    normalized === "exit"
  );
}

function isChatResetCommand(text: string) {
  const trimmed = text.trim();
  if (!trimmed) {
    return false;
  }
  const normalized = trimmed.toLowerCase();
  if (normalized === "/new" || normalized === "/reset") {
    return true;
  }
  return normalized.startsWith("/new ") || normalized.startsWith("/reset ");
}

export async function handleAbortChat(host: ChatHost) {
  if (!host.connected) {
    return;
  }
  host.chatMessage = "";
  await abortChatRun(host as unknown as OpenClawApp);
}

function enqueueChatMessage(
  host: ChatHost,
  text: string,
  attachments?: ChatAttachment[],
  refreshSessions?: boolean,
) {
  const trimmed = text.trim();
  const hasAttachments = Boolean(attachments && attachments.length > 0);
  if (!trimmed && !hasAttachments) {
    return;
  }
  host.chatQueue = [
    ...host.chatQueue,
    {
      id: generateUUID(),
      sessionKey: host.sessionKey,
      text: trimmed,
      createdAt: Date.now(),
      attachments: hasAttachments ? attachments?.map((att) => ({ ...att })) : undefined,
      refreshSessions,
    },
  ];
}

async function sendChatMessageNow(
  host: ChatHost,
  message: string,
  opts?: {
    previousDraft?: string;
    restoreDraft?: boolean;
    attachments?: ChatAttachment[];
    previousAttachments?: ChatAttachment[];
    restoreAttachments?: boolean;
    refreshSessions?: boolean;
  },
) {
  resetToolStream(host as unknown as Parameters<typeof resetToolStream>[0]);
  const runId = await sendChatMessage(
    host as unknown as OpenClawApp,
    message,
    opts?.attachments,
    host.chatModelRef ?? null,
  );
  const ok = Boolean(runId);
  if (!ok && opts?.previousDraft != null) {
    host.chatMessage = opts.previousDraft;
  }
  if (!ok && opts?.previousAttachments) {
    host.chatAttachments = opts.previousAttachments;
  }
  if (ok) {
    setLastActiveSessionKey(
      host as unknown as Parameters<typeof setLastActiveSessionKey>[0],
      host.sessionKey,
    );
  }
  if (ok && opts?.restoreDraft && opts.previousDraft?.trim()) {
    host.chatMessage = opts.previousDraft;
  }
  if (ok && opts?.restoreAttachments && opts.previousAttachments?.length) {
    host.chatAttachments = opts.previousAttachments;
  }
  scheduleChatScroll(host as unknown as Parameters<typeof scheduleChatScroll>[0]);
  if (ok && !host.chatRunId) {
    void flushChatQueue(host);
  }
  if (ok && opts?.refreshSessions && runId) {
    host.refreshSessionsAfterChat.add(runId);
  }
  return ok;
}

async function flushChatQueue(host: ChatHost) {
  if (!host.connected || isChatBusy(host)) {
    return;
  }
  const idx = host.chatQueue.findIndex((item) => item.sessionKey === host.sessionKey);
  if (idx === -1) return;
  const next = host.chatQueue[idx];
  host.chatQueue = [...host.chatQueue.slice(0, idx), ...host.chatQueue.slice(idx + 1)];
  const ok = await sendChatMessageNow(host, next.text, {
    attachments: next.attachments,
    refreshSessions: next.refreshSessions,
  });
  if (!ok) {
    // 发送失败：放回队列头部，避免丢失
    host.chatQueue = [next, ...host.chatQueue];
  }
}

export function removeQueuedMessage(host: ChatHost, id: string) {
  host.chatQueue = host.chatQueue.filter((item) => item.id !== id);
}

export async function handleSendChat(
  host: ChatHost,
  messageOverride?: string,
  opts?: { restoreDraft?: boolean; refreshSessions?: boolean },
) {
  if (!host.connected) {
    return;
  }
  const previousDraft = host.chatMessage;
  const message = (messageOverride ?? host.chatMessage).trim();
  const attachments = host.chatAttachments ?? [];
  const attachmentsToSend = messageOverride == null ? attachments : [];
  const hasAttachments = attachmentsToSend.length > 0;

  // Allow sending with just attachments (no message text required)
  if (!message && !hasAttachments) {
    return;
  }

  if (isChatStopCommand(message)) {
    await handleAbortChat(host);
    return;
  }

  const messageWithOpsContext = prependOpsContextIfNeeded(host as unknown as OpenClawApp, message);

  const refreshSessions = opts?.refreshSessions ?? isChatResetCommand(message);
  if (messageOverride == null) {
    host.chatMessage = "";
    // Clear attachments when sending
    host.chatAttachments = [];
  }

  if (isChatBusy(host)) {
    enqueueChatMessage(host, messageWithOpsContext, attachmentsToSend, refreshSessions);
    return;
  }

  await sendChatMessageNow(host, messageWithOpsContext, {
    previousDraft: messageOverride == null ? previousDraft : undefined,
    restoreDraft: Boolean(messageOverride && opts?.restoreDraft),
    attachments: hasAttachments ? attachmentsToSend : undefined,
    previousAttachments: messageOverride == null ? attachments : undefined,
    restoreAttachments: Boolean(messageOverride && opts?.restoreDraft),
    refreshSessions,
  });
}

export async function refreshChat(host: ChatHost) {
  const app = host as unknown as OpenClawApp;
  // 使用较大 limit，避免列表按更新时间截断后把当前会话误判为「已删除」（与员工市场入口的 loadSessions 不一致时尤其明显）。
  await loadSessions(app, {
    activeMinutes: CHAT_SESSIONS_ACTIVE_MINUTES,
    includeLastMessage: true,
    limit: 5000,
  });
  const switched = await reconcileInvalidChatSessionFromList(app);
  await Promise.all([loadChatHistory(app), refreshChatAvatar(host)]);
  if (switched && !app.lastError) {
    app.lastError = "该会话已不存在或已删除，已切换到可用会话。";
  }
  scheduleChatScroll(host as unknown as Parameters<typeof scheduleChatScroll>[0]);
}

export const flushChatQueueForEvent = flushChatQueue;

function prependOpsContextIfNeeded(app: OpenClawApp, message: string): string {
  if (!message || message.startsWith("/")) {
    return message;
  }
  if (message.includes("[运维上下文]")) {
    return message;
  }
  const tab = app.tab;
  if (!isOpsDomainTab(tab)) {
    return message;
  }
  const clusters = app.opsDomainClusters?.[tab] ?? [];
  const entityId = app.opsSelectedEntityIds?.[tab] ?? "all";
  const line = buildOpsContextLine(titleForTab(tab as Tab), entityId, clusters);
  return `${line}\n\n${message}`;
}

type SessionDefaultsSnapshot = {
  defaultAgentId?: string;
};

function resolveAgentIdForSession(host: ChatHost): string | null {
  const parsed = parseAgentSessionKey(host.sessionKey);
  if (parsed?.agentId) {
    return parsed.agentId;
  }
  const snapshot = host.hello?.snapshot as
    | { sessionDefaults?: SessionDefaultsSnapshot }
    | undefined;
  const fallback = snapshot?.sessionDefaults?.defaultAgentId?.trim();
  return fallback || "main";
}

function buildAvatarMetaUrl(basePath: string, agentId: string): string {
  const base = normalizeBasePath(basePath);
  const encoded = encodeURIComponent(agentId);
  return base ? `${base}/avatar/${encoded}?meta=1` : `/avatar/${encoded}?meta=1`;
}

export async function refreshChatAvatar(host: ChatHost) {
  if (!host.connected) {
    host.chatAvatarUrl = null;
    return;
  }
  const agentId = resolveAgentIdForSession(host);
  if (!agentId) {
    host.chatAvatarUrl = null;
    return;
  }
  host.chatAvatarUrl = null;
  const url = buildAvatarMetaUrl(host.basePath, agentId);
  try {
    const res = await fetch(url, { method: "GET" });
    if (!res.ok) {
      host.chatAvatarUrl = null;
      return;
    }
    const data = (await res.json()) as { avatarUrl?: unknown };
    const avatarUrl = typeof data.avatarUrl === "string" ? data.avatarUrl.trim() : "";
    host.chatAvatarUrl = avatarUrl || null;
  } catch {
    host.chatAvatarUrl = null;
  }
}
