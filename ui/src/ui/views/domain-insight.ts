import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { opsDomainLabel } from "../components/domain-filter.ts";
import { renderOpsEmpty } from "../components/ops-status.ts";
import type { BchDomainScenarioSummary, BchScenarioCardSummary } from "../controllers/bch-scenario-summary.ts";

export type DomainInsightProps = {
  domain: string;
  connected?: boolean;
  loading?: boolean;
  score?: number;
  clusterCount?: number;
  alertCount?: number;
  inspections?: any[];
  alertGroups?: any[];
  scenarioSummary?: BchDomainScenarioSummary | null;
  scenarioSummaryLoading?: boolean;
  scenarioSummaryError?: string | null;
  onNavigateTab?: (tab: string, domain?: string, view?: string) => void;
  onRunInspection?: () => void;
  isInspecting?: boolean;
  canInspect?: boolean;
  onAnalyzeScenario?: (params: { scenario: string; capability: string; initialQuestion: string; summary: string }) => void;
};

function scoreClass(score: number | null | undefined): string {
  if (score == null || score < 0) return "unknown";
  if (score >= 90) return "ok";
  if (score >= 75) return "warning";
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

function renderBchScenarioCard(
  props: DomainInsightProps,
  domain: string,
  scenario: BchScenarioCardSummary,
) {
  const tone = scenario.score != null ? scoreClass(scenario.score) : statusClass(scenario.status);
  return html`
    <div class="domain-card">
      <div class="domain-card-header">
        <div class="domain-name">
          <span class="domain-icon-wrapper">${scenario.id === "hdfs-storage" ? icons.server : scenario.id === "spark-tuning" ? icons.zap : icons.activity}</span>
          <span>${scenario.title}</span>
        </div>
        <span class="domain-score ${tone}">${scenario.score != null ? `${scenario.score}分` : "—"}</span>
      </div>
      <div class="health-bar-container">
        <div class="health-bar ${tone}" style="width: ${scenario.score != null ? Math.max(0, Math.min(100, scenario.score)) : 0}%;"></div>
      </div>
      <div class="domain-stats" style="flex-direction: column; align-items: flex-start; gap: 4px; margin-top: 8px;">
        <div>${scenario.primaryMetric}</div>
        <div class="muted" style="font-size: 12px;">${scenario.secondaryMetric}</div>
        <p class="muted" style="margin: 4px 0 0; font-size: 12px; line-height: 1.4;">${scenario.description}</p>
      </div>
      <div class="domain-card-actions" style="display: flex; gap: 8px; margin-top: 12px;">
        <button
          type="button"
          class="ops-btn ops-btn--primary"
          style="flex: 1; padding: 6px 12px; font-size: 13px;"
          @click=${() => props.onNavigateTab?.("workbench", domain, scenario.primaryView)}
        >
          ${scenario.primaryActionLabel}
        </button>
        ${scenario.secondaryView && scenario.secondaryActionLabel
          ? html`
              <button
                type="button"
                class="ops-btn"
                style="flex: 1; padding: 6px 12px; font-size: 13px;"
                @click=${() => props.onNavigateTab?.("workbench", domain, scenario.secondaryView)}
              >
                ${scenario.secondaryActionLabel}
              </button>
            `
          : nothing}
        <button
          type="button"
          class="ops-btn"
          style="flex: 1; padding: 6px 12px; font-size: 13px;"
          @click=${() => props.onAnalyzeScenario?.({
            scenario: scenario.workflowType,
            capability: scenario.capability,
            initialQuestion: scenario.initialQuestion,
            summary: scenario.summary,
          })}
        >
          AI 分析
        </button>
      </div>
    </div>
  `;
}

export function renderDomainInsight(props: DomainInsightProps) {
  const domain = props.domain || "hadoop";
  const name = opsDomainLabel(domain);
  const score = props.score ?? 92;
  const isBch = domain === "hadoop";

  // Last inspection run helper
  const latestInspection = props.inspections?.[0] || null;

  return html`
    <div class="ops-page ops-dashboard domain-insight-page">
      <div class="ops-page-header ops-dashboard-header" style="display: flex; justify-content: space-between; align-items: center; width: 100%;">
        <div>
          <div style="display: flex; align-items: center; gap: 8px;">
            <button
              class="ops-btn ops-btn--ghost"
              style="padding: 4px 8px; font-size: 13px;"
              @click=${() => props.onNavigateTab?.("overview")}
            >
              ${icons.arrowLeft} 返回驾驶舱
            </button>
            <h1 style="margin: 0;">${name} 详情</h1>
          </div>
          <p style="margin: 4px 0 0 42px;">单个技术域的健康度、核心指标、业务场景卡片以及快速操作入口。</p>
        </div>
        <div>
          <button
            class="ops-btn ops-btn--primary"
            ?disabled=${props.isInspecting || props.canInspect === false}
            @click=${() => props.onRunInspection?.()}
          >
            ${props.isInspecting ? html`${icons.loader} 巡检中...` : html`${icons.zap} 触发域巡检`}
          </button>
        </div>
      </div>

      <!-- Overview Stats Grid -->
      <div class="stats-grid" style="margin-top: 16px;">
        <div class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.activity}</div>
          <div class="stat-content">
            <h3>健康综合得分</h3>
            <div class="stat-value ${scoreClass(score)}">${score}分</div>
          </div>
        </div>
        <div class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.server}</div>
          <div class="stat-content">
            <h3>域内集群数</h3>
            <div class="stat-value">${props.clusterCount ?? 0}</div>
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
            <div class="stat-value ${latestInspection ? scoreClass(latestInspection.score) : "unknown"}">
              ${latestInspection?.score != null ? `${latestInspection.score}分` : "—"}
            </div>
          </div>
        </div>
      </div>

      <!-- Quick Context Links (看 -> 做 -> 管 -> 问) -->
      <section class="ops-panel" style="margin: 20px 0; padding: 16px;">
        <h3 style="margin: 0 0 12px; font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 6px;">
          ${icons.link} 快速入口
        </h3>
        <div style="display: flex; gap: 12px; flex-wrap: wrap;">
          <button class="ops-btn" @click=${() => props.onNavigateTab?.("workbench", domain, "events")}>
            ${icons.layout} 运维工作台 (处理事件/巡检)
          </button>
          <button class="ops-btn" @click=${() => props.onNavigateTab?.("assets", domain)}>
            ${icons.server} 服务与资产 (管理集群)
          </button>
          <button class="ops-btn" @click=${() => props.onNavigateTab?.("message", `agent:main:ops:${domain}`)}>
            ${icons.messageSquare} AI 运维助手 (对话分析)
          </button>
        </div>
      </section>

      <!-- Scenarios Cards Section -->
      <section style="margin: 24px 0;">
        <h2 class="section-title">
          <span class="section-title__icon">${icons.zap}</span>
          业务场景看板
        </h2>

        ${isBch
          ? html`
              ${props.scenarioSummaryLoading
                ? html`<div class="loading-placeholder">${icons.loader} 正在汇总 BCH 场景数据...</div>`
                : nothing}
              ${props.scenarioSummaryError
                ? html`<div class="ops-panel" style="margin-bottom: 12px; color: var(--danger, #d33);">${props.scenarioSummaryError}</div>`
                : nothing}
              ${props.scenarioSummary?.errors?.length
                ? html`
                    <div class="ops-banner" style="margin-bottom: 12px;">
                      <span class="ops-banner__icon">${icons.info}</span>
                      <span>部分 BCH 场景接口暂不可用：${props.scenarioSummary.errors.join("；")}</span>
                    </div>
                  `
                : nothing}
              ${props.scenarioSummary?.scenarios?.length
                ? html`
                    <div class="domain-grid">
                      ${props.scenarioSummary.scenarios.map((scenario) => renderBchScenarioCard(props, domain, scenario))}
                    </div>
                  `
                : props.scenarioSummaryLoading
                  ? nothing
                  : html`
                      <div class="ops-panel">
                        ${renderOpsEmpty({
                          icon: "activity",
                          title: "暂无 BCH 场景摘要",
                          description: "场景摘要来自 BCH Flink、Spark、HDFS 和集群健康接口。请确认网关接口已接入。",
                          compact: true,
                        })}
                      </div>
                    `}
            `
          : html`
              <div class="ops-panel">
                ${renderOpsEmpty({
                  icon: "activity",
                  title: `${name} 暂无特定场景卡片`,
                  description: "该域的特定分析场景（如数据库慢 SQL、开发元数据漂移等）正在集成中。",
                  hint: "您可以通过左侧快速入口或导航，在运维工作台直接对该域进行告警处置、一键健康巡检等操作。",
                  compact: true,
                })}
              </div>
            `}
      </section>

      <!-- Logs / Alerts and Inspections Summary Split -->
      <div class="ops-main-columns ops-shell-columns" style="margin-top: 24px;">
        <div class="ops-card list-column" style="min-height: 300px;">
          <div class="column-header" style="display: flex; justify-content: space-between; align-items: center;">
            <span>${icons.bell} 最新告警组</span>
            <button class="ops-btn ops-btn--ghost" style="padding: 2px 6px; font-size: 12px;" @click=${() => props.onNavigateTab?.("workbench", domain, "events")}>
              进入事件中心 ${icons.chevronRight}
            </button>
          </div>
          <div style="padding: 10px;">
            ${!props.alertGroups || props.alertGroups.length === 0
              ? html`<div class="empty-placeholder">暂无活动告警。</div>`
              : html`
                  <div class="alert-list">
                    ${props.alertGroups.slice(0, 5).map(
                      (g: any) => html`
                        <div class="alert-item alert-item--${g.severity}" style="cursor: pointer; padding: 10px; border-radius: 6px; margin-bottom: 8px;" @click=${() => props.onNavigateTab?.("workbench", domain, "events")}>
                          <div class="alert-item__meta" style="display: flex; justify-content: space-between; font-size: 12px; margin-bottom: 4px;">
                            <span class="alert-badge alert-badge--${g.severity}">${g.severity === "critical" ? "严重" : g.severity === "warning" ? "警告" : "通知"}</span>
                            <span class="alert-time">${g.timestamp}</span>
                          </div>
                          <div class="alert-item__title" style="font-weight: 500;">${g.title}</div>
                        </div>
                      `,
                    )}
                  </div>
                `}
          </div>
        </div>

        <div class="ops-card detail-column" style="min-height: 300px;">
          <div class="column-header" style="display: flex; justify-content: space-between; align-items: center;">
            <span>${icons.historyClock} 最近巡检摘要</span>
            <button class="ops-btn ops-btn--ghost" style="padding: 2px 6px; font-size: 12px;" @click=${() => props.onNavigateTab?.("workbench", domain, "inspection")}>
              进入巡检中心 ${icons.chevronRight}
            </button>
          </div>
          <div style="padding: 16px;">
            ${!latestInspection
              ? html`<div class="empty-placeholder">该域暂无巡检记录，可点击右上角按钮执行域巡检。</div>`
              : html`
                  <div class="report-header" style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                    <h3 style="margin: 0; font-size: 14px;">健康分 ${latestInspection.score}/100</h3>
                    <span class="score-badge score-badge--${scoreClass(latestInspection.score)}">
                      ${latestInspection.score >= 90 ? "状态良好" : latestInspection.score >= 75 ? "存在风险" : "异常偏高"}
                    </span>
                  </div>
                  <p class="muted" style="font-size: 12px; margin: 4px 0 12px;">报告生成时间: ${latestInspection.time}</p>
                  <div class="detail-section" style="margin-top: 8px;">
                    <div class="detail-section__header" style="font-size: 12px; font-weight: 600; margin-bottom: 4px;">巡检发现:</div>
                    <div class="detail-section__content highlight" style="font-size: 13px; line-height: 1.5; padding: 10px; border-radius: 4px; background: var(--bg-highlight, rgba(0,0,0,0.02));">
                      ${latestInspection.reportSummary}
                    </div>
                  </div>
                `}
          </div>
        </div>
      </div>
    </div>
  `;
}
