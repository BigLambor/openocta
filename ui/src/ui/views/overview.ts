import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { renderOpsEmpty, renderOpsError, renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsDashboardSummary } from "../controllers/ops-clusters.ts";
import type { Tab } from "../navigation.ts";

export type OverviewDomainCard = {
  tab: Tab;
  name: string;
  icon: "network" | "building" | "database" | "layout" | "activity";
  score?: number;
  clusterCount?: number;
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
  dashboardSummary?: OpsDashboardSummary | null;
  dashboardError?: string | null;
  stats?: OverviewStats | null;
  domains?: OverviewDomainCard[] | null;
  globalInspecting?: boolean;
  dashboardToast?: string | null;
  onOpenAssets?: () => void;
  onOpenConfig?: () => void;
  onNavigateDomain?: (tab: Tab) => void;
  onOpenScheduledTasks?: () => void;
  onRunGlobalInspection?: () => void;
  onOpenPendingAlerts?: () => void;
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
        clusterCount: d.clusterCount,
        note,
      };
    }),
  };
}

function iconForDomain(icon: OverviewDomainCard["icon"]) {
  return icons[icon];
}

export function renderOverview(props: OverviewProps) {
  const fromApi = props.dashboardSummary ? summaryToCards(props.dashboardSummary) : null;
  const stats = fromApi?.stats ?? props.stats ?? null;
  const hasStats = stats != null && (stats.totalClusters > 0 || fromApi != null);
  const domains =
    fromApi?.domains ??
    (props.domains?.length ? props.domains : hasStats && stats && stats.totalClusters > 0 ? [] : null);
  const showEmptyMetrics = fromApi != null && stats != null && stats.totalClusters === 0;

  return html`
    <div class="ops-page ops-dashboard">
      <div class="ops-page-header ops-dashboard-header">
        <div>
          <h1>运维全局视图</h1>
          <p>
            ${stats && stats.totalClusters > 0
              ? `已纳管 ${stats.totalClusters} 个集群，覆盖 ${domains?.filter((d) => (d.clusterCount ?? 0) > 0).length ?? 0} 个有资产业务域。`
              : fromApi != null
                ? "尚未登记集群资产，请先在「集群资产管理」中添加。"
                : "接入集群资产与 VictoriaMetrics 后，将在此展示纳管规模、健康度与告警概况。"}
          </p>
        </div>
      </div>

      ${props.dashboardError
        ? html`
            <div class="ops-panel" style="margin-bottom: 16px;">
              ${renderOpsError({ message: props.dashboardError })}
            </div>
          `
        : nothing}

      ${props.loading
        ? html`
            <div class="ops-panel" style="margin-bottom: 24px;">${renderOpsSkeleton({ lines: 4 })}</div>
          `
        : nothing}

      ${!hasStats && !props.loading && !showEmptyMetrics
        ? html`
            <div class="ops-panel ops-panel--empty">
              ${renderOpsEmpty({
                icon: "overviewGrid",
                title: "暂无汇总数据",
                description:
                  "运维大屏指标来自集群资产登记与监控查询，当前尚未接入或未返回数据。",
                hint: "请先在「集群资产管理」登记集群，并在环境变量中配置 VICTORIAMETRICS_URL。",
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

      <section class="domain-status-section">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.activity}</span>
          业务域健康度
        </h2>
        ${domains === null
          ? html`
              <div class="domain-grid domain-grid--placeholders">
                ${DOMAIN_PLACEHOLDERS.map(
                  (d) => html`
                    <div class="domain-card domain-card--muted">
                      <div class="domain-card-header">
                        <div class="domain-name">
                          <span class="domain-icon-wrapper">${iconForDomain(d.icon)}</span>
                          <span>${d.name}</span>
                        </div>
                        <span class="domain-score domain-score--muted">—</span>
                      </div>
                      <p class="domain-card-hint">待接入监控指标</p>
                      ${props.onNavigateDomain
                        ? html`
                            <button
                              type="button"
                              class="ops-btn domain-card-link"
                              @click=${() => props.onNavigateDomain!(d.tab)}
                            >
                              进入 ${d.name}
                            </button>
                          `
                        : nothing}
                    </div>
                  `,
                )}
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
                <div class="domain-grid">
                  ${domains.map((d) => {
                    const score = d.score;
                    const scoreClass =
                      score == null
                        ? "muted"
                        : score >= 90
                          ? ""
                          : score >= 75
                            ? "warning"
                            : "critical";
                    return html`
                      <div class="domain-card ${score == null ? "domain-card--muted" : ""}">
                        <div class="domain-card-header">
                          <div class="domain-name">
                            <span class="domain-icon-wrapper">${iconForDomain(d.icon)}</span>
                            <span>${d.name}</span>
                          </div>
                          <span class="domain-score ${scoreClass}">
                            ${score != null ? `${score}分` : "—"}
                          </span>
                        </div>
                        <div class="health-bar-container">
                          <div
                            class="health-bar ${scoreClass}"
                            style="width: ${score != null ? Math.min(score, 100) : 0}%;"
                          ></div>
                        </div>
                        <div class="domain-stats">
                          <span>${d.clusterCount ?? 0} 个集群</span>
                          <span>${d.note ?? "—"}</span>
                        </div>
                        ${props.onNavigateDomain
                          ? html`
                              <button
                                type="button"
                                class="ops-btn domain-card-link"
                                @click=${() => props.onNavigateDomain!(d.tab)}
                              >
                                打开运维域
                              </button>
                            `
                          : nothing}
                      </div>
                    `;
                  })}
                </div>
              `}
      </section>

      ${props.dashboardToast
        ? html`<div class="ops-dashboard-toast" role="status">${props.dashboardToast}</div>`
        : nothing}

      <section class="ops-dashboard-actions">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          快捷操作
        </h2>
        <div class="ops-panel ops-dashboard-actions__inner">
          <button
            type="button"
            class="ops-btn ops-btn--primary ops-dashboard-actions__btn"
            ?disabled=${!props.onRunGlobalInspection || !props.connected || props.globalInspecting || props.canInspect === false}
            title=${props.canInspect === false
              ? "当前账号无 ops:inspect 权限"
              : !props.connected
                ? "请先连接网关"
                : "并行触发五个业务域的深度健康巡检任务"}
            @click=${() => props.onRunGlobalInspection?.()}
          >
            ${props.globalInspecting ? "正在启动全局巡检…" : "启动全局深度巡检"}
          </button>
          <button
            type="button"
            class="ops-btn ops-dashboard-actions__btn"
            ?disabled=${!props.onOpenPendingAlerts}
            title="跳转到业务域「告警降噪与影响评估」子页"
            @click=${() => props.onOpenPendingAlerts?.()}
          >
            查看未处理告警
          </button>
          <button
            type="button"
            class="ops-btn ops-dashboard-actions__btn"
            ?disabled=${!props.onOpenScheduledTasks}
            @click=${() => props.onOpenScheduledTasks?.()}
          >
            定时任务与运行历史
          </button>
          <button type="button" class="ops-btn ops-dashboard-actions__btn" @click=${() => props.onOpenConfig?.()}>
            系统与环境配置
          </button>
        </div>
      </section>
    </div>
  `;
}
