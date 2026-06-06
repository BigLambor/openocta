import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Tab } from "./navigation.ts";
const refreshChatMock = vi.hoisted(() => vi.fn(async () => undefined));
const loadLogsMock = vi.hoisted(() => vi.fn(async () => undefined));
vi.mock("./app-chat.ts", () => ({
  refreshChat: refreshChatMock,
}));
vi.mock("./controllers/logs.ts", () => ({
  loadLogs: loadLogsMock,
}));
import { setTab, setTabFromRoute, syncUrlWithTab } from "./app-settings.ts";

type SettingsHost = Parameters<typeof setTabFromRoute>[0] & {
  logsPollInterval: number | null;
  debugPollInterval: number | null;
  logsScrollFrame?: number | null;
  updateComplete?: Promise<void>;
  querySelector?: ReturnType<typeof vi.fn>;
};

const createHost = (tab: Tab, content?: HTMLElement): SettingsHost => ({
  settings: {
    gatewayUrl: "",
    token: "",
    sessionKey: "main",
    lastActiveSessionKey: "main",
    theme: "system",
    chatFocusMode: false,
    chatShowThinking: true,
    splitRatio: 0.6,
    navCollapsed: false,
    navGroupsCollapsed: {},
    opsDomain: "all",
  },
  theme: "system",
  themeResolved: "dark",
  applySessionKey: "main",
  sessionKey: "main",
  tab,
  connected: false,
  chatHasAutoScrolled: false,
  logsAtBottom: false,
  eventLog: [],
  eventLogBuffer: [],
  basePath: "",
  themeMedia: null,
  themeMediaHandler: null,
  logsPollInterval: null,
  debugPollInterval: null,
  logsScrollFrame: null,
  updateComplete: Promise.resolve(),
  querySelector: vi.fn((selector: string) => (selector === ".content" ? content ?? null : null)),
});

describe("syncUrlWithTab ops domain", () => {
  const originalUrl = "/chat";

  beforeEach(() => {
    window.history.replaceState({}, "", originalUrl);
  });

  it("omits domain query for all-domain workbench", () => {
    const host = createHost("workbench");
    host.settings.opsDomain = "all";

    syncUrlWithTab(host, "workbench", true);

    expect(window.location.pathname).toBe("/workbench");
    expect(window.location.search).toBe("");
  });

  it("preserves explicit technical domain query for workbench", () => {
    const host = createHost("workbench");
    host.settings.opsDomain = "hadoop";

    syncUrlWithTab(host, "workbench", true);

    expect(window.location.pathname).toBe("/workbench");
    expect(window.location.search).toBe("?domain=hadoop");
  });

  it("removes ops domain query outside ops-shell pages", () => {
    const host = createHost("chat");
    host.settings.opsDomain = "gbase";
    window.history.replaceState({}, "", "/workbench?domain=gbase");

    syncUrlWithTab(host, "chat", true);

    expect(window.location.pathname).toBe("/chat");
    expect(new URLSearchParams(window.location.search).has("domain")).toBe(false);
  });

  it("uses concrete domain path for domain insight", () => {
    const host = createHost("overview");
    host.settings.opsDomain = "hadoop";

    syncUrlWithTab(host, "domainInsight", true);

    expect(window.location.pathname).toBe("/overview/domain/hadoop");
  });

  it("never writes /overview/domain/all to the URL", () => {
    const host = createHost("overview");
    host.settings.opsDomain = "all";

    syncUrlWithTab(host, "domainInsight", true);

    expect(window.location.pathname).toBe("/overview/domain/hadoop");
  });
});

describe("setTabFromRoute", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts and stops log polling based on the tab", () => {
    const host = createHost("chat");

    setTabFromRoute(host, "logs");
    expect(host.logsPollInterval).not.toBeNull();
    expect(host.debugPollInterval).toBeNull();

    setTabFromRoute(host, "chat");
    expect(host.logsPollInterval).toBeNull();
  });

  it("starts and stops debug polling based on the tab", () => {
    const host = createHost("chat");

    setTabFromRoute(host, "debug");
    expect(host.debugPollInterval).not.toBeNull();
    expect(host.logsPollInterval).toBeNull();

    setTabFromRoute(host, "chat");
    expect(host.debugPollInterval).toBeNull();
  });

  it("scrolls content to top when setTab changes the route tab", () => {
    const content = document.createElement("main");
    content.className = "content";
    content.scrollTop = 240;
    const host = createHost("chat", content);

    setTab(host, "logs");

    expect(content.scrollTop).toBe(0);
  });

  it("does not scroll content when setTab targets the current tab", () => {
    const content = document.createElement("main");
    content.className = "content";
    content.scrollTop = 240;
    const host = createHost("logs", content);

    setTab(host, "logs");

    expect(host.querySelector).not.toHaveBeenCalledWith(".content");
  });

  it("scrolls content to top when route changes via setTabFromRoute", () => {
    const content = document.createElement("main");
    content.className = "content";
    content.scrollTop = 180;
    const host = createHost("chat", content);

    setTabFromRoute(host, "logs");

    expect(content.scrollTop).toBe(0);
  });

  it("normalizes all-domain context when opening domain insight", () => {
    const host = createHost("overview");
    host.settings.opsDomain = "all";

    setTab(host, "domainInsight");

    expect(host.tab).toBe("domainInsight");
    expect(host.settings.opsDomain).toBe("hadoop");
  });

  it("allows routing to overview before gateway connects", () => {
    const host = createHost("message");
    host.connected = false;

    setTabFromRoute(host, "overview");

    expect(host.tab).toBe("overview");
  });
});
