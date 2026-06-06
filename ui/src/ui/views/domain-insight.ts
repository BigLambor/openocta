import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { opsDomainLabel } from "../components/domain-filter.ts";
import {
  clusterStatusLabel,
  clusterStatusTone,
  distributionFromClusters,
  distributionFromCounts,
  formatClusterCount,
  pickTopRiskClusters,
  renderHealthDistributionBar,
  renderHealthDistributionLegend,
} from "../components/ops-health-distribution.ts";
import { renderOpsEmpty, renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import type { BchDomainScenarioSummary, BchScenarioCardSummary } from "../controllers/bch-scenario-summary.ts";

const MOCK_SCENARIOS: BchScenarioCardSummary[] = [
  {
    id: "flink-health",
    title: "Flink 作业健康度",
    workflowType: "flink",
    capability: "health",
    status: "warning",
    score: 82,
    primaryMetric: "5 个作业运行异常",
    secondaryMetric: "背压 / Checkpoint 失败 / OOM",
    description: "监控 Flink 实时作业运行状态，快速定位背压、倾斜与重启瓶颈。",
    primaryActionLabel: "查看异常作业",
    primaryView: "diagnosis",
    secondaryActionLabel: "整体监控",
    secondaryView: "governance",
    initialQuestion: "分析当前 Flink 集群中存在背压的作业情况",
    summary: "当前 Flink 实时计算域整体得分为 82 分。存在 5 个高风险作业，其中 3 个作业发生严重背压，2 个作业 Checkpoint 连续失败。建议立即查看异常作业详情以进行针对性调优。",
  },
  {
    id: "spark-tuning",
    title: "Spark 作业调优",
    workflowType: "spark",
    capability: "tuning",
    status: "critical",
    score: 65,
    primaryMetric: "12 个作业资源浪费",
    secondaryMetric: "数据倾斜 / 内存闲置 / 慢节点",
    description: "洞察 Spark 离线与交互式分析作业，提供智能参数推荐与诊断。",
    primaryActionLabel: "进入调优中心",
    primaryView: "governance",
    initialQuestion: "帮我找出昨天执行最慢的 5 个 Spark 作业并提供调优建议",
    summary: "Spark 作业调优专项评估得分 65 分，低于健康水位。识别出 12 个作业存在严重的资源浪费与数据倾斜问题，累计消耗了 30% 的非必要计算资源。亟需进行参数优化。",
  },
  {
    id: "hdfs-storage",
    title: "HDFS 健康度",
    workflowType: "hdfs",
    capability: "storage",
    status: "healthy",
    score: 95,
    primaryMetric: "容量使用率 68%",
    secondaryMetric: "小文件健康 / 坏块清零 / 节点均衡",
    description: "全方位评估 HDFS 存储容量、小文件分布与 DataNode 负载均衡度。",
    primaryActionLabel: "容量管理",
    primaryView: "capacity",
    initialQuestion: "检查 HDFS 存储的小文件分布情况，是否有需要合并的目录",
    summary: "HDFS 存储域状态非常健康（95分）。容量使用率为 68%，无坏块产生。小文件占比已通过自动治理控制在 5% 以内。节点间数据分布均衡，目前无需人工介入。",
  },
  {
    id: "alert-noise",
    title: "告警降噪与收敛",
    workflowType: "alert",
    capability: "noise_reduction",
    status: "warning",
    score: 75,
    primaryMetric: "当前收敛率 85%",
    secondaryMetric: "昨日降噪 2.4万 条告警",
    description: "基于 AI 的智能告警降噪，识别告警风暴，提取根因告警，减少运维干扰。",
    primaryActionLabel: "配置降噪规则",
    primaryView: "governance",
    secondaryActionLabel: "降噪效果评估",
    secondaryView: "events",
    initialQuestion: "分析过去一周触发次数最多的 Top 3 告警，并给出降噪建议",
    summary: "告警系统运行中，近 24 小时产生原始告警 28,000 条，经智能降噪收敛为 4,200 个告警组，收敛率达 85%。但某几个特定集群频繁报出 CPU 抖动告警，建议检查对应的降噪规则阈值。",
  },
];

export type DomainInsightProps = {
  domain: string;
  connected?: boolean;
  loading?: boolean;
  score?: number;
  clusterCount?: number;
  healthyCount?: number;
  warningCount?: number;
  criticalCount?: number;
  alertCount?: number;
  clusters?: OpsClusterRecord[];
  clustersLoading?: boolean;
  inspections?: Array<{
    score?: number | null;
    time?: string;
    reportSummary?: string;
  }>;
  alertGroups?: Array<{
    id?: string;
    title?: string;
    severity?: string;
    timestamp?: string;
  }>;
  scenarioSummary?: BchDomainScenarioSummary | null;
  scenarioSummaryLoading?: boolean;
  scenarioSummaryError?: string | null;
  onNavigateTab?: (tab: string, domain?: string, view?: string, scenarioId?: string) => void;
  onRunInspection?: () => void;
  isInspecting?: boolean;
  canInspect?: boolean;
  onAnalyzeScenario?: (params: {
    scenario: string;
    capability: string;
    initialQuestion: string;
    summary: string;
  }) => void;
};

function scoreClass(score: number | null | undefined): string {
  if (score == null || score < 0) {
    return "unknown";
  }
  if (score >= 90) {
    return "ok";
  }
  if (score >= 75) {
    return "warning";
  }
  return "danger";
}

function statusClass(status: BchScenarioCardSummary["status"]): string {
  switch (status) {
    case "healthy":
      return "ok";
    case "warning":
      return "warning";
    case "critical":
      return "danger";
    default:
      return "unknown";
  }
}

function workbenchScenarioId(scenario: BchScenarioCardSummary, view?: string): string | null {
  switch (scenario.id) {
    case "flink-health":
      return view === "diagnosis" ? "bch-flink-health" : null;
    case "spark-tuning":
      return view === "governance" ? "bch-spark-tuning" : null;
    case "hdfs-storage":
      return view === "capacity" ? "bch-hdfs-capacity" : null;
    default:
      return null;
  }
}

function renderBchScenarioCard(
  props: DomainInsightProps,
  domain: string,
  scenario: BchScenarioCardSummary,
) {
  const tone = scenario.score != null ? scoreClass(scenario.score) : statusClass(scenario.status);
  return html`
    <article class="domain-card">
      <div class="domain-card-header">
        <div class="domain-name">
          <span class="domain-icon-wrapper">
            ${scenario.id === "hdfs-storage"
              ? icons.server
              : scenario.id === "spark-tuning"
                ? icons.zap
                : scenario.id === "alert-noise"
                  ? icons.bell
                  : icons.activity}
          </span>
          <span class="domain-name__text">${scenario.title}</span>
        </div>
        <span class="domain-score ${tone}">${scenario.score != null ? `${scenario.score}分` : "—"}</span>
      </div>
      <div class="health-bar-container">
        <div
          class="health-bar ${tone}"
          style="width: ${scenario.score != null ? Math.max(0, Math.min(100, scenario.score)) : 0}%;"
        ></div>
      </div>
      <div class="domain-stats domain-stats--stacked">
        <div>${scenario.primaryMetric}</div>
        <div class="domain-stats__hint">${scenario.secondaryMetric}</div>
        <p class="domain-stats__hint">${scenario.description}</p>
      </div>
      <div class="domain-card-actions">
        <button
          type="button"
          class="ops-btn ops-btn--primary domain-card-link"
          @click=${() => props.onNavigateTab?.("workbench", domain, scenario.primaryView, workbenchScenarioId(scenario, scenario.primaryView) ?? undefined)}
        >
          ${scenario.primaryActionLabel}
        </button>
        ${scenario.secondaryView && scenario.secondaryActionLabel
          ? html`
              <button
                type="button"
                class="ops-btn domain-card-link domain-card-link--secondary"
                @click=${() => props.onNavigateTab?.("workbench", domain, scenario.secondaryView, workbenchScenarioId(scenario, scenario.secondaryView) ?? undefined)}
              >
                ${scenario.secondaryActionLabel}
              </button>
            `
          : nothing}
        <button
          type="button"
          class="ops-btn domain-card-link domain-card-link--secondary"
          @click=${() =>
            props.onAnalyzeScenario?.({
              scenario: scenario.workflowType,
              capability: scenario.capability,
              initialQuestion: scenario.initialQuestion,
              summary: scenario.summary,
            })}
        >
          AI 分析
        </button>
      </div>
    </article>
  `;
}

function renderTopRiskClusters(props: DomainInsightProps, domain: string) {
  const clusters = props.clusters ?? [];
  const risky = pickTopRiskClusters(clusters, 5);
  const hasMany = clusters.length > 5;

  return html`
    <section class="ops-dashboard-panel domain-insight-section">
      <div class="domain-insight-section__head">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.alertTriangle}</span>
          风险集群 Top 5
        </h2>
        ${hasMany
          ? html`
              <button
                type="button"
                class="ops-btn ops-btn--ghost domain-insight-section__link"
                @click=${() => props.onNavigateTab?.("assets", domain)}
              >
                查看全部 ${formatClusterCount(clusters.length)} 个集群 ${icons.chevronRight}
              </button>
            `
          : nothing}
      </div>
      <div class="ops-panel ops-dashboard-panel__body">
        ${props.clustersLoading
          ? renderOpsSkeleton({ lines: 4 })
          : risky.length === 0
            ? html`
                <p class="ops-dashboard-panel__empty">
                  ${clusters.length === 0
                    ? "该域尚未登记集群，请前往服务与资产纳管。"
                    : "当前域内集群状态正常，暂无需要优先处理的风险集群。"}
                </p>
              `
            : html`
                <ul class="ops-risk-cluster-list">
                  ${risky.map(
                    (cluster) => html`
                      <li class="ops-risk-cluster-item">
                        <button
                          type="button"
                          class="ops-risk-cluster-item__btn"
                          @click=${() => props.onNavigateTab?.("assets", domain)}
                        >
                          <span class="ops-risk-cluster-item__head">
                            <span class="ops-feed-severity ops-feed-severity--${clusterStatusTone(cluster.status)}">
                              ${clusterStatusLabel(cluster.status)}
                            </span>
                            <span class="ops-risk-cluster-item__name">${cluster.name}</span>
                            ${cluster.region
                              ? html`<span class="ops-risk-cluster-item__meta">${cluster.region}</span>`
                              : nothing}
                          </span>
                          <span class="ops-risk-cluster-item__detail">
                            ${formatClusterCount(cluster.nodeCount)} 节点
                            ${cluster.owner ? ` · ${cluster.owner}` : nothing}
                            ${cluster.components?.length
                              ? ` · ${cluster.components.slice(0, 3).join(", ")}`
                              : nothing}
                          </span>
                        </button>
                      </li>
                    `,
                  )}
                </ul>
              `}
      </div>
    </section>
  `;
}

export function renderDomainInsight(props: DomainInsightProps) {
  const domain = props.domain || "hadoop";
  const name = opsDomainLabel(domain);
  const score = props.score;
  const isBch = domain === "hadoop";
  const latestInspection = props.inspections?.[0] ?? null;
  const dist =
    (props.clusters?.length ?? 0) > 0
      ? distributionFromClusters(props.clusters ?? [])
      : distributionFromCounts(
          props.healthyCount ?? 0,
          props.warningCount ?? 0,
          props.criticalCount ?? 0,
        );

  return html`
    <div class="ops-page ops-dashboard domain-insight-page">
      <div class="ops-page-header ops-dashboard-header ops-dashboard-header--split">
        <div class="domain-insight-header__left">
          <button
            type="button"
            class="ops-btn ops-btn--ghost domain-insight-header__back"
            @click=${() => props.onNavigateTab?.("overview")}
          >
            ${icons.arrowLeft} <span>返回驾驶舱</span>
          </button>
          <div class="domain-insight-header__title-wrapper">
            <h1>${name} <span class="domain-insight-header__subtitle">详情</span></h1>
          </div>
          <p class="domain-insight-header__desc">
            域级健康聚合视图：不展开全量集群，仅展示分布、风险 Top 5 与场景入口。全量清单请前往服务与资产。
          </p>
        </div>
        <div class="ops-dashboard-header__actions">
          <button
            type="button"
            class="ops-btn"
            @click=${() => props.onNavigateTab?.("assets", domain)}
          >
            ${icons.server} 查看全部集群
          </button>
          <button
            type="button"
            class="ops-btn ops-btn--primary"
            ?disabled=${props.isInspecting || props.canInspect === false}
            @click=${() => props.onRunInspection?.()}
          >
            ${props.isInspecting ? html`${icons.loader} 巡检中…` : html`${icons.zap} 触发域巡检`}
          </button>
        </div>
      </div>

      <div class="stats-grid">
        <div class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.activity}</div>
          <div class="stat-content">
            <h3>健康综合得分</h3>
            <div class="stat-value stat-value--${scoreClass(score)}">
              ${score != null ? `${score}分` : "—"}
            </div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.server}</div>
          <div class="stat-content">
            <h3>域内集群数</h3>
            <div class="stat-value">${formatClusterCount(props.clusterCount ?? 0)}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--danger">${icons.bell}</div>
          <div class="stat-content">
            <h3>待处理告警组</h3>
            <div class="stat-value">${props.alertCount ?? 0}</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--ok">${icons.checkCircle}</div>
          <div class="stat-content">
            <h3>最近巡检得分</h3>
            <div class="stat-value stat-value--${latestInspection ? scoreClass(latestInspection.score) : "unknown"}">
              ${latestInspection?.score != null ? `${latestInspection.score}分` : "—"}
            </div>
          </div>
        </div>
      </div>

      <section class="ops-dashboard-panel domain-insight-section">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.barChart}</span>
          集群健康分布
        </h2>
        <div class="ops-panel ops-dashboard-panel__body domain-insight-distribution">
          ${renderHealthDistributionBar(dist)}
          ${renderHealthDistributionLegend(dist)}
          <p class="domain-insight-distribution__hint">
            共 ${formatClusterCount(props.clusterCount ?? 0)} 个集群。
            ${(props.criticalCount ?? 0) > 0
              ? `${formatClusterCount(props.criticalCount ?? 0)} 个异常需优先处理。`
              : (props.warningCount ?? 0) > 0
                ? `${formatClusterCount(props.warningCount ?? 0)} 个亚健康建议关注。`
                : "当前域整体运行平稳。"}
          </p>
        </div>
      </section>

      ${renderTopRiskClusters(props, domain)}

      <section class="domain-insight-section">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          业务场景看板
        </h2>
        ${isBch
          ? html`
              <div class="domain-grid domain-grid--managed">
                ${MOCK_SCENARIOS.map((scenario) => renderBchScenarioCard(props, domain, scenario))}
              </div>
            `
          : html`
              <div class="domain-insight-section-empty">
                <div class="empty-icon">${icons.activity}</div>
                <div class="empty-title">暂无该域业务场景数据</div>
                <div class="empty-desc">您可以联系管理员接入专属的业务场景监控卡片。</div>
              </div>
            `}
      </section>

      <div class="ops-dashboard-bottom domain-insight-bottom">
        <section class="ops-dashboard-panel">
          <div class="domain-insight-section__head">
            <h2 class="section-title">
              <span class="section-title__icon">${icons.bell}</span>
              最新告警组
            </h2>
            <button
              type="button"
              class="domain-insight-section__action-link"
              @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
            >
              进入事件中心 ${icons.chevronRight}
            </button>
          </div>
          <div class="ops-panel ops-dashboard-panel__body">
            ${!props.alertGroups || props.alertGroups.length === 0
              ? html`<p class="ops-dashboard-panel__empty">暂无活动告警。</p>`
              : html`
                  <ul class="ops-feed-list">
                    ${props.alertGroups.slice(0, 5).map(
                      (g) => html`
                        <li class="ops-feed-item">
                          <button
                            type="button"
                            class="ops-feed-item__btn"
                            @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
                          >
                            <span class="ops-feed-item__head">
                              <span class="ops-feed-severity ops-feed-severity--${g.severity ?? "info"}">
                                ${g.severity === "critical" ? "严重" : g.severity === "warning" ? "警告" : "信息"}
                              </span>
                              <span class="ops-feed-item__time">${g.timestamp ?? "—"}</span>
                            </span>
                            <span class="ops-feed-item__title">${g.title}</span>
                          </button>
                        </li>
                      `,
                    )}
                  </ul>
                `}
          </div>
        </section>

        <section class="ops-dashboard-panel">
          <div class="domain-insight-section__head">
            <h2 class="section-title">
              <span class="section-title__icon">${icons.historyClock}</span>
              最近巡检摘要
            </h2>
            <button
              type="button"
              class="domain-insight-section__action-link"
              @click=${() => props.onNavigateTab?.("workbench", domain, "inspection")}
            >
              进入巡检中心 ${icons.chevronRight}
            </button>
          </div>
          <div class="ops-panel ops-dashboard-panel__body">
            ${!latestInspection
              ? html`
                  <p class="ops-dashboard-panel__empty">该域暂无巡检记录，可点击右上角「触发域巡检」。</p>
                `
              : html`
                  <div class="domain-insight-inspection">
                    <div class="domain-insight-inspection__head">
                      <span class="domain-insight-inspection__score stat-value--${scoreClass(latestInspection.score)}">
                        ${latestInspection.score != null ? `${latestInspection.score}分` : "—"}
                      </span>
                      <span class="ops-feed-item__score ops-feed-item__score--${scoreClass(latestInspection.score)}">
                        ${latestInspection.score != null && latestInspection.score >= 90
                          ? "状态良好"
                          : latestInspection.score != null && latestInspection.score >= 75
                            ? "存在风险"
                            : "需关注"}
                      </span>
                    </div>
                    <p class="domain-insight-inspection__time">${latestInspection.time ?? "—"}</p>
                    <p class="domain-insight-inspection__summary">${latestInspection.reportSummary ?? "—"}</p>
                  </div>
                `}
          </div>
        </section>
      </div>
    </div>
  `;
}
