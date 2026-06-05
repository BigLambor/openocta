import type { OpenClawApp } from "./app.ts";
import { parseGatewayHost } from "./gateway-url.ts";
import type { AgentsListResult } from "./types.ts";
import { refreshChat } from "./app-chat.ts";
import {
  startLogsPolling,
  stopLogsPolling,
  startDebugPolling,
  stopDebugPolling,
} from "./app-polling.ts";
import { scheduleChatScroll, scheduleLogsScroll } from "./app-scroll.ts";
import { loadAgentIdentities, loadAgentIdentity } from "./controllers/agent-identity.ts";
import { loadAgentSkills } from "./controllers/agent-skills.ts";
import { loadAgents } from "./controllers/agents.ts";
import { loadChannels } from "./controllers/channels.ts";
import { loadConfig, loadConfigSchema } from "./controllers/config.ts";
import { loadCronJobs, loadCronStatus } from "./controllers/cron.ts";
import { loadDebug } from "./controllers/debug.ts";
import { loadDevices } from "./controllers/devices.ts";
import { loadExecApprovals } from "./controllers/exec-approvals.ts";
import { loadLogs } from "./controllers/logs.ts";
import { loadNodes } from "./controllers/nodes.ts";
import { loadPresence } from "./controllers/presence.ts";
import { loadSessions } from "./controllers/sessions.ts";
import { loadDigitalEmployees } from "./controllers/digital-employees.ts";
import { loadEmployeeTasks, loadEmployeeEffectiveness } from "./controllers/employee-tasks.ts";
import { loadTraceList } from "./controllers/llm-trace.ts";
import { syncLlmTraceFromConfig } from "./app-llm-trace.ts";
import { syncSecurityFromConfig } from "./app-security.ts";
import { loadApprovalsList } from "./controllers/approvals.ts";
import { loadSkills } from "./controllers/skills.ts";
import {
  inferBasePathFromPathname,
  normalizeBasePath,
  normalizePath,
  pathForTab,
  tabFromPath,
  titleForTab,
  type Tab,
} from "./navigation.ts";
import { saveSettings, type UiSettings } from "./storage.ts";
import { normalizeOpsDomain } from "./components/domain-filter.ts";
import { applyOpsDeepLinkFromUrl } from "./ops/deeplink.ts";
import { isOpsDomainTab } from "./ops/entity-config.ts";
import { ensureDefaultOpsCapabilityTab } from "./ops/navigation.ts";
import { startThemeTransition, type ThemeTransitionContext } from "./theme-transition.ts";
import { resolveTheme, type ResolvedTheme, type ThemeMode } from "./theme.ts";

type SettingsHost = {
  settings: UiSettings;
  password?: string;
  theme: ThemeMode;
  themeResolved: ResolvedTheme;
  applySessionKey: string;
  sessionKey: string;
  tab: Tab;
  connected: boolean;
  chatHasAutoScrolled: boolean;
  logsAtBottom: boolean;
  eventLog: unknown[];
  eventLogBuffer: unknown[];
  securityForm?: unknown;
  basePath: string;
  agentsList?: AgentsListResult | null;
  agentsSelectedId?: string | null;
  agentsPanel?: "overview" | "files" | "tools" | "skills" | "channels" | "cron";
  themeMedia: MediaQueryList | null;
  themeMediaHandler: ((event: MediaQueryListEvent) => void) | null;
  pendingGatewayUrl?: string | null;
  rbacUser?: any;
};

function scrollContentToTop(host: SettingsHost) {
  const querySelector = (host as unknown as ParentNode).querySelector;
  if (typeof querySelector !== "function") {
    return;
  }
  const content = querySelector.call(host, ".content");
  if (!(content instanceof HTMLElement)) {
    return;
  }
  content.scrollTop = 0;
}

function canOpenTab(host: SettingsHost, tab: Tab): boolean {
  if (!host.rbacUser) {
    return true;
  }
  if (host.rbacUser.roleName === "admin") {
    return true;
  }
  if (tab === "automation" || tab === "employeeCenter" || tab === "employeeMarket" || tab === "digitalEmployee" || tab === "agentSwarm") {
    return false;
  }
  if (tab === "workbench" || tab === "assets" || tab === "techDomains" || tab === "opsCapabilities") {
    return ["hadoop", "fi", "gbase", "governance", "dataapps"].some((key) =>
      host.rbacUser.permissions?.includes(`menu:${key}`),
    );
  }
  if (["overview", "hadoop", "fi", "gbase", "governance", "dataapps", "config"].includes(tab)) {
    return host.rbacUser.permissions?.includes(`menu:${tab}`);
  }
  return true;
}

export function applySettings(host: SettingsHost, next: UiSettings) {
  const normalized = {
    ...next,
    lastActiveSessionKey: next.lastActiveSessionKey?.trim() || next.sessionKey.trim() || "main",
  };
  host.settings = normalized;
  saveSettings(normalized);
  if (next.theme !== host.theme) {
    host.theme = next.theme;
    applyResolvedTheme(host, resolveTheme(next.theme));
  }
  host.applySessionKey = host.settings.lastActiveSessionKey;
}

export function setLastActiveSessionKey(host: SettingsHost, next: string) {
  const trimmed = next.trim();
  if (!trimmed) {
    return;
  }
  if (host.settings.lastActiveSessionKey === trimmed) {
    return;
  }
  applySettings(host, { ...host.settings, lastActiveSessionKey: trimmed });
}

export function applySettingsFromUrl(host: SettingsHost) {
  if (!window.location.search) {
    return;
  }
  const params = new URLSearchParams(window.location.search);
  const tokenRaw = params.get("token");
  const passwordRaw = params.get("password");
  const sessionRaw = params.get("session");
  const gatewayUrlRaw = params.get("gatewayUrl");
  let shouldCleanUrl = false;

  if (tokenRaw != null) {
    params.delete("token");
    shouldCleanUrl = true;
  }

  if (passwordRaw != null) {
    const password = passwordRaw.trim();
    if (password) {
      (host as { password: string }).password = password;
    }
    params.delete("password");
    shouldCleanUrl = true;
  }

  if (sessionRaw != null) {
    const session = sessionRaw.trim();
    if (session) {
      host.sessionKey = session;
      applySettings(host, {
        ...host.settings,
        sessionKey: session,
        lastActiveSessionKey: session,
      });
    }
  }

  if (gatewayUrlRaw != null) {
    const raw = gatewayUrlRaw.trim();
    const gatewayHost = raw ? parseGatewayHost(raw) : "";
    if (gatewayHost && gatewayHost !== host.settings.gatewayUrl) {
      host.pendingGatewayUrl = gatewayHost;
    }
    params.delete("gatewayUrl");
    shouldCleanUrl = true;
  }

  const domainRaw = params.get("domain");
  if (domainRaw != null) {
    const domain = normalizeOpsDomain(domainRaw);
    applySettings(host, { ...host.settings, opsDomain: domain });
  }

  if (!shouldCleanUrl) {
    return;
  }
  const url = new URL(window.location.href);
  url.search = params.toString();
  window.history.replaceState({}, "", url.toString());
}

export function setTab(host: SettingsHost, next: Tab) {
  const nextTab =
    next === "chat" && (host.sessionKey?.trim() ?? "") ? ("message" as Tab) : next;

  if (!canOpenTab(host, nextTab)) {
    return;
  }

  const tabChanged = host.tab !== nextTab;
  if (tabChanged) {
    host.tab = nextTab;
    scrollContentToTop(host);
  }
  if (isOpsDomainTab(nextTab)) {
    ensureDefaultOpsCapabilityTab(
      host as unknown as Parameters<typeof ensureDefaultOpsCapabilityTab>[0],
      nextTab,
    );
    const domainSessionKey = `agent:main:ops:${nextTab}`;
    if (host.sessionKey !== domainSessionKey) {
      host.sessionKey = domainSessionKey;
      host.applySessionKey = domainSessionKey;
    }
  }
  if (nextTab === "chat") {
    host.chatHasAutoScrolled = false;
  }
  if (nextTab === "logs") {
    startLogsPolling(host as unknown as Parameters<typeof startLogsPolling>[0]);
  } else {
    stopLogsPolling(host as unknown as Parameters<typeof stopLogsPolling>[0]);
  }
  if (nextTab === "debug") {
    startDebugPolling(host as unknown as Parameters<typeof startDebugPolling>[0]);
  } else {
    stopDebugPolling(host as unknown as Parameters<typeof stopDebugPolling>[0]);
  }
  if (isOpsDomainTab(nextTab)) {
    const link = applyOpsDeepLinkFromUrl(host as unknown as Parameters<typeof applyOpsDeepLinkFromUrl>[0]);
    if (link.alertsTab) {
      const app = host as unknown as { loadOpsDomainAlerts?: (d: string) => Promise<void> };
      void app.loadOpsDomainAlerts?.(nextTab);
    }
  }
  void refreshActiveTab(host);
  syncUrlWithTab(host, nextTab, false);
}

export function setTheme(host: SettingsHost, next: ThemeMode, context?: ThemeTransitionContext) {
  const applyTheme = () => {
    host.theme = next;
    applySettings(host, { ...host.settings, theme: next });
    applyResolvedTheme(host, resolveTheme(next));
  };
  startThemeTransition({
    nextTheme: next,
    applyTheme,
    context,
    currentTheme: host.theme,
  });
}

export async function refreshActiveTab(host: SettingsHost) {
  if (isOpsDomainTab(host.tab)) {
    const domainSessionKey = `agent:main:ops:${host.tab}`;
    const domainLabel = `${titleForTab(host.tab)}智能诊断`;
    const { ensureSessionForKey } = await import("./controllers/sessions.ts");
    const { loadChatHistory } = await import("./controllers/chat.ts");
    const { loadConfig } = await import("./controllers/config.ts");
    await loadConfig(host as any);
    await ensureSessionForKey(host as any, { key: domainSessionKey, label: domainLabel });
    await loadChatHistory(host as any);
    const app = host as unknown as {
      loadOpsDomainClusters?: (d: string) => Promise<void>;
      loadOpsDomainAlerts?: (d: string) => Promise<void>;
    };
    await Promise.allSettled([
      app.loadOpsDomainClusters?.(host.tab) ?? Promise.resolve(),
      app.loadOpsDomainAlerts?.(host.tab) ?? Promise.resolve(),
    ]);
  }
  if (host.tab === "overview" || host.tab === "techDomains") {
    await loadOverview(host);
    const app = host as unknown as {
      loadOpsDashboard?: () => Promise<void>;
    };
    if (app.loadOpsDashboard) {
      await app.loadOpsDashboard();
    }
  }
  if (host.tab === "workbench") {
    const app = host as unknown as { loadOpsDomainAlerts?: (d: string) => Promise<void> };
    const domain = normalizeOpsDomain(host.settings.opsDomain);
    await app.loadOpsDomainAlerts?.(domain === "all" ? "hadoop" : domain);
  }
  if (host.tab === "assets" || host.tab === "assetManagement") {
    const app = host as unknown as { loadOpsClusters?: () => Promise<void> };
    if (app.loadOpsClusters) {
      await app.loadOpsClusters();
    }
  }
  if (host.tab === "channels") {
    await loadChannelsTab(host);
  }
  if (host.tab === "instances") {
    await loadPresence(host as unknown as OpenClawApp);
  }
  if (host.tab === "sessions") {
    await loadSessions(host as unknown as OpenClawApp, { includeLastMessage: true });
  }
  if (host.tab === "cron") {
    await loadCron(host);
    // Cron 配置需要数字员工列表用于下拉选择/校验。这里 await，避免编辑时误判“已删除”。
    await loadDigitalEmployees(host as unknown as Parameters<typeof loadDigitalEmployees>[0]);
  }
  if (host.tab === "scheduledTasks") {
    await loadCron(host);
    await loadDigitalEmployees(host as unknown as Parameters<typeof loadDigitalEmployees>[0]);
  }
  if (host.tab === "cronHistory") {
    await loadCron(host);
    await loadDigitalEmployees(host as unknown as Parameters<typeof loadDigitalEmployees>[0]);
  }
  if (host.tab === "skills") {
    await loadSkills(host as unknown as OpenClawApp);
  }
  if (host.tab === "skillLibrary") {
    await loadSkills(host as unknown as OpenClawApp);
  }
  if (host.tab === "toolLibrary") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
  }
  if (host.tab === "mcp") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
  }
  if (host.tab === "llmTrace") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
    await loadTraceList(host as unknown as Parameters<typeof loadTraceList>[0]);
  }
  if (host.tab === "sandbox") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    host.securityForm = syncSecurityFromConfig(host as unknown as Parameters<typeof syncSecurityFromConfig>[0]);
    await loadApprovalsList(host as unknown as Parameters<typeof loadApprovalsList>[0]);
  }
  if (host.tab === "digitalEmployee") {
    await loadDigitalEmployees(host as unknown as Parameters<typeof loadDigitalEmployees>[0]);
  }
  if (host.tab === "employeeTasks") {
    await loadEmployeeTasks(host as any);
    await loadDigitalEmployees(host as any);
  }
  if (host.tab === "employeeEffectiveness") {
    await loadEmployeeEffectiveness(host as any);
    await loadDigitalEmployees(host as any);
  }
  if (host.tab === "agentSwarm") {
    const { ensureSwarmWorkspace } = await import("./controllers/swarm.ts");
    await ensureSwarmWorkspace(host as unknown as Parameters<typeof ensureSwarmWorkspace>[0]);
  }
  if (host.tab === "agents") {
    await loadAgents(host as unknown as OpenClawApp);
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
    const agentIds = host.agentsList?.agents?.map((entry) => entry.id) ?? [];
    if (agentIds.length > 0) {
      void loadAgentIdentities(host as unknown as OpenClawApp, agentIds);
    }
    const agentId =
      host.agentsSelectedId ?? host.agentsList?.defaultId ?? host.agentsList?.agents?.[0]?.id;
    if (agentId) {
      void loadAgentIdentity(host as unknown as OpenClawApp, agentId);
      if (host.agentsPanel === "skills") {
        void loadAgentSkills(host as unknown as OpenClawApp, agentId);
      }
      if (host.agentsPanel === "channels") {
        void loadChannels(host as unknown as OpenClawApp, false);
      }
      if (host.agentsPanel === "cron") {
        void loadCron(host);
      }
    }
  }
  if (host.tab === "nodes") {
    await loadNodes(host as unknown as OpenClawApp);
    await loadDevices(host as unknown as OpenClawApp);
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
    await loadExecApprovals(host as unknown as OpenClawApp);
  }
  if (host.tab === "chat" || host.tab === "message") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    await refreshChat(host as unknown as Parameters<typeof refreshChat>[0]);
    void loadDigitalEmployees(host as unknown as Parameters<typeof loadDigitalEmployees>[0]);
    scheduleChatScroll(
      host as unknown as Parameters<typeof scheduleChatScroll>[0],
      !host.chatHasAutoScrolled,
    );
  }
  if (host.tab === "config") {
    await loadConfigSchema(host as unknown as OpenClawApp);
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
  }
  if (host.tab === "envVars" || host.tab === "models" || host.tab === "modelLibrary") {
    await loadConfig(host as unknown as Parameters<typeof loadConfig>[0]);
    syncLlmTraceFromConfig(host as unknown as Parameters<typeof syncLlmTraceFromConfig>[0]);
  }
  if (host.tab === "debug") {
    await loadDebug(host as unknown as OpenClawApp);
    host.eventLog = host.eventLogBuffer;
  }
  if (host.tab === "logs") {
    host.logsAtBottom = true;
    await loadLogs(host as unknown as OpenClawApp, { reset: true });
    scheduleLogsScroll(host as unknown as Parameters<typeof scheduleLogsScroll>[0], true);
  }
}

export function inferBasePath() {
  if (typeof window === "undefined") {
    return "";
  }
  const configured = window.__OPENCLAW_CONTROL_UI_BASE_PATH__;
  if (typeof configured === "string" && configured.trim()) {
    return normalizeBasePath(configured);
  }
  return inferBasePathFromPathname(window.location.pathname);
}

export function syncThemeWithSettings(host: SettingsHost) {
  host.theme = host.settings.theme ?? "light";
  applyResolvedTheme(host, resolveTheme(host.theme));
}

export function applyResolvedTheme(host: SettingsHost, resolved: ResolvedTheme) {
  host.themeResolved = resolved;
  if (typeof document === "undefined") {
    return;
  }
  const root = document.documentElement;
  root.dataset.theme = resolved;
  root.style.colorScheme = resolved;
}

export function attachThemeListener(host: SettingsHost) {
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
    return;
  }
  host.themeMedia = window.matchMedia("(prefers-color-scheme: dark)");
  host.themeMediaHandler = (event) => {
    if (host.theme !== "system") {
      return;
    }
    applyResolvedTheme(host, event.matches ? "dark" : "light");
  };
  if (typeof host.themeMedia.addEventListener === "function") {
    host.themeMedia.addEventListener("change", host.themeMediaHandler);
    return;
  }
  const legacy = host.themeMedia as MediaQueryList & {
    addListener: (cb: (event: MediaQueryListEvent) => void) => void;
  };
  legacy.addListener(host.themeMediaHandler);
}

export function detachThemeListener(host: SettingsHost) {
  if (!host.themeMedia || !host.themeMediaHandler) {
    return;
  }
  if (typeof host.themeMedia.removeEventListener === "function") {
    host.themeMedia.removeEventListener("change", host.themeMediaHandler);
    return;
  }
  const legacy = host.themeMedia as MediaQueryList & {
    removeListener: (cb: (event: MediaQueryListEvent) => void) => void;
  };
  legacy.removeListener(host.themeMediaHandler);
  host.themeMedia = null;
  host.themeMediaHandler = null;
}

/** 旧版书签/外链 /chat?session=… 与消息页统一为 /message?session=…，以保留侧栏会话列表 */
function normalizeLegacyChatSessionPath(host: SettingsHost): void {
  if (typeof window === "undefined") {
    return;
  }
  const url = new URL(window.location.href);
  const sessionFromUrl = url.searchParams.get("session")?.trim() ?? "";
  const tab = tabFromPath(url.pathname, host.basePath);
  if (tab !== "chat" || !sessionFromUrl) {
    return;
  }
  url.pathname = normalizePath(pathForTab("message", host.basePath));
  window.history.replaceState({}, "", url.toString());
}

export function syncTabWithLocation(host: SettingsHost, replace: boolean) {
  if (typeof window === "undefined") {
    return;
  }
  normalizeLegacyChatSessionPath(host);
  let resolved = tabFromPath(window.location.pathname, host.basePath) ?? "chat";
  // 配置入口默认进入概览
  if (resolved === "config") {
    resolved = "overview";
  }
  setTabFromRoute(host, resolved);
  const link = applyOpsDeepLinkFromUrl(host as unknown as Parameters<typeof applyOpsDeepLinkFromUrl>[0]);
  if (link.alertsTab) {
    const app = host as unknown as { loadOpsDomainAlerts?: (d: string) => Promise<void> };
    void app.loadOpsDomainAlerts?.(resolved);
  }
  syncUrlWithTab(host, resolved, replace);
}

export function onPopState(host: SettingsHost) {
  if (typeof window === "undefined") {
    return;
  }
  normalizeLegacyChatSessionPath(host);
  let resolved = tabFromPath(window.location.pathname, host.basePath);
  if (!resolved) {
    return;
  }
  if (resolved === "config") {
    resolved = "overview";
  }

  const url = new URL(window.location.href);
  const session = url.searchParams.get("session")?.trim();
  const domain = url.searchParams.get("domain");
  if (session) {
    host.sessionKey = session;
    applySettings(host, {
      ...host.settings,
      sessionKey: session,
      lastActiveSessionKey: session,
    });
  }
  if (domain != null) {
    applySettings(host, { ...host.settings, opsDomain: normalizeOpsDomain(domain) });
  }

  setTabFromRoute(host, resolved);
}

export function setTabFromRoute(host: SettingsHost, next: Tab) {
  if (!canOpenTab(host, next)) {
    return;
  }

  const tabChanged = host.tab !== next;
  if (tabChanged) {
    host.tab = next;
    scrollContentToTop(host);
  }
  if (next === "hadoop" || next === "fi" || next === "gbase" || next === "governance" || next === "dataapps") {
    const domainSessionKey = `agent:main:ops:${next}`;
    if (host.sessionKey !== domainSessionKey) {
      host.sessionKey = domainSessionKey;
      host.applySessionKey = domainSessionKey;
    }
  }
  if (next === "chat") {
    host.chatHasAutoScrolled = false;
  }
  if (next === "logs") {
    startLogsPolling(host as unknown as Parameters<typeof startLogsPolling>[0]);
  } else {
    stopLogsPolling(host as unknown as Parameters<typeof stopLogsPolling>[0]);
  }
  if (next === "debug") {
    startDebugPolling(host as unknown as Parameters<typeof startDebugPolling>[0]);
  } else {
    stopDebugPolling(host as unknown as Parameters<typeof stopDebugPolling>[0]);
  }
  if (host.connected) {
    void refreshActiveTab(host);
  }
}

export function syncUrlWithTab(host: SettingsHost, tab: Tab, replace: boolean) {
  if (typeof window === "undefined") {
    return;
  }
  const targetPath = normalizePath(pathForTab(tab, host.basePath));
  const currentPath = normalizePath(window.location.pathname);
  const url = new URL(window.location.href);

  if ((tab === "chat" || tab === "message") && host.sessionKey) {
    url.searchParams.set("session", host.sessionKey);
  } else {
    url.searchParams.delete("session");
  }

  if (tab === "overview" || tab === "workbench" || tab === "assets") {
    const domain = normalizeOpsDomain(host.settings.opsDomain);
    if (domain === "all") {
      url.searchParams.delete("domain");
    } else {
      url.searchParams.set("domain", domain);
    }
  } else {
    url.searchParams.delete("domain");
  }

  if (currentPath !== targetPath) {
    url.pathname = targetPath;
  }

  if (replace) {
    window.history.replaceState({}, "", url.toString());
  } else {
    window.history.pushState({}, "", url.toString());
  }
}

export function syncUrlWithSessionKey(host: SettingsHost, sessionKey: string, replace: boolean) {
  if (typeof window === "undefined") {
    return;
  }
  const url = new URL(window.location.href);
  url.searchParams.set("session", sessionKey);
  if (replace) {
    window.history.replaceState({}, "", url.toString());
  } else {
    window.history.pushState({}, "", url.toString());
  }
}

export async function loadOverview(host: SettingsHost) {
  await Promise.all([
    loadChannels(host as unknown as OpenClawApp, false),
    loadPresence(host as unknown as OpenClawApp),
    loadSessions(host as unknown as OpenClawApp, { includeLastMessage: true }),
    loadCronStatus(host as unknown as OpenClawApp),
    loadDebug(host as unknown as OpenClawApp),
  ]);
}

export async function loadChannelsTab(host: SettingsHost) {
  await Promise.all([
    loadChannels(host as unknown as OpenClawApp, true),
    loadConfigSchema(host as unknown as OpenClawApp),
    loadConfig(host as unknown as OpenClawApp),
  ]);
}

export async function loadCron(host: SettingsHost) {
  await Promise.all([
    loadChannels(host as unknown as OpenClawApp, false),
    loadCronStatus(host as unknown as OpenClawApp),
    loadCronJobs(host as unknown as OpenClawApp),
  ]);
}
