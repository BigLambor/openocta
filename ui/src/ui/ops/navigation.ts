import type { OpsDomainKey } from "./entity-config.ts";

export type TechOpsCapabilityTab =
  | "overview"
  | "assetTopology"
  | "observability"
  | "inspection"
  | "jobGovernance"
  | "diagnosis"
  | "governance"
  | "capacity"
  | "change"
  | "employees";

export const DEFAULT_TECH_OPS_CAPABILITY: TechOpsCapabilityTab = "overview";

const CAPABILITY_TABS = new Set<string>([
  "overview",
  "assetTopology",
  "observability",
  "inspection",
  "jobGovernance",
  "diagnosis",
  "governance",
  "capacity",
  "change",
  "employees",
]);

/** Legacy URL / stored values from pre–capability-domain navigation. */
const LEGACY_SUB_TAB_MAP: Record<string, TechOpsCapabilityTab> = {
  agent: "diagnosis",
  alerts: "observability",
  inspections: "inspection",
};

export function normalizeTechOpsCapabilityTab(
  raw: string | null | undefined,
): TechOpsCapabilityTab | null {
  if (!raw) {
    return null;
  }
  if (CAPABILITY_TABS.has(raw)) {
    return raw as TechOpsCapabilityTab;
  }
  return LEGACY_SUB_TAB_MAP[raw] ?? null;
}

export type OpenTechDomainPrefetch = {
  clusters?: boolean;
  alerts?: boolean;
};

export type OpenTechDomainOptions = {
  capabilityTab?: TechOpsCapabilityTab;
  prefetch?: OpenTechDomainPrefetch;
};

export type OpsDomainNavigationHost = {
  tab: string;
  opsActiveSubTabs: Record<string, TechOpsCapabilityTab | string>;
  setTab: (tab: import("../navigation.ts").Tab) => void | Promise<void>;
  loadOpsDomainClusters?: (domain: string) => Promise<void>;
  loadOpsDomainAlerts?: (domain: string) => Promise<void>;
};

export function ensureDefaultOpsCapabilityTab(
  host: Pick<OpsDomainNavigationHost, "opsActiveSubTabs">,
  domain: OpsDomainKey,
  capabilityTab: TechOpsCapabilityTab = DEFAULT_TECH_OPS_CAPABILITY,
): void {
  const current = normalizeTechOpsCapabilityTab(host.opsActiveSubTabs[domain]);
  if (!current) {
    host.opsActiveSubTabs = { ...host.opsActiveSubTabs, [domain]: capabilityTab };
  }
}

export async function openTechDomain(
  host: OpsDomainNavigationHost,
  domain: OpsDomainKey,
  opts: OpenTechDomainOptions = {},
): Promise<void> {
  const capabilityTab = opts.capabilityTab ?? DEFAULT_TECH_OPS_CAPABILITY;
  host.opsActiveSubTabs = { ...host.opsActiveSubTabs, [domain]: capabilityTab };

  const prefetch = {
    clusters: opts.prefetch?.clusters !== false,
    alerts: opts.prefetch?.alerts !== false,
  };

  const preload: Promise<void>[] = [];
  if (prefetch.clusters && host.loadOpsDomainClusters) {
    preload.push(host.loadOpsDomainClusters(domain));
  }
  if (prefetch.alerts && host.loadOpsDomainAlerts) {
    preload.push(host.loadOpsDomainAlerts(domain));
  }
  if (preload.length > 0) {
    void Promise.allSettled(preload);
  }

  await host.setTab(domain);
}
