import { html, nothing } from "lit";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";

export type HealthDistribution = {
  healthy: number;
  warning: number;
  critical: number;
  unknown?: number;
  inactive?: number;
};

const STATUS_RANK: Record<string, number> = {
  critical: 0,
  warning: 1,
  unknown: 2,
  healthy: 3,
  inactive: 4,
};

export function distributionFromCounts(
  healthy = 0,
  warning = 0,
  critical = 0,
  unknown = 0,
  inactive = 0,
): HealthDistribution {
  return { healthy, warning, critical, unknown, inactive };
}

export function distributionFromClusters(clusters: OpsClusterRecord[]): HealthDistribution {
  const dist: HealthDistribution = { healthy: 0, warning: 0, critical: 0, unknown: 0, inactive: 0 };
  for (const cluster of clusters) {
    switch (cluster.status) {
      case "healthy":
        dist.healthy++;
        break;
      case "warning":
        dist.warning++;
        break;
      case "critical":
        dist.critical++;
        break;
      case "inactive":
        dist.inactive = (dist.inactive ?? 0) + 1;
        break;
      default:
        dist.unknown = (dist.unknown ?? 0) + 1;
        break;
    }
  }
  return dist;
}

export function distributionTotal(dist: HealthDistribution): number {
  return (
    dist.healthy +
    dist.warning +
    dist.critical +
    (dist.unknown ?? 0) +
    (dist.inactive ?? 0)
  );
}

/** Rollup score from cluster asset status when VictoriaMetrics health is unavailable. */
export function computeRollupHealthScore(dist: HealthDistribution): number | null {
  const total = distributionTotal(dist);
  if (total <= 0) {
    return null;
  }
  const weighted =
    dist.healthy * 100 +
    dist.warning * 72 +
    dist.critical * 35 +
    (dist.unknown ?? 0) * 60 +
    (dist.inactive ?? 0) * 50;
  return Math.round(weighted / total);
}

function segmentWidth(count: number, total: number): number {
  if (total <= 0 || count <= 0) {
    return 0;
  }
  return Math.max((count / total) * 100, count > 0 ? 2 : 0);
}

export function formatClusterCount(count: number): string {
  return count.toLocaleString("zh-CN");
}

export function clusterStatusLabel(status: string): string {
  switch (status) {
    case "healthy":
      return "健康";
    case "warning":
      return "亚健康";
    case "critical":
      return "异常";
    case "inactive":
      return "已下线";
    default:
      return "未知";
  }
}

export function clusterStatusTone(status: string): string {
  switch (status) {
    case "healthy":
      return "ok";
    case "warning":
      return "warning";
    case "critical":
      return "critical";
    default:
      return "muted";
  }
}

export function pickTopRiskClusters(clusters: OpsClusterRecord[], limit = 5): OpsClusterRecord[] {
  return [...clusters]
    .sort((a, b) => {
      const ra = STATUS_RANK[a.status] ?? 5;
      const rb = STATUS_RANK[b.status] ?? 5;
      if (ra !== rb) {
        return ra - rb;
      }
      return a.name.localeCompare(b.name, "zh-CN");
    })
    .slice(0, limit);
}

export function renderHealthDistributionBar(
  dist: HealthDistribution,
  opts: { compact?: boolean; emptyLabel?: string } = {},
) {
  const total = distributionTotal(dist);
  if (total === 0) {
    return html`
      <div class="health-dist-bar health-dist-bar--empty ${opts.compact ? "health-dist-bar--compact" : ""}">
        <span class="health-dist-bar__empty">${opts.emptyLabel ?? "暂无集群"}</span>
      </div>
    `;
  }

  const segments = [
    { key: "ok", count: dist.healthy },
    { key: "warn", count: dist.warning },
    { key: "critical", count: dist.critical },
    { key: "muted", count: dist.unknown ?? 0 },
    { key: "inactive", count: dist.inactive ?? 0 },
  ].filter((s) => s.count > 0);

  return html`
    <div
      class="health-dist-bar ${opts.compact ? "health-dist-bar--compact" : ""}"
      role="img"
      aria-label="健康 ${dist.healthy}，亚健康 ${dist.warning}，异常 ${dist.critical}"
    >
      ${segments.map(
        (seg) => html`
          <div
            class="health-dist-bar__seg health-dist-bar__seg--${seg.key}"
            style="width: ${segmentWidth(seg.count, total)}%;"
          ></div>
        `,
      )}
    </div>
  `;
}

export function renderHealthDistributionLegend(dist: HealthDistribution, opts: { compact?: boolean } = {}) {
  const total = distributionTotal(dist);
  if (total === 0) {
    return nothing;
  }
  const items = [
    { key: "ok", label: "健康", count: dist.healthy },
    { key: "warn", label: "亚健康", count: dist.warning },
    { key: "critical", label: "异常", count: dist.critical },
    { key: "muted", label: "未知", count: dist.unknown ?? 0 },
    { key: "inactive", label: "已下线", count: dist.inactive ?? 0 },
  ].filter((item) => item.count > 0);

  return html`
    <div class="health-dist-legend ${opts.compact ? "health-dist-legend--compact" : ""}">
      ${items.map(
        (item) => html`
          <span class="health-dist-legend__item health-dist-legend__item--${item.key}">
            <span class="health-dist-legend__dot"></span>
            ${formatClusterCount(item.count)} ${item.label}
          </span>
        `,
      )}
    </div>
  `;
}
