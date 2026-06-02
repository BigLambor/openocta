import { html } from "lit";
import { icons } from "../icons.ts";
import type { Tab } from "../navigation.ts";
import type { OpsDashboardSummary } from "../controllers/ops-clusters.ts";
import { renderOpsEmpty, renderOpsError, renderOpsSkeleton } from "../components/ops-status.ts";

type TechOpsHubProps = {
  onOpenDomain: (tab: Extract<Tab, "hadoop" | "fi" | "gbase" | "governance" | "dataapps">) => void;
  loading?: boolean;
  dashboardSummary?: OpsDashboardSummary | null;
  dashboardError?: string | null;
  globalInspecting?: boolean;
  dashboardToast?: string | null;
  onOpenAssets?: () => void;
  onRunGlobalInspection?: () => void;
  onOpenPendingAlerts?: () => void;
  canInspect?: boolean;
};

const DOMAINS: Array<{
  tab: Extract<Tab, "hadoop" | "fi" | "gbase" | "governance" | "dataapps">;
  name: string;
  summary: string;
  icon: keyof typeof icons;
}> = [
  {
    tab: "hadoop",
    name: "BCH 生态",
    summary: "面向 BCH / Hadoop 生态的集群、组件、作业治理与容量优化。",
    icon: "network",
  },
  {
    tab: "fi",
    name: "FI 商业生态",
    summary: "面向 FusionInsight 商业闭源生态的可观测、巡检、诊断与治理。",
    icon: "building",
  },
  {
    tab: "gbase",
    name: "GBase 数据库",
    summary: "面向数据库实例、性能瓶颈、告警关联与故障诊断的技术域运维。",
    icon: "database",
  },
  {
    tab: "governance",
    name: "开发治理平台",
    summary: "覆盖研发流水线、元数据、配置与合规风险的治理场景。",
    icon: "layout",
  },
  {
    tab: "dataapps",
    name: "数据 App 运维",
    summary: "覆盖数据应用链路、调度、服务质量与稳定性保障场景。",
    icon: "activity",
  },
];

export function renderTechOpsHub(props: TechOpsHubProps) {
  const summary = props.dashboardSummary;
  const stats = summary
    ? {
        totalClusters: summary.totalClusters,
        healthyClusters: summary.healthyClusters,
        warningClusters: summary.warningClusters,
        criticalClusters: summary.criticalClusters,
        pendingAlerts: summary.pendingAlerts,
      }
    : null;

  return html`
    <main class="ops-dashboard">
      <div class="ops-dashboard-header">
        <div>
          <h1>技术域运维</h1>
          <p class="muted">
            从运维对象域进入平台能力，统一承载 BCH、FI、GBase、开发治理与数据 App 场景。
          </p>
        </div>
      </div>

      ${props.dashboardError
        ? html`<div class="ops-panel" style="margin-bottom: 16px;">${renderOpsError({ message: props.dashboardError })}</div>`
        : null}

      ${props.loading
        ? html`<div class="ops-panel" style="margin-bottom: 16px;">${renderOpsSkeleton({ lines: 4 })}</div>`
        : null}

      ${!props.loading && stats && stats.totalClusters > 0
        ? html`
            <section class="stats-grid">
              <article class="stat-card">
                <div class="stat-icon stat-icon--blue">${icons.server}</div>
                <div class="stat-content">
                  <h3>纳管集群</h3>
                  <div class="stat-value">${stats.totalClusters}</div>
                </div>
              </article>
              <article class="stat-card">
                <div class="stat-icon stat-icon--ok">${icons.checkCircle}</div>
                <div class="stat-content">
                  <h3>健康</h3>
                  <div class="stat-value">${stats.healthyClusters}</div>
                </div>
              </article>
              <article class="stat-card">
                <div class="stat-icon stat-icon--warn">${icons.alertTriangle}</div>
                <div class="stat-content">
                  <h3>亚健康</h3>
                  <div class="stat-value">${stats.warningClusters}</div>
                </div>
              </article>
              <article class="stat-card">
                <div class="stat-icon stat-icon--danger">${icons.bell}</div>
                <div class="stat-content">
                  <h3>待处理告警</h3>
                  <div class="stat-value">${stats.pendingAlerts}</div>
                </div>
              </article>
            </section>
          `
        : !props.loading && summary
          ? html`
              <section class="ops-panel ops-panel--empty" style="margin-bottom: 16px;">
                ${renderOpsEmpty({
                  icon: "server",
                  title: "尚无纳管集群",
                  description: "请先在集群资产管理中登记集群，技术域健康矩阵会自动汇总展示。",
                  actionLabel: "前往集群资产管理",
                  onAction: props.onOpenAssets,
                  spread: true,
                })}
              </section>
            `
          : null}

      <section class="domain-status-section">
        <div class="section-title">
          <span class="section-title__icon">${icons.network}</span>
          <span>技术域健康矩阵</span>
        </div>
        <div class="domain-grid">
          ${DOMAINS.map((domain) => {
            const icon = icons[domain.icon] ?? icons.folder;
            const domainSummary = summary?.domains.find((d) => d.domain === domain.tab);
            const score =
              domainSummary?.healthScore != null && Number.isFinite(domainSummary.healthScore)
                ? Math.round(domainSummary.healthScore)
                : null;
            const scoreClass = score == null ? "muted" : score >= 90 ? "" : score >= 75 ? "warning" : "critical";
            return html`
              <button class="domain-card domain-card-link" type="button" @click=${() => props.onOpenDomain(domain.tab)}>
                <div class="domain-card-header">
                  <div class="domain-name">
                    <span class="domain-icon-wrapper">${icon}</span>
                    <div class="domain-name">${domain.name}</div>
                  </div>
                  <span class="domain-score ${scoreClass}">${score != null ? `${score}分` : "—"}</span>
                </div>
                <div class="domain-card-hint">${domain.summary}</div>
                <div class="domain-stats" style="margin-top: 8px;">
                  <span>${domainSummary?.clusterCount ?? 0} 个集群</span>
                  <span>${domainSummary?.note ?? domainSummary?.healthScoreNote ?? "待补充域级洞察"}</span>
                </div>
              </button>
            `;
          })}
        </div>
      </section>

      <section class="ops-dashboard-actions">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          快捷操作
        </h2>
        <div class="ops-dashboard-actions__inner">
          <button
            class="ops-dashboard-actions__btn"
            type="button"
            ?disabled=${props.globalInspecting || props.canInspect === false}
            @click=${() => props.onRunGlobalInspection?.()}
            title=${props.canInspect === false ? "当前账号无 ops:inspect 权限" : ""}
          >
            ${props.globalInspecting ? icons.loader : icons.historyClock}
            ${props.globalInspecting ? "全域巡检中..." : "一键全域巡检"}
          </button>
          <button class="ops-dashboard-actions__btn" type="button" @click=${() => props.onOpenPendingAlerts?.()}>
            ${icons.bell} 查看待处理告警
          </button>
          <button class="ops-dashboard-actions__btn" type="button" @click=${() => props.onOpenAssets?.()}>
            ${icons.server} 管理集群资产
          </button>
        </div>
      </section>

      ${props.dashboardToast ? html`<div class="ops-dashboard-toast" role="status">${props.dashboardToast}</div>` : null}
    </main>
  `;
}
