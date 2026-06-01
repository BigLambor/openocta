import type { Tab } from "../navigation.ts";

const OPS_DOMAIN_TABS = new Set<Tab>(["hadoop", "fi", "gbase", "governance", "dataapps"]);

/** Apply ?opsSubTab= & ?alertGroup= deep links on ops domain routes (P2-C1). */
export function applyOpsDeepLinkFromUrl(host: {
  tab: Tab;
  opsActiveSubTabs: Record<string, "agent" | "alerts" | "inspections">;
  opsSelectedAlertGroupIds: Record<string, string | null>;
}): { applied: boolean; alertsTab?: boolean } {
  if (typeof window === "undefined" || !OPS_DOMAIN_TABS.has(host.tab)) {
    return { applied: false };
  }
  const params = new URLSearchParams(window.location.search);
  const sub = params.get("opsSubTab");
  const group = params.get("alertGroup");
  if (!sub && !group) {
    return { applied: false };
  }

  if (sub === "agent" || sub === "alerts" || sub === "inspections") {
    host.opsActiveSubTabs = { ...host.opsActiveSubTabs, [host.tab]: sub };
  }
  if (group) {
    host.opsSelectedAlertGroupIds = {
      ...host.opsSelectedAlertGroupIds,
      [host.tab]: group,
    };
  }

  params.delete("opsSubTab");
  params.delete("alertGroup");
  const url = new URL(window.location.href);
  url.search = params.toString();
  window.history.replaceState({}, "", url.toString());
  const alertsTab = sub === "alerts" || group != null;
  return { applied: true, alertsTab };
}
