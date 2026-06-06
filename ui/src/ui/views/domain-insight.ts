import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { opsDomainLabel } from "../components/domain-filter.ts";
import {
  clusterStatusLabel,
  clusterStatusTone,
  formatClusterCount,
  pickTopRiskClusters,
} from "../components/ops-health-distribution.ts";
import { renderOpsSkeleton } from "../components/ops-status.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import type { BchDomainScenarioSummary, BchScenarioCardSummary } from "../controllers/bch-scenario-summary.ts";

const FALLBACK_SCENARIOS: BchScenarioCardSummary[] = [
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
    initialQuestion: "分析当前 Flink 集群中存在背压的作业情况",
    summary:
      "当前 Flink 实时计算域整体得分为 82 分。存在 5 个高风险作业，其中 3 个作业发生严重背压，2 个作业 Checkpoint 连续失败。",
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
    summary:
      "Spark 作业调优专项评估得分 65 分，低于健康水位。识别出 12 个作业存在严重的资源浪费与数据倾斜问题。",
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
    summary: "HDFS 存储域状态非常健康（95分）。容量使用率为 68%，无坏块产生。",
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
    status?: string;
  }>;
  scenarioSummary?: BchDomainScenarioSummary | null;
  scenarioSummaryLoading?: boolean;
  scenarioSummaryError?: string | null;
  onNavigateTab?: (tab: string, domain?: string, view?: string, scenarioId?: string) => void;
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

function scenarioNeedsAttention(scenario: BchScenarioCardSummary): boolean {
  if (scenario.status === "warning" || scenario.status === "critical") {
    return true;
  }
  return scenario.score != null && scenario.score < 90;
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

function resolveScenarios(props: DomainInsightProps, isBch: boolean): BchScenarioCardSummary[] {
  if (props.scenarioSummary?.scenarios?.length) {
    return props.scenarioSummary.scenarios;
  }
  return isBch ? FALLBACK_SCENARIOS : [];
}

function renderStatusStrip(props: DomainInsightProps, domain: string, name: string) {
  const score = props.score;
  const latestInspection = props.inspections?.[0] ?? null;
  const critical = props.criticalCount ?? 0;
  const warning = props.warningCount ?? 0;

  return html`
    <section class="domain-insight-status-strip" aria-label="${name} 域状态">
      <div class="domain-insight-status-strip__item">
        <span class="domain-insight-status-strip__label">综合健康</span>
        <span class="domain-insight-status-strip__value stat-value--${scoreClass(score)}">
          ${score != null ? `${score}分` : "—"}
        </span>
      </div>
      <div class="domain-insight-status-strip__item">
        <span class="domain-insight-status-strip__label">集群</span>
        <span class="domain-insight-status-strip__value">
          ${formatClusterCount(props.clusterCount ?? 0)}
          ${critical > 0
            ? html`<span class="domain-insight-status-strip__meta domain-insight-status-strip__meta--danger">${critical} 异常</span>`
            : warning > 0
              ? html`<span class="domain-insight-status-strip__meta domain-insight-status-strip__meta--warn">${warning} 亚健康</span>`
              : nothing}
        </span>
      </div>
      <div class="domain-insight-status-strip__item">
        <span class="domain-insight-status-strip__label">待处理告警</span>
        <span class="domain-insight-status-strip__value">${props.alertCount ?? 0} 组</span>
      </div>
      <div class="domain-insight-status-strip__item">
        <span class="domain-insight-status-strip__label">最近巡检</span>
        <span class="domain-insight-status-strip__value stat-value--${latestInspection ? scoreClass(latestInspection.score) : "unknown"}">
          ${latestInspection?.score != null ? `${latestInspection.score}分` : "—"}
          ${latestInspection?.time
            ? html`<span class="domain-insight-status-strip__meta">${latestInspection.time}</span>`
            : nothing}
        </span>
      </div>
    </section>
  `;
}

function renderImmediateActions(props: DomainInsightProps, domain: string) {
  const alerts = (props.alertGroups ?? []).filter((g) => g.status !== "resolved").slice(0, 5);
  const risky = pickTopRiskClusters(props.clusters ?? [], 5);
  const loading = props.clustersLoading && risky.length === 0 && alerts.length === 0;
  const hasItems = alerts.length > 0 || risky.length > 0;

  return html`
    <section class="ops-dashboard-panel domain-insight-section">
      <div class="domain-insight-section__head">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.alertTriangle}</span>
          需立即处理
        </h2>
        ${hasItems
          ? html`
              <button
                type="button"
                class="ops-btn ops-btn--ghost domain-insight-section__link"
                @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
              >
                进入工作台 ${icons.chevronRight}
              </button>
            `
          : nothing}
      </div>
      <div class="ops-panel ops-dashboard-panel__body">
        ${loading
          ? renderOpsSkeleton({ lines: 4 })
          : !hasItems
            ? html`
                <p class="ops-dashboard-panel__empty">
                  ${(props.clusterCount ?? 0) === 0
                    ? "该域尚未登记集群，请前往服务与资产纳管。"
                    : "当前域内暂无待处理告警或风险集群。"}
                </p>
              `
            : html`
                <ul class="ops-immediate-list">
                  ${alerts.map(
                    (g) => html`
                      <li class="ops-immediate-item">
                        <button
                          type="button"
                          class="ops-immediate-item__btn"
                          @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
                        >
                          <span class="ops-immediate-item__type">告警</span>
                          <span class="ops-feed-severity ops-feed-severity--${g.severity ?? "info"}">
                            ${g.severity === "critical" ? "严重" : g.severity === "warning" ? "警告" : "信息"}
                          </span>
                          <span class="ops-immediate-item__title">${g.title ?? "未命名告警"}</span>
                          <span class="ops-immediate-item__meta">${g.timestamp ?? "—"}</span>
                          ${icons.chevronRight}
                        </button>
                      </li>
                    `,
                  )}
                  ${risky.map(
                    (cluster) => html`
                      <li class="ops-immediate-item">
                        <button
                          type="button"
                          class="ops-immediate-item__btn"
                          @click=${() => props.onNavigateTab?.("assets", domain)}
                        >
                          <span class="ops-immediate-item__type">集群</span>
                          <span class="ops-feed-severity ops-feed-severity--${clusterStatusTone(cluster.status)}">
                            ${clusterStatusLabel(cluster.status)}
                          </span>
                          <span class="ops-immediate-item__title">${cluster.name}</span>
                          <span class="ops-immediate-item__meta">
                            ${formatClusterCount(cluster.nodeCount)} 节点
                          </span>
                          ${icons.chevronRight}
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

function renderScenarioRow(
  props: DomainInsightProps,
  domain: string,
  scenario: BchScenarioCardSummary,
) {
  const tone = scenario.score != null ? scoreClass(scenario.score) : statusClass(scenario.status);
  return html`
    <li class="domain-scenario-row domain-scenario-row--${tone}">
      <div class="domain-scenario-row__main">
        <span class="domain-scenario-row__title">${scenario.title}</span>
        <span class="domain-scenario-row__metric">${scenario.primaryMetric}</span>
      </div>
      <span class="domain-scenario-row__score ${tone}">${scenario.score != null ? `${scenario.score}分` : "—"}</span>
      <button
        type="button"
        class="ops-btn ops-btn--ghost domain-scenario-row__action"
        @click=${() =>
          props.onNavigateTab?.(
            "workbench",
            domain,
            scenario.primaryView,
            workbenchScenarioId(scenario, scenario.primaryView) ?? undefined,
          )}
      >
        ${scenario.primaryActionLabel} ${icons.chevronRight}
      </button>
    </li>
  `;
}

function renderScenarioSection(props: DomainInsightProps, domain: string, isBch: boolean) {
  const scenarios = resolveScenarios(props, isBch);
  const abnormal = scenarios.filter(scenarioNeedsAttention);
  const healthy = scenarios.filter((s) => !scenarioNeedsAttention(s));

  if (props.scenarioSummaryLoading && scenarios.length === 0) {
    return html`
      <section class="domain-insight-section">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          场景异常
        </h2>
        <div class="ops-panel">${renderOpsSkeleton({ lines: 3 })}</div>
      </section>
    `;
  }

  if (!isBch && scenarios.length === 0) {
    return html`
      <section class="domain-insight-section">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          场景异常
        </h2>
        <div class="ops-panel ops-dashboard-panel__body">
          <p class="ops-dashboard-panel__empty">
            该域业务场景接入中。可前往运维工作台查看事件与巡检。
          </p>
          <button
            type="button"
            class="ops-btn ops-dashboard-panel__action"
            @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
          >
            进入运维工作台
          </button>
        </div>
      </section>
    `;
  }

  return html`
    <section class="domain-insight-section">
      <h2 class="section-title">
        <span class="section-title__icon">${icons.zap}</span>
        场景异常
      </h2>
      ${abnormal.length > 0
        ? html`
            <ul class="domain-scenario-list">
              ${abnormal.map((scenario) => renderScenarioRow(props, domain, scenario))}
            </ul>
          `
        : html`
            <div class="ops-panel ops-dashboard-panel__body">
              <p class="ops-dashboard-panel__empty">各业务场景运行平稳，暂无异常项。</p>
            </div>
          `}
      ${healthy.length > 0
        ? html`
            <details class="domain-collapse domain-collapse--scenarios">
              <summary class="domain-collapse__summary">${healthy.length} 个场景运行正常</summary>
              <ul class="domain-scenario-list domain-scenario-list--healthy">
                ${healthy.map(
                  (scenario) => html`
                    <li class="domain-scenario-row domain-scenario-row--ok domain-scenario-row--compact">
                      <span class="domain-scenario-row__title">${scenario.title}</span>
                      <span class="domain-scenario-row__score ok">${scenario.score != null ? `${scenario.score}分` : "正常"}</span>
                    </li>
                  `,
                )}
              </ul>
            </details>
          `
        : nothing}
    </section>
  `;
}

function renderFooterLinks(
  props: DomainInsightProps,
  domain: string,
  latestInspection: NonNullable<DomainInsightProps["inspections"]>[number] | undefined,
) {
  return html`
    <nav class="domain-insight-footer-links" aria-label="更多操作">
      <button type="button" class="domain-insight-footer-links__link" @click=${() => props.onNavigateTab?.("assets", domain)}>
        查看全部集群
      </button>
      <button
        type="button"
        class="domain-insight-footer-links__link"
        @click=${() => props.onNavigateTab?.("workbench", domain, "inspection")}
      >
        ${latestInspection ? "查看巡检报告" : "进入巡检中心"}
      </button>
      <button
        type="button"
        class="domain-insight-footer-links__link"
        @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
      >
        进入事件中心
      </button>
    </nav>
  `;
}

export function renderDomainInsight(props: DomainInsightProps) {
  const domain = props.domain || "hadoop";
  const name = opsDomainLabel(domain);
  const isBch = domain === "hadoop";
  const latestInspection = props.inspections?.[0] ?? null;

  return html`
    <div class="ops-page ops-dashboard domain-insight-page">
      <div class="ops-page-header ops-dashboard-header ops-dashboard-header--split">
        <div class="domain-insight-header__left">
          <nav class="domain-insight-breadcrumb" aria-label="面包屑">
            <button
              type="button"
              class="domain-insight-breadcrumb__link"
              @click=${() => props.onNavigateTab?.("overview")}
            >
              运维驾驶舱
            </button>
            <span class="domain-insight-breadcrumb__sep">${icons.chevronRight}</span>
            <span class="domain-insight-breadcrumb__current">${name}</span>
          </nav>
          <div class="domain-insight-header__title-wrapper">
            <h1>${name}</h1>
          </div>
          <p class="domain-insight-header__desc">
            域级异常清单：聚焦待处理事项与场景风险，全量集群与操作请前往工作台或服务与资产。
          </p>
        </div>
        <div class="ops-dashboard-header__actions">
          <button type="button" class="ops-btn" @click=${() => props.onNavigateTab?.("assets", domain)}>
            查看全部集群
          </button>
          <button
            type="button"
            class="ops-btn ops-btn--primary"
            @click=${() => props.onNavigateTab?.("workbench", domain, "events")}
          >
            进入工作台
          </button>
        </div>
      </div>

      ${renderStatusStrip(props, domain, name)}
      ${renderImmediateActions(props, domain)}
      ${renderScenarioSection(props, domain, isBch)}
      ${renderFooterLinks(props, domain, latestInspection)}
    </div>
  `;
}
