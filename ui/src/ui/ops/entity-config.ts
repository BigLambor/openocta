/** Ops context selector: built from cluster API (P1-3) with stable entity IDs. */

import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";

export type OpsDomainKey = "hadoop" | "fi" | "gbase" | "governance" | "dataapps";

export const OPS_DOMAIN_KEYS: OpsDomainKey[] = [
  "hadoop",
  "fi",
  "gbase",
  "governance",
  "dataapps",
];

export function isOpsDomainTab(tab: string): tab is OpsDomainKey {
  return (OPS_DOMAIN_KEYS as string[]).includes(tab);
}

export type OpsEntityOption = {
  id: string;
  label: string;
  indent?: boolean;
};

export type OpsEntityGroup = {
  groupLabel: string;
  options: OpsEntityOption[];
};

/** Domain-wide scope when multiple clusters exist. */
export const ENTITY_ID_DOMAIN_ALL = "all";

export function componentEntityId(clusterId: string, component: string): string {
  return `${clusterId}#${encodeURIComponent(component.trim())}`;
}

export function parseComponentEntityId(entityId: string): { clusterId: string; component: string } | null {
  const idx = entityId.indexOf("#");
  if (idx <= 0) {
    return null;
  }
  try {
    return {
      clusterId: entityId.slice(0, idx),
      component: decodeURIComponent(entityId.slice(idx + 1)),
    };
  } catch {
    return null;
  }
}

export function buildEntityGroupsFromClusters(clusters: OpsClusterRecord[]): OpsEntityGroup[] {
  if (clusters.length === 0) {
    return [];
  }
  const groups: OpsEntityGroup[] = [];
  if (clusters.length > 1) {
    groups.push({
      groupLabel: "业务域",
      options: [{ id: ENTITY_ID_DOMAIN_ALL, label: "全域视角（全部集群）" }],
    });
  }
  for (const c of clusters) {
    const header = c.region ? `${c.name} · ${c.region}` : c.name;
    const options: OpsEntityOption[] = [{ id: c.id, label: "集群全域视角" }];
    for (const comp of c.components ?? []) {
      const name = comp.trim();
      if (!name) {
        continue;
      }
      options.push({
        id: componentEntityId(c.id, name),
        label: name,
        indent: true,
      });
    }
    groups.push({ groupLabel: header, options });
  }
  return groups;
}

export function getDefaultEntityIdFromClusters(clusters: OpsClusterRecord[]): string {
  if (clusters.length === 0) {
    return ENTITY_ID_DOMAIN_ALL;
  }
  if (clusters.length > 1) {
    return ENTITY_ID_DOMAIN_ALL;
  }
  return clusters[0]!.id;
}

export function entityIdInGroups(groups: OpsEntityGroup[], entityId: string): boolean {
  for (const g of groups) {
    if (g.options.some((o) => o.id === entityId)) {
      return true;
    }
  }
  return false;
}

export function findCluster(clusters: OpsClusterRecord[], clusterId: string): OpsClusterRecord | undefined {
  return clusters.find((c) => c.id === clusterId);
}

/** Display title + subtitle for the context selector chip. */
export function formatEntityContextFromClusters(
  clusters: OpsClusterRecord[],
  entityId: string,
): { title: string; subtitle: string } {
  if (entityId === ENTITY_ID_DOMAIN_ALL) {
    return {
      title: "业务域全域",
      subtitle: `${clusters.length} 个集群`,
    };
  }
  const comp = parseComponentEntityId(entityId);
  if (comp) {
    const cluster = findCluster(clusters, comp.clusterId);
    const clusterLabel = cluster?.name ?? comp.clusterId;
    return {
      title: comp.component,
      subtitle: clusterLabel,
    };
  }
  const cluster = findCluster(clusters, entityId);
  if (cluster) {
    return {
      title: cluster.name,
      subtitle: cluster.region ? `${cluster.region} · 集群全域` : "集群全域视角",
    };
  }
  return { title: entityId, subtitle: "自定义上下文" };
}

/** Line prepended to Agent messages (P1-4). */
export function buildOpsContextLine(
  domainName: string,
  entityId: string,
  clusters: OpsClusterRecord[],
): string {
  const { title, subtitle } = formatEntityContextFromClusters(clusters, entityId);
  const clusterPart =
    entityId === ENTITY_ID_DOMAIN_ALL
      ? `clusters=${clusters.length}`
      : parseComponentEntityId(entityId)
        ? `cluster=${parseComponentEntityId(entityId)!.clusterId}`
        : `cluster=${entityId}`;
  return `[运维上下文] 业务域: ${domainName} | 目标: ${title} | ${subtitle} | ${clusterPart}`;
}

/** @deprecated Use getDefaultEntityIdFromClusters after clusters load. */
export function getDefaultEntityId(domain: OpsDomainKey): string {
  return ENTITY_ID_DOMAIN_ALL;
}

/** @deprecated Use buildEntityGroupsFromClusters. */
export function getEntityGroups(_domain: OpsDomainKey): OpsEntityGroup[] {
  return [];
}
