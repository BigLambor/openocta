import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { OpenClawApp } from "./app";
import { GatewayBrowserClient } from "./gateway";

// oxlint-disable-next-line typescript/unbound-method
const originalConnect = OpenClawApp.prototype.connect;
// oxlint-disable-next-line typescript/unbound-method
const originalCheckRbacSession = OpenClawApp.prototype.checkRbacSession;
// oxlint-disable-next-line typescript/unbound-method
const originalStart = GatewayBrowserClient.prototype.start;

function mountApp(pathname: string) {
  window.history.replaceState({}, "", pathname);
  const app = document.createElement("openclaw-app") as OpenClawApp;
  document.body.append(app);
  return app;
}

beforeEach(() => {
  OpenClawApp.prototype.connect = () => {
    // no-op: avoid real gateway WS connections in browser tests
  };
  OpenClawApp.prototype.checkRbacSession = async function() {
    this.rbacUser = {
      userId: 1,
      username: "admin",
      roleName: "admin",
      permissions: ["menu:chat", "menu:sessions", "menu:overview", "menu:cron", "menu:config"]
    };
    this.rbacChecked = true;
  };
  GatewayBrowserClient.prototype.start = function() {
    // no-op: prevent WebSocket creation in tests
  };
  window.__OPENCLAW_CONTROL_UI_BASE_PATH__ = undefined;
  localStorage.clear();
  document.body.innerHTML = "";
});

afterEach(() => {
  OpenClawApp.prototype.connect = originalConnect;
  OpenClawApp.prototype.checkRbacSession = originalCheckRbacSession;
  GatewayBrowserClient.prototype.start = originalStart;
  window.__OPENCLAW_CONTROL_UI_BASE_PATH__ = undefined;
  localStorage.clear();
  document.body.innerHTML = "";
});

describe("chat markdown rendering", () => {
  it("renders markdown inside tool output sidebar", async () => {
    const app = mountApp("/chat");
    await app.updateComplete;
    app.chatConversationOnly = false;

    const timestamp = Date.now();
    app.chatMessages = [
      {
        role: "assistant",
        content: [
          { type: "toolcall", name: "noop", arguments: {} },
          { type: "toolresult", name: "noop", text: "Hello **world**" },
        ],
        timestamp,
      },
    ];

    await app.updateComplete;

    const openSidebarBtn = app.querySelector<HTMLElement>(".chat-tool-run__open-sidebar");
    expect(openSidebarBtn).not.toBeNull();
    openSidebarBtn?.click();

    await app.updateComplete;

    const strong = app.querySelector(".sidebar-markdown strong");
    expect(strong?.textContent).toBe("world");
  });
});
