import type { Tab } from "../navigation.ts";
import {
  normalizeTechOpsCapabilityTab,
  type TechOpsCapabilityTab,
} from "./navigation.ts";

const OPS_DOMAIN_TABS = new Set<Tab>(["hadoop", "fi", "gbase", "governance", "dataapps"]);

/** Apply ?opsSubTab= & ?alertGroup= deep links on ops domain routes (P2-C1). */
export function applyOpsDeepLinkFromUrl(host: {
  tab: Tab;
  opsActiveSubTabs: Record<string, TechOpsCapabilityTab | string>;
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

  const normalized = normalizeTechOpsCapabilityTab(sub);
  if (normalized) {
    host.opsActiveSubTabs = { ...host.opsActiveSubTabs, [host.tab]: normalized };
  } else if (group) {
    host.opsActiveSubTabs = { ...host.opsActiveSubTabs, [host.tab]: "observability" };
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
  const active = normalizeTechOpsCapabilityTab(host.opsActiveSubTabs[host.tab]);
  const alertsTab = active === "observability" || group != null;
  return { applied: true, alertsTab };
}
