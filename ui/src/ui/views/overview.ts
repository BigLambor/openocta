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
  healthyCount?: number;
  warningCount?: number;
  criticalCount?: number;
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
  onOpenDomainAssets?: (tab: Tab) => void;
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
    return "";
  }
  if (score >= 75) {
    return "warning";
  }
  return "critical";
}

function domainStatusLabel(d: OverviewDomainCard): string {
  if ((d.clusterCount ?? 0) === 0) {
    return "尚未纳管";
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

function partitionDomains(domains: OverviewDomainCard[]): {
  managed: OverviewDomainCard[];
  unmanaged: OverviewDomainCard[];
  attention: OverviewDomainCard[];
} {
  const managed = domains.filter((d) => (d.clusterCount ?? 0) > 0);
  const unmanaged = domains.filter((d) => (d.clusterCount ?? 0) === 0);
  managed.sort((a, b) => {
    const sa = a.score ?? 101;
    const sb = b.score ?? 101;
    if (sa !== sb) {
      return sa - sb;
    }
    return a.name.localeCompare(b.name, "zh-CN");
  });
  const attention = managed.filter(
    (d) =>
      (d.warningCount ?? 0) > 0 ||
      (d.criticalCount ?? 0) > 0 ||
      (d.score != null && d.score < 85),
  );
  return { managed, unmanaged, attention };
}

function renderDomainCard(
  d: OverviewDomainCard,
  props: OverviewProps,
  opts: { compact?: boolean; placeholder?: boolean } = {},
) {
  const score = d.score;
  const cls = scoreClass(score);
  const managed = (d.clusterCount ?? 0) > 0;
  const statusLabel = opts.placeholder ? "待接入监控指标" : domainStatusLabel(d);

  return html`
    <article
      class="domain-card ${!managed || score == null ? "domain-card--muted" : ""} ${opts.compact
        ? "domain-card--compact"
        : ""}"
      role="button"
      tabindex="0"
      @click=${(e: Event) => {
        const target = e.target as HTMLElement;
        if (target.closest("button, a")) {
          return;
        }
        props.onNavigateDomain?.(d.tab);
      }}
      @keydown=${(e: KeyboardEvent) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          props.onNavigateDomain?.(d.tab);
        }
      }}
    >
      <div class="domain-card-header">
        <div class="domain-name">
          <span class="domain-icon-wrapper">${iconForDomain(d.icon)}</span>
          <span class="domain-name__text">${d.name}</span>
        </div>
        <span class="domain-score ${cls}">${score != null ? `${score}分` : "—"}</span>
      </div>
      ${managed && score != null
        ? html`
            <div class="health-bar-container">
              <div class="health-bar ${cls}" style="width: ${Math.min(score, 100)}%;"></div>
            </div>
          `
        : nothing}
      <div class="domain-stats">
        <span>${d.clusterCount ?? 0} 个集群</span>
        <span class="domain-stats__status">${statusLabel}</span>
      </div>
      ${props.onNavigateDomain || props.onOpenDomainAssets
        ? html`
            <div class="domain-card-actions">
              ${props.onNavigateDomain
                ? html`
                    <button
                      type="button"
                      class="ops-btn ops-btn--primary domain-card-link"
                      @click=${() => props.onNavigateDomain!(d.tab)}
                    >
                      进入域详情
                    </button>
                  `
                : nothing}
              ${props.onOpenDomainAssets
                ? html`
                    <button
                      type="button"
                      class="ops-btn domain-card-link domain-card-link--secondary"
                      @click=${() => props.onOpenDomainAssets!(d.tab)}
                    >
                      资产
                    </button>
                  `
                : nothing}
            </div>
          `
        : nothing}
    </article>
  `;
}

function renderQuickLinks(props: OverviewProps) {
  return html`
    <nav class="ops-dashboard-quicklinks" aria-label="快捷操作">
      <button
        type="button"
        class="ops-dashboard-quicklink"
        ?disabled=${!props.onOpenPendingAlerts}
        title="跳转到业务域「告警降噪与影响评估」子页"
        @click=${() => props.onOpenPendingAlerts?.()}
      >
        <span class="ops-dashboard-quicklink__icon">${icons.bell}</span>
        <span class="ops-dashboard-quicklink__label">未处理告警</span>
      </button>
      <button
        type="button"
        class="ops-dashboard-quicklink"
        ?disabled=${!props.onOpenScheduledTasks}
        @click=${() => props.onOpenScheduledTasks?.()}
      >
        <span class="ops-dashboard-quicklink__icon">${icons.historyClock}</span>
        <span class="ops-dashboard-quicklink__label">定时任务</span>
      </button>
      <button
        type="button"
        class="ops-dashboard-quicklink"
        @click=${() => props.onOpenConfig?.()}
      >
        <span class="ops-dashboard-quicklink__icon">${icons.monitor}</span>
        <span class="ops-dashboard-quicklink__label">系统配置</span>
      </button>
      <button type="button" class="ops-dashboard-quicklink" @click=${() => props.onOpenAssets?.()}>
        <span class="ops-dashboard-quicklink__icon">${icons.server}</span>
        <span class="ops-dashboard-quicklink__label">集群资产</span>
      </button>
    </nav>
  `;
}

function renderAttentionPanel(
  attention: OverviewDomainCard[],
  stats: OverviewStats | null,
  props: OverviewProps,
) {
  const pending = stats?.pendingAlerts ?? 0;
  return html`
    <section class="ops-dashboard-panel">
      <h2 class="section-title">
        <span class="section-title__icon">${icons.alertTriangle}</span>
        需关注
      </h2>
      <div class="ops-panel ops-dashboard-panel__body">
        ${attention.length > 0
          ? html`
              <ul class="ops-attention-list">
                ${attention.map(
                  (d) => html`
                    <li class="ops-attention-item">
                      <button
                        type="button"
                        class="ops-attention-item__btn"
                        @click=${() => props.onNavigateDomain?.(d.tab)}
                      >
                        <span class="ops-attention-item__name">${d.name}</span>
                        <span class="ops-attention-item__meta">
                          ${d.score != null ? `${d.score}分` : "—"} · ${domainStatusLabel(d)}
                        </span>
                      </button>
                    </li>
                  `,
                )}
              </ul>
            `
          : html`
              <p class="ops-dashboard-panel__empty">
                ${pending > 0
                  ? `当前有 ${pending} 组待处理告警，建议优先查看。`
                  : "各业务域运行平稳，暂无需要优先处理的事项。"}
              </p>
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

function renderOpsSnapshot(summary: OpsDashboardSummary | null | undefined, stats: OverviewStats | null) {
  const vmConfigured = summary?.vmConfigured ?? false;
  const criticalClusters = summary?.criticalClusters ?? 0;
  return html`
    <section class="ops-dashboard-panel">
      <h2 class="section-title">
        <span class="section-title__icon">${icons.barChart}</span>
        运营概览
      </h2>
      <div class="ops-panel ops-dashboard-panel__body">
        <dl class="ops-snapshot-list">
          <div class="ops-snapshot-row">
            <dt>监控接入</dt>
            <dd class=${vmConfigured ? "ops-snapshot-row--ok" : "ops-snapshot-row--warn"}>
              ${vmConfigured ? "VictoriaMetrics 已配置" : "未配置 VictoriaMetrics"}
            </dd>
          </div>
          <div class="ops-snapshot-row">
            <dt>集群健康分布</dt>
            <dd>
              ${stats
                ? `${stats.healthyClusters} 健康 / ${stats.warningClusters} 亚健康${criticalClusters > 0 ? ` / ${criticalClusters} 严重` : ""}`
                : "—"}
            </dd>
          </div>
          <div class="ops-snapshot-row">
            <dt>业务域覆盖</dt>
            <dd>
              ${summary?.domains.filter((d) => d.clusterCount > 0).length ?? 0} /
              ${summary?.domains.length ?? DOMAIN_PLACEHOLDERS.length} 已纳管
            </dd>
          </div>
        </dl>
      </div>
    </section>
  `;
}

export function renderOverview(props: OverviewProps) {
  const fromApi = props.dashboardSummary ? summaryToCards(props.dashboardSummary) : null;
  const stats = fromApi?.stats ?? props.stats ?? null;
  const hasStats = stats != null && (stats.totalClusters > 0 || fromApi != null);
  const domains =
    fromApi?.domains ??
    (props.domains?.length ? props.domains : hasStats && stats && stats.totalClusters > 0 ? [] : null);
  const showEmptyMetrics = fromApi != null && stats != null && stats.totalClusters === 0;
  const partitioned = domains ? partitionDomains(domains) : null;

  return html`
    <div class="ops-page ops-dashboard">
      <div class="ops-page-header ops-dashboard-header ops-dashboard-header--split">
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
        <div class="ops-dashboard-header__actions">
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

      <div class="ops-dashboard-main">
        <section class="domain-status-section">
          <div class="domain-status-section__head">
            <h2 class="section-title">
              <span class="section-title__icon">${icons.activity}</span>
              业务域健康度
            </h2>
            ${partitioned && partitioned.managed.length > 0
              ? html`<span class="domain-status-section__hint">按健康分从低到高排序，点击查看详情</span>`
              : nothing}
          </div>
          ${domains === null
            ? html`
                <div class="domain-grid domain-grid--placeholders">
                  ${DOMAIN_PLACEHOLDERS.map((d) => renderDomainCard(d, props, { placeholder: true }))}
                </div>
                <p class="domain-unmanaged-hint">以上业务域待接入监控与资产登记后可展示健康分。</p>
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
                        <div class="domain-grid domain-grid--managed">
                          ${partitioned.managed.map((d) => renderDomainCard(d, props))}
                        </div>
                      `
                    : nothing}
                  ${partitioned && partitioned.unmanaged.length > 0
                    ? html`
                        <details class="domain-unmanaged" ?open=${partitioned.managed.length === 0}>
                          <summary class="domain-unmanaged__summary">
                            待接入业务域（${partitioned.unmanaged.length}）
                          </summary>
                          <div class="domain-chips">
                            ${partitioned.unmanaged.map(
                              (d) => html`
                                <button
                                  type="button"
                                  class="domain-chip"
                                  @click=${() => props.onOpenDomainAssets?.(d.tab)}
                                >
                                  <span class="domain-chip__icon">${iconForDomain(d.icon)}</span>
                                  ${d.name}
                                </button>
                              `,
                            )}
                          </div>
                        </details>
                      `
                    : nothing}
                `}
        </section>

        <aside class="ops-dashboard-aside">
          <h2 class="section-title">
            <span class="section-title__icon">${icons.zap}</span>
            快捷操作
          </h2>
          ${renderQuickLinks(props)}
        </aside>
      </div>

      ${props.dashboardToast
        ? html`<div class="ops-dashboard-toast" role="status">${props.dashboardToast}</div>`
        : nothing}

      ${hasStats && !props.loading
        ? html`
            <div class="ops-dashboard-bottom">
              ${renderAttentionPanel(partitioned?.attention ?? [], stats, props)}
              ${renderOpsSnapshot(props.dashboardSummary, stats)}
            </div>
          `
        : nothing}
    </div>
  `;
}
