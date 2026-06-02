import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { applyOpsDeepLinkFromUrl } from "./deeplink.ts";

describe("applyOpsDeepLinkFromUrl", () => {
  beforeEach(() => {
    window.history.replaceState({}, "", "/hadoop?opsSubTab=alerts&alertGroup=g-1");
  });

  afterEach(() => {
    window.history.replaceState({}, "", "/overview");
  });

  it("maps legacy alerts sub-tab to observability", () => {
    const host = {
      tab: "hadoop" as const,
      opsActiveSubTabs: {} as Record<string, string>,
      opsSelectedAlertGroupIds: {} as Record<string, string | null>,
    };
    const result = applyOpsDeepLinkFromUrl(host);
    expect(result.applied).toBe(true);
    expect(result.alertsTab).toBe(true);
    expect(host.opsActiveSubTabs.hadoop).toBe("observability");
    expect(host.opsSelectedAlertGroupIds.hadoop).toBe("g-1");
    expect(window.location.search).not.toContain("opsSubTab");
    expect(window.location.search).not.toContain("alertGroup");
  });

  it("accepts new capability sub-tab ids", () => {
    window.history.replaceState({}, "", "/hadoop?opsSubTab=inspection");
    const host = {
      tab: "hadoop" as const,
      opsActiveSubTabs: {},
      opsSelectedAlertGroupIds: {},
    };
    applyOpsDeepLinkFromUrl(host);
    expect(host.opsActiveSubTabs.hadoop).toBe("inspection");
  });

  it("no-ops on non-domain tabs", () => {
    window.history.replaceState({}, "", "/overview?opsSubTab=alerts");
    const host = {
      tab: "overview" as const,
      opsActiveSubTabs: {},
      opsSelectedAlertGroupIds: {},
    };
    expect(applyOpsDeepLinkFromUrl(host)).toEqual({ applied: false });
  });
});
