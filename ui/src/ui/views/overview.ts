import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { renderOpsEmpty, renderOpsError, renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsDashboardSummary, OpsClusterRecord } from "../controllers/ops-clusters.ts";
import "../components/ops-map.ts";
import {
  computeRollupHealthScore,
  distributionFromCounts,
  formatClusterCount,
  renderHealthDistributionBar,
  renderHealthDistributionLegend,
  type HealthDistribution,
} from "../components/ops-health-distribution.ts";
import {
  DOMAIN_TABLE_THRESHOLD,
  type DashboardAlertHighlight,
  type DashboardInspectionRun,
} from "../controllers/ops-dashboard-feed.ts";
import type { Tab } from "../navigation.ts";

export type OverviewDomainCard = {
  tab: Tab;
  name: string;
  icon: "network" | "building" | "database" | "layout" | "activity";
  score?: number;
  healthScoreSource?: string;
  healthScoreNote?: string;
  scoreStatus?: string;
  coverage?: number | null;
  missingSources?: string[];
  presentSources?: string[];
  clusterCount?: number;
  healthyCount?: number;
  warningCount?: number;
  criticalCount?: number;
  pendingAlerts?: number;
  note?: string;
};

export type OverviewStats = {
  totalClusters: number;
  healthyClusters: number;
  warningClusters: number;
  pendingAlerts: number;
};

export type OverviewProps = {
  connected?: boolean;
  loading?: boolean;
  clusters?: OpsClusterRecord[];
  snapshots?: any[];
  dashboardSummary?: OpsDashboardSummary | null;
  dashboardError?: string | null;
  stats?: OverviewStats | null;
  domains?: OverviewDomainCard[] | null;
  globalInspecting?: boolean;
  dashboardToast?: string | null;
  onOpenAssets?: () => void;
  onOpenConfig?: () => void;
  onNavigateDomain?: (tab: Tab) => void;
  onOpenDomainAssets?: (tab: Tab) => void;
  onOpenScheduledTasks?: () => void;
  onRunGlobalInspection?: () => void;
  onOpenPendingAlerts?: () => void;
  onOpenDomainAlerts?: (domain: string) => void;
  onReloadFeed?: () => void;
  feedLoading?: boolean;
  feedError?: string | null;
  alertHighlights?: DashboardAlertHighlight[];
  domainPendingAlerts?: Record<string, number>;
  recentInspections?: DashboardInspectionRun[];
  canInspect?: boolean;
};

const DOMAIN_TAB: Record<string, Tab> = {
  hadoop: "hadoop",
  fi: "fi",
  gbase: "gbase",
  governance: "governance",
  dataapps: "dataapps",
};

const DOMAIN_ICON: Record<string, OverviewDomainCard["icon"]> = {
  hadoop: "network",
  fi: "building",
  gbase: "database",
  governance: "layout",
  dataapps: "activity",
};

const DOMAIN_PLACEHOLDERS: OverviewDomainCard[] = [
  { tab: "hadoop", name: "BCH生态", icon: "network" },
  { tab: "fi", name: "FI商业生态", icon: "building" },
  { tab: "gbase", name: "GBase数据库", icon: "database" },
  { tab: "governance", name: "开发治理平台", icon: "layout" },
  { tab: "dataapps", name: "数据App运维", icon: "activity" },
];

const DOMAIN_DISPLAY_NAME: Record<string, string> = Object.fromEntries(
  DOMAIN_PLACEHOLDERS.map((d) => [d.tab, d.name]),
);

function summaryToCards(summary: OpsDashboardSummary): {
  stats: OverviewStats;
  domains: OverviewDomainCard[];
} {
  return {
    stats: {
      totalClusters: summary.totalClusters,
      healthyClusters: summary.healthyClusters,
      warningClusters: summary.warningClusters,
      pendingAlerts: summary.pendingAlerts,
    },
    domains: summary.domains.map((d) => {
      const score =
        d.healthScore != null && Number.isFinite(d.healthScore) ? Math.round(d.healthScore) : undefined;
      const tab = DOMAIN_TAB[d.domain] ?? "hadoop";
      const note =
        d.note ||
        (d.clusterCount > 0 && score == null && d.healthScoreNote ? d.healthScoreNote : undefined);
      return {
        tab,
        name: DOMAIN_DISPLAY_NAME[tab] ?? d.domain,
        icon: DOMAIN_ICON[d.domain] ?? "network",
        score,
        healthScoreSource: d.healthScoreSource,
        healthScoreNote: d.healthScoreNote,
        scoreStatus: d.scoreStatus,
        coverage: d.coverage,
        missingSources: d.missingSources ?? [],
        presentSources: d.presentSources ?? [],
        clusterCount: d.clusterCount,
        healthyCount: d.healthyCount,
        warningCount: d.warningCount,
        criticalCount: d.criticalCount,
        note,
      };
    }),
  };
}

function iconForDomain(icon: OverviewDomainCard["icon"]) {
  return icons[icon];
}

function scoreClass(score: number | undefined | null): string {
  if (score == null) {
    return "muted";
  }
  if (score >= 90) {
    return "ok";
  }
  if (score >= 75) {
    return "warning";
  }
  return "critical";
}

function domainDistribution(d: OverviewDomainCard): HealthDistribution {
  return distributionFromCounts(d.healthyCount ?? 0, d.warningCount ?? 0, d.criticalCount ?? 0);
}

function effectiveDomainScore(d: OverviewDomainCard): {
  value: number | null;
  source: "composite" | "vm" | "rollup" | "none";
} {
  if (d.score != null) {
    return { value: d.score, source: d.healthScoreSource === "composite" ? "composite" : "vm" };
  }
  if (d.healthScoreSource === "composite") {
    return { value: null, source: "composite" };
  }
  const rollup = computeRollupHealthScore(domainDistribution(d));
  if (rollup != null) {
    return { value: rollup, source: "rollup" };
  }
  return { value: null, source: "none" };
}

function domainStatusTone(
  d: OverviewDomainCard,
  pending: number,
): "ok" | "warning" | "critical" | "muted" {
  if ((d.criticalCount ?? 0) > 0 || pending > 0) {
    return "critical";
  }
  if ((d.warningCount ?? 0) > 0) {
    return "warning";
  }
  if (d.scoreStatus === "degraded") {
    return "critical";
  }
  if (d.scoreStatus === "partial") {
    return "warning";
  }
  const { value } = effectiveDomainScore(d);
  if (value != null && value < 75) {
    return "critical";
  }
  if (value != null && value < 90) {
    return "warning";
  }
  return "ok";
}

function renderDomainScore(d: OverviewDomainCard) {
  const { value, source } = effectiveDomainScore(d);
  const cls = scoreClass(value);
  if (source === "composite" && value == null) {
    const status = d.scoreStatus || "unknown";
    const label = status === "degraded" ? "降级" : status === "partial" ? "部分覆盖" : "待评分";
    return html`
      <div class="domain-score-block" title=${l3HealthTitle(d)}>
        <span class="domain-score ${status === "degraded" ? "critical" : "warning"}">${label}</span>
        ${renderCoverageBadge(d)}
      </div>
    `;
  }
  if (value == null) {
    return html`
      <span class="domain-score domain-score--muted" title="尚未纳管集群或未返回健康分">待评分</span>
    `;
  }
  return html`
    <div class="domain-score-block" title=${scoreSourceTitle(source)}>
      <span class="domain-score ${cls}">${value}分</span>
      <span class="domain-score__source domain-score__source--${source}">
        ${scoreSourceLabel(source)}
      </span>
      ${source === "composite" ? renderCoverageBadge(d) : nothing}
    </div>
  `;
}

function scoreSourceLabel(source: "composite" | "vm" | "rollup" | "none"): string {
  switch (source) {
    case "composite":
      return "综合";
    case "vm":
      return "监控";
    case "rollup":
      return "资产";
    default:
      return "";
  }
}

function scoreSourceTitle(source: "composite" | "vm" | "rollup" | "none"): string {
  switch (source) {
    case "composite":
      return "L3 Facts 多源综合健康分";
    case "vm":
      return "VictoriaMetrics / Prometheus 监控健康分";
    case "rollup":
      return "按集群资产状态聚合估算";
    default:
      return "";
  }
}

function renderCoverageBadge(d: OverviewDomainCard) {
  if (d.coverage == null || !Number.isFinite(d.coverage)) {
    return nothing;
  }
  return html`<span class="domain-score__source domain-score__source--coverage">${Math.round(d.coverage * 100)}%</span>`;
}

function l3HealthTitle(d: OverviewDomainCard): string {
  const parts: string[] = [];
  if (d.healthScoreNote) {
    parts.push(d.healthScoreNote);
  }
  if (d.missingSources?.length) {
    parts.push(`缺失源: ${d.missingSources.join(", ")}`);
  }
  if (d.presentSources?.length) {
    parts.push(`已有源: ${d.presentSources.join(", ")}`);
  }
  return parts.join("；") || "L3 Facts 暂无完整综合分";
}

function domainStatusLabel(d: OverviewDomainCard): string {
  if ((d.clusterCount ?? 0) === 0) {
    return "尚未纳管";
  }
  if (d.healthScoreSource === "composite" && d.scoreStatus === "degraded") {
    return d.missingSources?.length ? `必需源缺失：${d.missingSources.join(", ")}` : "必需源缺失或失败";
  }
  if (d.healthScoreSource === "composite" && d.scoreStatus === "partial") {
    return d.missingSources?.length ? `覆盖不足：缺 ${d.missingSources.join(", ")}` : "覆盖不足";
  }
  if (d.note) {
    return d.note;
  }
  if ((d.criticalCount ?? 0) > 0) {
    return `${d.criticalCount} 个严重`;
  }
  if ((d.warningCount ?? 0) > 0) {
    return `${d.warningCount} 个亚健康`;
  }
  if (d.score != null && d.score < 90) {
    return "需关注";
  }
  return "运行平稳";
}

function partitionDomains(
  domains: OverviewDomainCard[],
  pendingByDomain?: Record<string, number>,
): {
  managed: OverviewDomainCard[];
  unmanaged: OverviewDomainCard[];
  attention: OverviewDomainCard[];
  stable: OverviewDomainCard[];
} {
  const managed = domains.filter((d) => (d.clusterCount ?? 0) > 0);
  const unmanaged = domains.filter((d) => (d.clusterCount ?? 0) === 0);
  managed.sort((a, b) => {
    const sa = effectiveDomainScore(a).value ?? 101;
    const sb = effectiveDomainScore(b).value ?? 101;
    if (sa !== sb) {
      return sa - sb;
    }
    const riskA = (a.criticalCount ?? 0) * 10 + (a.warningCount ?? 0);
    const riskB = (b.criticalCount ?? 0) * 10 + (b.warningCount ?? 0);
    if (riskA !== riskB) {
      return riskB - riskA;
    }
    return a.name.localeCompare(b.name, "zh-CN");
  });
  const attention = managed.filter((d) => {
    const pending = pendingAlertsForDomain(d, pendingByDomain);
    const { value } = effectiveDomainScore(d);
    return (
      pending > 0 ||
      (d.warningCount ?? 0) > 0 ||
      (d.criticalCount ?? 0) > 0 ||
      d.scoreStatus === "degraded" ||
      d.scoreStatus === "partial" ||
      (value != null && value < 90)
    );
  });
  const attentionTabs = new Set(attention.map((d) => d.tab));
  const stable = managed.filter((d) => !attentionTabs.has(d.tab));
  return { managed, unmanaged, attention, stable };
}

function pendingAlertsForDomain(d: OverviewDomainCard, pendingByDomain?: Record<string, number>): number {
  const key = String(d.tab);
  if (pendingByDomain && key in pendingByDomain) {
    return pendingByDomain[key] ?? 0;
  }
  return d.pendingAlerts ?? 0;
}

function renderDomainTable(
  domains: OverviewDomainCard[],
  props: OverviewProps,
  pendingByDomain?: Record<string, number>,
) {
  return html`
    <div class="ops-panel domain-table-wrap">
      <table class="domain-table">
        <thead>
          <tr>
            <th scope="col">业务域</th>
            <th scope="col">健康分</th>
            <th scope="col">集群</th>
            <th scope="col">健康分布</th>
            <th scope="col">待处理告警</th>
            <th scope="col" class="domain-table__actions-col">操作</th>
          </tr>
        </thead>
        <tbody>
          ${domains.map((d) => {
            const pending = pendingAlertsForDomain(d, pendingByDomain);
            const dist = domainDistribution(d);
            const display = effectiveDomainScore(d);
            const cls = scoreClass(display.value);
            return html`
              <tr
                class="domain-table__row"
                @click=${(e: Event) => {
                  const target = e.target as HTMLElement;
                  if (target.closest("button")) {
                    return;
                  }
                  props.onNavigateDomain?.(d.tab);
                }}
              >
                <td>
                  <div class="domain-table__name">
                    <span class="domain-icon-wrapper">${iconForDomain(d.icon)}</span>
                    <span>${d.name}</span>
                  </div>
                </td>
                <td>
                  ${display.value != null
                    ? html`
                        <span class="domain-score ${cls}">${display.value}分</span>
                        <span class="domain-score__source domain-score__source--${display.source}">
                          ${scoreSourceLabel(display.source)}
                        </span>
                        ${display.source === "composite" ? renderCoverageBadge(d) : nothing}
                      `
                    : display.source === "composite"
                      ? html`
                          <span
                            class="domain-score ${d.scoreStatus === "degraded" ? "critical" : "warning"}"
                            title=${l3HealthTitle(d)}
                          >
                            ${d.scoreStatus === "degraded" ? "降级" : d.scoreStatus === "partial" ? "部分覆盖" : "待评分"}
                          </span>
                          ${renderCoverageBadge(d)}
                        `
                      : html`<span class="domain-score domain-score--muted">待评分</span>`}
                </td>
                <td>${formatClusterCount(d.clusterCount ?? 0)}</td>
                <td class="domain-table__dist-col">
                  ${renderHealthDistributionBar(dist, { compact: true })}
                  ${renderHealthDistributionLegend(dist, { compact: true })}
                </td>
                <td>
                  <span class=${pending > 0 ? "domain-table__alerts domain-table__alerts--active" : "domain-table__alerts"}>
                    ${pending}
                  </span>
                </td>
                <td class="domain-table__actions-col">
                  <button
                    type="button"
                    class="ops-btn ops-btn--ghost domain-table__action"
                    @click=${() => props.onNavigateDomain?.(d.tab)}
                  >
                    详情
                  </button>
                </td>
              </tr>
            `;
          })}
        </tbody>
      </table>
    </div>
  `;
}

function openDomainFromCard(d: OverviewDomainCard, managed: boolean, props: OverviewProps) {
  if (managed) {
    props.onNavigateDomain?.(d.tab);
  } else {
    props.onOpenDomainAssets?.(d.tab);
  }
}

function renderDomainCard(
  d: OverviewDomainCard,
  props: OverviewProps,
  opts: { compact?: boolean; placeholder?: boolean } = {},
) {
  const managed = !opts.placeholder && (d.clusterCount ?? 0) > 0;
  const statusLabel = opts.placeholder || !managed ? "尚未纳管，待登记集群" : domainStatusLabel(d);
  const dist = domainDistribution(d);
  const pending = pendingAlertsForDomain(d, props.domainPendingAlerts);
  const statusTone = managed ? domainStatusTone(d, pending) : "muted";
  const display = effectiveDomainScore(d);

  return html`
    <article
      class="domain-card ${!managed ? "domain-card--muted" : ""} ${display.value != null && display.value < 90
        ? "domain-card--attention"
        : ""} domain-card--tone-${statusTone} ${opts.compact
        ? "domain-card--compact"
        : ""}"
      role="button"
      tabindex="0"
      @click=${(e: Event) => {
        const target = e.target as HTMLElement;
        if (target.closest("button, a")) {
          return;
        }
        openDomainFromCard(d, managed, props);
      }}
      @keydown=${(e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          openDomainFromCard(d, managed, props);
        }
      }}
    >
      <div class="domain-card-header">
        <div class="domain-name">
          <span class="domain-icon-wrapper">${iconForDomain(d.icon)}</span>
          <span class="domain-name__text">${d.name}</span>
        </div>
        ${renderDomainScore(d)}
      </div>
      ${managed
        ? html`
            <div class="domain-card__distribution">
              ${renderHealthDistributionBar(dist)}
              ${renderHealthDistributionLegend(dist, { compact: true })}
            </div>
          `
        : nothing}
      <div class="domain-stats">
        <span>${managed ? `${formatClusterCount(d.clusterCount ?? 0)} 个集群` : "0 个集群"}</span>
        <span class="domain-stats__status domain-stats__status--${statusTone}">
          ${pending > 0 ? `${pending} 组告警` : statusLabel}
        </span>
      </div>
      ${managed && props.onNavigateDomain
        ? html`
            <div class="domain-card-actions">
              <button
                type="button"
                class="ops-btn ops-btn--ghost domain-card-action"
                @click=${() => props.onNavigateDomain!(d.tab)}
              >
                域详情 ${icons.chevronRight}
              </button>
              ${props.onOpenDomainAssets
                ? html`
                    <button
                      type="button"
                      class="ops-btn ops-btn--ghost domain-card-action"
                      @click=${() => props.onOpenDomainAssets!(d.tab)}
                    >
                      资产
                    </button>
                  `
                : nothing}
            </div>
          `
        : props.onOpenDomainAssets && !managed
          ? html`
              <div class="domain-card-actions">
                <button
                  type="button"
                  class="ops-btn ops-btn--primary domain-card-action domain-card-action--primary"
                  @click=${() => props.onOpenDomainAssets!(d.tab)}
                >
                  登记集群
                </button>
              </div>
            `
          : nothing}
    </article>
  `;
}

function renderUnmanagedInline(domains: OverviewDomainCard[], props: OverviewProps) {
  if (domains.length === 0) {
    return nothing;
  }
  const names = domains.map((d) => d.name).join("、");
  return html`
    <div class="domain-pending-inline">
      <span class="domain-pending-inline__text">
        待接入 ${domains.length} 个域：${names}
      </span>
      ${props.onOpenAssets
        ? html`
            <button type="button" class="ops-btn ops-btn--ghost domain-pending-inline__action" @click=${props.onOpenAssets}>
              去登记
            </button>
          `
        : nothing}
    </div>
  `;
}

function renderRecentActivityFeed(props: OverviewProps, stats: OverviewStats | null) {
  const highlights = props.alertHighlights ?? [];
  const runs = props.recentInspections ?? [];
  const loading = props.feedLoading && highlights.length === 0 && runs.length === 0;
  const pending = stats?.pendingAlerts ?? 0;

  return html`
    <section class="ops-dashboard-panel ops-dashboard-panel--feed">
      <h2 class="section-title">
        <span class="section-title__icon">${icons.historyClock}</span>
        最近动态
      </h2>
      <div class="ops-panel ops-dashboard-panel__body">
        ${props.feedError
          ? html`
              <p class="ops-dashboard-panel__empty">${props.feedError}</p>
              ${props.onReloadFeed
                ? html`
                    <button type="button" class="ops-btn ops-dashboard-panel__action" @click=${props.onReloadFeed}>
                      重新加载
                    </button>
                  `
                : nothing}
            `
          : loading
            ? renderOpsSkeleton({ lines: 4 })
            : highlights.length === 0 && runs.length === 0
              ? html`
                  <p class="ops-dashboard-panel__empty">
                    ${pending > 0
                      ? `当前有 ${pending} 组待处理告警，建议优先查看。`
                      : "各业务域运行平稳，暂无新的告警或巡检动态。"}
                  </p>
                `
              : html`
                  <ul class="ops-feed-list">
                    ${highlights.slice(0, 3).map(
                      (item) => html`
                        <li class="ops-feed-item">
                          <button
                            type="button"
                            class="ops-feed-item__btn"
                            @click=${() => props.onOpenDomainAlerts?.(item.domain)}
                          >
                            <span class="ops-feed-item__head">
                              <span class="ops-feed-item__type">告警</span>
                              <span class="ops-feed-severity ops-feed-severity--${item.severity}">
                                ${item.severity === "critical" ? "严重" : item.severity === "warning" ? "警告" : "信息"}
                              </span>
                              <span class="ops-feed-item__domain">${item.domainLabel}</span>
                              <span class="ops-feed-item__time">${item.timestamp}</span>
                            </span>
                            <span class="ops-feed-item__title">${item.title}</span>
                          </button>
                        </li>
                      `,
                    )}
                    ${runs.slice(0, 2).map(
                      (run) => html`
                        <li class="ops-feed-item">
                          <button
                            type="button"
                            class="ops-feed-item__btn"
                            @click=${() => props.onOpenScheduledTasks?.()}
                          >
                            <span class="ops-feed-item__head">
                              <span class="ops-feed-item__type">巡检</span>
                              <span class="ops-feed-item__domain">${run.domainLabel}</span>
                              <span class="ops-feed-item__score ops-feed-item__score--${run.status}">
                                ${run.score != null ? `${run.score}分` : inspectionStatusLabel(run.status)}
                              </span>
                              <span class="ops-feed-item__time">${run.time}</span>
                            </span>
                            <span class="ops-feed-item__title">${run.summary}</span>
                          </button>
                        </li>
                      `,
                    )}
                  </ul>
                `}
        ${pending > 0 && props.onOpenPendingAlerts
          ? html`
              <button
                type="button"
                class="ops-btn ops-dashboard-panel__action"
                @click=${() => props.onOpenPendingAlerts?.()}
              >
                查看 ${pending} 组待处理告警
              </button>
            `
          : nothing}
      </div>
    </section>
  `;
}

function inspectionStatusLabel(status: DashboardInspectionRun["status"]): string {
  switch (status) {
    case "healthy":
      return "健康";
    case "warning":
      return "亚健康";
    case "critical":
      return "风险";
    case "error":
      return "失败";
    default:
      return "未知";
  }
}

export function renderOverview(props: OverviewProps) {
  const fromApi = props.dashboardSummary ? summaryToCards(props.dashboardSummary) : null;
  const stats = fromApi?.stats ?? props.stats ?? null;
  const hasStats = stats != null && (stats.totalClusters > 0 || fromApi != null);
  const domains =
    fromApi?.domains ??
    (props.domains?.length ? props.domains : hasStats && stats && stats.totalClusters > 0 ? [] : null);
  const showEmptyMetrics = fromApi != null && stats != null && stats.totalClusters === 0;
  const pendingByDomain = props.domainPendingAlerts;
  const partitioned = domains ? partitionDomains(domains, pendingByDomain) : null;
  const useDomainTable = (partitioned?.managed.length ?? 0) >= DOMAIN_TABLE_THRESHOLD;
  const awaitingData =
    props.loading || (!props.dashboardSummary && !props.dashboardError);

  return html`
    <div class="ops-page ops-dashboard">
      <div class="ops-page-header ops-dashboard-header ops-dashboard-header--split">
        <div>
          <h1>运维全局视图</h1>
          <p>
            ${stats && stats.totalClusters > 0
              ? `已纳管 ${stats.totalClusters} 个集群，覆盖 ${domains?.filter((d) => (d.clusterCount ?? 0) > 0).length ?? 0} 个有资产业务域。`
              : fromApi != null
                ? "尚未登记集群资产，请点击「接入业务域」或顶部导航「服务与资产」进行登记。"
                : "接入集群资产与 VictoriaMetrics 后，将在此展示纳管规模、健康度与告警概况。"}
          </p>
        </div>
        <div class="ops-dashboard-header__actions">
          <button
            type="button"
            class="ops-btn"
            title="在「服务与资产」中登记集群，接入新业务域"
            @click=${() => props.onOpenAssets?.()}
          >
            接入业务域
          </button>
          <button
            type="button"
            class="ops-btn ops-btn--primary"
            ?disabled=${!props.onRunGlobalInspection || !props.connected || props.globalInspecting || props.canInspect === false}
            title=${props.canInspect === false
              ? "当前账号无 ops:inspect 权限"
              : !props.connected
                ? "请先连接网关"
                : "并行触发各业务域的深度健康巡检任务"}
            @click=${() => props.onRunGlobalInspection?.()}
          >
            ${props.globalInspecting ? "正在启动全局巡检…" : "启动全局深度巡检"}
          </button>
        </div>
      </div>

      ${props.dashboardError
        ? html`
            <div class="ops-panel" style="margin-bottom: 16px;">
              ${renderOpsError({ message: props.dashboardError })}
            </div>
          `
        : nothing}

      ${awaitingData
        ? html`
            <div class="ops-panel" style="margin-bottom: 24px;">${renderOpsSkeleton({ lines: 4 })}</div>
          `
        : nothing}

      ${!hasStats && !awaitingData && !showEmptyMetrics
        ? html`
            <div class="ops-panel ops-panel--empty">
              ${renderOpsEmpty({
                icon: "overviewGrid",
                title: "暂无汇总数据",
                description:
                  "运维大屏指标来自集群资产登记与监控查询，当前尚未接入或未返回数据。",
                hint: "请先在「服务与资产 → 集群资产管理」登记集群，并在环境变量中配置 VICTORIAMETRICS_URL。",
                actionLabel: "前往集群资产管理",
                onAction: props.onOpenAssets,
                spread: true,
              })}
            </div>
          `
        : showEmptyMetrics
          ? html`
              <div class="ops-panel ops-panel--empty">
                ${renderOpsEmpty({
                  icon: "server",
                  title: "尚无纳管集群",
                  description: "在集群资产管理中登记后，指标将自动汇总到此页。",
                  actionLabel: "前往集群资产管理",
                  onAction: props.onOpenAssets,
                  spread: true,
                })}
              </div>
            `
          : stats
            ? html`
                <div class="stats-grid">
                  <div class="stat-card">
                    <div class="stat-icon stat-icon--blue">${icons.server}</div>
                    <div class="stat-content">
                      <h3>纳管集群</h3>
                      <div class="stat-value">${stats.totalClusters}</div>
                    </div>
                  </div>
                  <div class="stat-card">
                    <div class="stat-icon stat-icon--ok">${icons.checkCircle}</div>
                    <div class="stat-content">
                      <h3>健康</h3>
                      <div class="stat-value">${stats.healthyClusters}</div>
                    </div>
                  </div>
                  <div class="stat-card">
                    <div class="stat-icon stat-icon--warn">${icons.alertTriangle}</div>
                    <div class="stat-content">
                      <h3>亚健康</h3>
                      <div class="stat-value">${stats.warningClusters}</div>
                    </div>
                  </div>
                  <div class="stat-card">
                    <div class="stat-icon stat-icon--danger">${icons.bell}</div>
                    <div class="stat-content">
                      <h3>待处理告警</h3>
                      <div class="stat-value">${stats.pendingAlerts}</div>
                    </div>
                  </div>
                </div>
              `
            : nothing}

      ${hasStats && !awaitingData
        ? html`
            <section class="ops-map-section">
              <div class="ops-map-card">
                <h2 class="section-title" style="margin-bottom: 16px;">
                  <span class="section-title__icon">${icons.network}</span>
                  业务地域拓扑与健康状态
                </h2>
                <ops-map 
                  .clusters=${props.clusters || []}
                  .snapshots=${props.snapshots || []}
                  .onNavigateDomain=${(domain: string) => {
                    const tabMap: Record<string, Tab> = {
                      hadoop: "hadoop",
                      fi: "fi",
                      gbase: "gbase",
                      governance: "governance",
                      dataapps: "dataapps",
                    };
                    const targetTab = tabMap[domain];
                    if (targetTab && props.onNavigateDomain) {
                      props.onNavigateDomain(targetTab);
                    }
                  }}
                ></ops-map>
              </div>
            </section>
          `
        : nothing}

      <div class="ops-dashboard-main ops-dashboard-main--full">
        <section class="domain-status-section">
          <div class="domain-status-section__head">
            <h2 class="section-title">
              <span class="section-title__icon">${icons.activity}</span>
              业务域健康度
            </h2>
            ${domains !== null && domains.length > 0
              ? html`<span class="domain-status-section__hint">
                  ${partitioned && partitioned.managed.length > 0
                    ? "默认展示需关注的业务域，运行平稳的域可展开查看"
                    : "以下业务域尚未纳管，请前往资产管理登记集群"}
                </span>`
              : domains === null
                ? html`<span class="domain-status-section__hint">完成资产登记与监控接入后，将在此展示各域健康分</span>`
                : nothing}
          </div>
          ${awaitingData
            ? html`
                <div class="domain-subsection">
                  <p class="domain-subsection__desc">正在加载业务域健康数据…</p>
                </div>
              `
            : domains === null
            ? html`
                <div class="domain-subsection domain-subsection--pending">
                  <p class="domain-subsection__desc">
                    在顶部导航 <strong>服务与资产</strong> 中登记集群，接入后即可在此查看健康分与告警概况。
                  </p>
                  ${renderUnmanagedInline(DOMAIN_PLACEHOLDERS, props)}
                </div>
              `
            : domains.length === 0
              ? html`
                  <div class="ops-panel">
                    ${renderOpsEmpty({
                      icon: "activity",
                      title: "暂无业务域健康分",
                      description: "完成 VictoriaMetrics 对接后，将按域聚合健康得分。",
                      compact: true,
                    })}
                  </div>
                `
              : html`
                  ${partitioned && partitioned.managed.length > 0
                    ? html`
                        <div class="domain-subsection">
                          ${partitioned.attention.length > 0
                            ? html`
                                <h3 class="domain-subsection__title">
                                  需关注（${partitioned.attention.length}）
                                </h3>
                                ${useDomainTable
                                  ? renderDomainTable(partitioned.attention, props, pendingByDomain)
                                  : html`
                                      <div class="domain-grid domain-grid--managed">
                                        ${partitioned.attention.map((d) => renderDomainCard(d, props))}
                                      </div>
                                    `}
                              `
                            : html`
                                <p class="domain-subsection__desc domain-subsection__desc--ok">
                                  各业务域运行平稳，暂无需要优先处理的事项。
                                </p>
                              `}
                          ${partitioned.stable.length > 0
                            ? html`
                                <details class="domain-collapse">
                                  <summary class="domain-collapse__summary">
                                    运行平稳（${partitioned.stable.length}）
                                  </summary>
                                  ${useDomainTable
                                    ? renderDomainTable(partitioned.stable, props, pendingByDomain)
                                    : html`
                                        <div class="domain-grid domain-grid--managed">
                                          ${partitioned.stable.map((d) => renderDomainCard(d, props))}
                                        </div>
                                      `}
                                </details>
                              `
                            : nothing}
                        </div>
                      `
                    : nothing}
                  ${partitioned && partitioned.unmanaged.length > 0
                    ? renderUnmanagedInline(partitioned.unmanaged, props)
                    : nothing}
                `}
        </section>
      </div>

      ${props.dashboardToast
        ? html`<div class="ops-dashboard-toast" role="status">${props.dashboardToast}</div>`
        : nothing}

      ${hasStats && !awaitingData
        ? html`
            <div class="ops-dashboard-bottom ops-dashboard-bottom--feed">
              ${renderRecentActivityFeed(props, stats)}
            </div>
          `
        : nothing}
    </div>
  `;
}
