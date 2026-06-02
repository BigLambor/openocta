import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { OpenClawApp } from "./app.ts";
import { GatewayBrowserClient } from "./gateway.ts";
import "../styles.css";

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

function nextFrame() {
  return new Promise<void>((resolve) => {
    requestAnimationFrame(() => resolve());
  });
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

describe("control UI routing", () => {
  it("hydrates the tab from the location", async () => {
    const app = mountApp("/sessions");
    await app.updateComplete;

    expect(app.tab).toBe("sessions");
    expect(window.location.pathname).toBe("/sessions");
  });

  it("respects /ui base paths", async () => {
    const app = mountApp("/ui/cron");
    await app.updateComplete;

    expect(app.basePath).toBe("/ui");
    expect(app.tab).toBe("cron");
    expect(window.location.pathname).toBe("/ui/cron");
  });

  it("infers nested base paths", async () => {
    const app = mountApp("/apps/openclaw/cron");
    await app.updateComplete;

    expect(app.basePath).toBe("/apps/openclaw");
    expect(app.tab).toBe("cron");
    expect(window.location.pathname).toBe("/apps/openclaw/cron");
  });

  it("honors explicit base path overrides", async () => {
    window.__OPENCLAW_CONTROL_UI_BASE_PATH__ = "/openclaw";
    const app = mountApp("/openclaw/sessions");
    await app.updateComplete;

    expect(app.basePath).toBe("/openclaw");
    expect(app.tab).toBe("sessions");
    expect(window.location.pathname).toBe("/openclaw/sessions");
  });

  it("updates the URL when clicking nav items", async () => {
    const app = mountApp("/overview");
    await app.updateComplete;

    const content = app.querySelector<HTMLElement>(".content");
    expect(content).not.toBeNull();
    if (!content) {
      return;
    }
    content.scrollTop = 320;

    const link = app.querySelector<HTMLButtonElement>('button[data-tour-tab="techDomains"]');
    expect(link).not.toBeNull();
    link?.dispatchEvent(new MouseEvent("click", { bubbles: true, cancelable: true, button: 0 }));

    await app.updateComplete;
    expect(app.tab).toBe("techDomains");
    expect(window.location.pathname).toBe("/tech-domains");
    expect(app.querySelector<HTMLElement>(".content")?.scrollTop).toBe(0);
  });

  it("highlights the active top tab for catalog routes", async () => {
    const cases = [
      ["/employee-market", "数字员工中心"],
      ["/skill-library", "技能库"],
      ["/tool-library", "工具库"],
      ["/model-library", "模型"],
      ["/tutorials", "教程"],
    ] as const;

    for (const [pathname, expected] of cases) {
      document.body.innerHTML = "";
      const app = mountApp(pathname);
      await app.updateComplete;

      const activeTab = app.querySelector(".top-tab--active") || app.querySelector(".dropdown-item.active");
      expect(activeTab?.textContent).toContain(expected);
      expect(window.location.pathname).toBe(pathname);
    }
  });

  it("renders the model tab before tutorials and removes the community tab", async () => {
    const app = mountApp("/message");
    await app.updateComplete;

    const labels = Array.from(app.querySelectorAll(".dropdown-menu-content .dropdown-item")).map((node) =>
      node.querySelector("span:not(.dropdown-icon)")?.textContent?.trim() ?? "",
    );

    expect(labels).toContain("模型");
    expect(labels).not.toContain("社区");
    expect(labels.indexOf("模型")).toBeGreaterThan(-1);
    expect(labels.indexOf("教程")).toBeGreaterThan(-1);
    expect(labels.indexOf("模型")).toBeLessThan(labels.indexOf("教程"));
  });

  it("auto-scrolls chat history to the latest message", async () => {
    const app = mountApp("/chat");
    await app.updateComplete;

    const initialContainer: HTMLElement | null = app.querySelector(".chat-thread");
    expect(initialContainer).not.toBeNull();
    if (!initialContainer) {
      return;
    }
    initialContainer.style.maxHeight = "180px";
    initialContainer.style.overflow = "auto";

    app.chatMessages = Array.from({ length: 60 }, (_, index) => ({
      role: "assistant",
      content: `Line ${index} - ${"x".repeat(200)}`,
      timestamp: Date.now() + index,
    }));

    await app.updateComplete;
    for (let i = 0; i < 6; i++) {
      await nextFrame();
    }

    const container = app.querySelector(".chat-thread");
    expect(container).not.toBeNull();
    if (!container) {
      return;
    }
    const maxScroll = container.scrollHeight - container.clientHeight;
    expect(maxScroll).toBeGreaterThan(0);
    for (let i = 0; i < 10; i++) {
      if (container.scrollTop === maxScroll) {
        break;
      }
      await nextFrame();
    }
    expect(container.scrollTop).toBe(maxScroll);
  });

  it("strips token URL params without importing them", async () => {
    const app = mountApp("/ui/overview?token=abc123");
    await app.updateComplete;

    expect(window.location.pathname).toBe("/ui/overview");
    expect(window.location.search).toBe("");
  });

  it("strips password URL params without importing them", async () => {
    const app = mountApp("/ui/overview?password=sekret");
    await app.updateComplete;

    expect(app.password).toBe("sekret");
    expect(window.location.pathname).toBe("/ui/overview");
    expect(window.location.search).toBe("");
  });

  it("does not override stored settings from URL token params", async () => {
    localStorage.setItem(
      "openclaw.control.settings.v1",
      JSON.stringify({ token: "existing-token" }),
    );
    const app = mountApp("/ui/overview?token=abc123");
    await app.updateComplete;

    expect(app.settings.token).toBe("existing-token");
    expect(window.location.pathname).toBe("/ui/overview");
    expect(window.location.search).toBe("");
  });
});
