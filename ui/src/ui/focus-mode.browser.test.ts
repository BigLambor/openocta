import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { OpenClawApp } from "./app.ts";
import { GatewayBrowserClient } from "./gateway.ts";

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

describe("chat focus mode", () => {
  it("collapses header + sidebar on chat tab only", async () => {
    const app = mountApp("/chat");
    await app.updateComplete;

    const shell = app.querySelector(".shell");
    expect(shell).not.toBeNull();
    expect(shell?.classList.contains("shell--chat-focus")).toBe(false);

    const toggle = app.querySelector<HTMLButtonElement>('button[title^="Toggle focus mode"]');
    expect(toggle).not.toBeNull();
    toggle?.click();

    await app.updateComplete;
    expect(shell?.classList.contains("shell--chat-focus")).toBe(true);

    const overviewTab = Array.from(app.querySelectorAll<HTMLButtonElement>("button.top-tab")).find((button) =>
      button.textContent?.includes("驾驶"),
    );
    expect(overviewTab).not.toBeUndefined();
    overviewTab?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 }));

    await app.updateComplete;
    expect(app.tab).toBe("overview");
    expect(shell?.classList.contains("shell--chat-focus")).toBe(false);

    const messageTab = Array.from(app.querySelectorAll<HTMLButtonElement>("button.top-tab")).find((button) =>
      button.textContent?.includes("助手"),
    );
    messageTab?.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 }),
    );

    await app.updateComplete;
    expect(app.tab).toBe("message");
    expect(shell?.classList.contains("shell--chat-focus")).toBe(true);
  });
});
