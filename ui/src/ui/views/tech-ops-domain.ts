import { html, nothing } from "lit";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { icons } from "../icons.ts";
import { renderOpsEmpty, renderOpsError } from "../components/ops-status.ts";
import { toSanitizedMarkdownHtml } from "../markdown.ts";
import {
  formatEntityContextFromClusters,
  type OpsEntityGroup,
} from "../ops/entity-config.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderChat, type ChatProps } from "./chat.ts";

export type TechOpsDomainProps = {
  domainKey: "hadoop" | "fi" | "gbase" | "governance" | "dataapps";
  domainName: string;
  activeSubTab: "agent" | "alerts" | "inspections";
  onSubTabChange: (tab: "agent" | "alerts" | "inspections") => void;
  selectedEntityId: string;
  isEntitySelectorOpen: boolean;
  onSelectEntity: (id: string) => void;
  onToggleEntitySelector: () => void;
  entityGroups: OpsEntityGroup[];
  entityGroupsLoading?: boolean;
  domainClusters: OpsClusterRecord[];
  onOpenAssets?: () => void;
  // Chat Props
  chatProps: ChatProps;
  // Alert Props
  alertsApiAvailable?: boolean;
  alertsLoading: boolean;
  alertsError?: string | null;
  alertStats?: {
    originalTotal: number;
    reductionRate: number;
    mergedTotal: number;
  };
  alertGroups: Array<{
    id: string;
    title: string;
    severity: "critical" | "warning" | "info";
    timestamp: string;
    originalCount: number;
    reducedTo: number;
    rootCause: string;
    impact: string;
    analysisMarkdown?: string;
    status: "resolved" | "active";
  }>;
  selectedAlertGroupId: string | null;
  onSelectAlertGroup: (id: string) => void;
  // Inspection Props
  inspectionsLoading: boolean;
  inspections: Array<{
    id: string;
    time: string;
    score: number;
    status: "healthy" | "warning" | "critical";
    reportSummary: string;
    reportMarkdown: string;
  }>;
  selectedInspectionId: string | null;
  onSelectInspection: (id: string) => void;
  onRunInspection: () => void;
  isInspecting: boolean;
  inspectionImStatus?: {
    imConfigured: boolean;
    channels: string[];
    lowScoreThreshold: number;
    hint?: string;
  } | null;
  onOpenChannels?: () => void;
  canInspect?: boolean;
  canAckAlerts?: boolean;
  onAckAlert?: (groupId: string) => void;
};



const DOMAIN_SCENARIOS: Record<string, Array<{ id: "agent" | "alerts" | "inspections"; label: string; icon: string }>> = {
  hadoop: [
    { id: "agent", label: "大数据专家 Agent", icon: "messageSquare" },
    { id: "alerts", label: "大数据组件告警降噪", icon: "zap" },
    { id: "inspections", label: "大数据集群深度巡检", icon: "historyClock" },
  ],
  fi: [
    { id: "agent", label: "FI 智能助手 Agent", icon: "messageSquare" },
    { id: "alerts", label: "FI 告警影响评估", icon: "zap" },
    { id: "inspections", label: "FI 服务健康巡检", icon: "historyClock" },
  ],
  gbase: [
    { id: "agent", label: "SQL 优化专家 Agent", icon: "messageSquare" },
    { id: "alerts", label: "数据库告警关联分析", icon: "zap" },
    { id: "inspections", label: "数据库实例健康巡检", icon: "historyClock" },
  ],
  governance: [
    { id: "agent", label: "治理平台智能助手", icon: "messageSquare" },
    { id: "alerts", label: "流水线与部署告警降噪", icon: "zap" },
    { id: "inspections", label: "开发治理合规度巡检", icon: "historyClock" },
  ],
  dataapps: [
    { id: "agent", label: "数据任务诊断 Agent", icon: "messageSquare" },
    { id: "alerts", label: "数据流告警评估", icon: "zap" },
    { id: "inspections", label: "数据应用健康度巡检", icon: "historyClock" },
  ],
  default: [
    { id: "agent", label: "智能诊断 Agent", icon: "messageSquare" },
    { id: "alerts", label: "告警降噪与影响评估", icon: "zap" },
    { id: "inspections", label: "深度健康巡检", icon: "historyClock" },
  ],
};

export function renderTechOpsDomain(props: TechOpsDomainProps) {
  const entityCtx = formatEntityContextFromClusters(
    props.domainClusters,
    props.selectedEntityId,
  );
  const entityGroups = props.entityGroups;
  const scenarios = DOMAIN_SCENARIOS[props.domainKey] || DOMAIN_SCENARIOS.default;

  return html`
    <div class="ops-domain-container">
      <div class="ops-layout-wrapper">
        <!-- 侧边栏：场景列表 -->
        <div class="ops-sidebar">
          <div class="ops-sidebar__header">
            <div class="ops-sidebar__domain-card ops-sidebar__domain-title">
              <span class="ops-sidebar__domain-icon">${icons[props.domainKey === "hadoop" ? "network" : props.domainKey === "fi" ? "building" : props.domainKey === "gbase" ? "database" : props.domainKey === "governance" ? "layout" : props.domainKey === "dataapps" ? "activity" : "folder"]}</span>
              <span class="ops-sidebar__domain-name">${props.domainName}</span>
            </div>
          </div>
          <div class="ops-sidebar__menu">
            <div class="ops-sidebar__group-label">业务场景</div>
            ${scenarios.map((sc) => {
              const active = props.activeSubTab === sc.id;
              const iconSvg = (icons as any)[sc.icon] || icons.globe;
              return html`
                <button 
                  class="ops-sidebar__menu-item ${active ? "active" : ""}" 
                  @click=${() => props.onSubTabChange(sc.id)}
                >
                  ${iconSvg} <span>${sc.label}</span>
                </button>
              `;
            })}
          </div>
        </div>

        <!-- 页面主要内容区分发 -->
        <div class="ops-main-content">
          <div class="ops-main-header">
            <div class="ops-main-header__left">
              <span class="ops-main-header__breadcrumb-domain">${props.domainName}</span>
              <span class="ops-main-header__breadcrumb-separator">/</span>
              <div class="ops-entity-selector">
                <button
                  type="button"
                  class="ops-entity-selector__current"
                  @click=${() => props.onToggleEntitySelector()}
                >
                  <div class="ops-entity-selector__meta">
                    <span style="color: var(--accent); display: flex;">${icons.server}</span>
                    <div style="min-width: 0;">
                      <div class="ops-entity-selector__title">${entityCtx.title}</div>
                      <div class="ops-entity-selector__subtitle">${entityCtx.subtitle}</div>
                    </div>
                  </div>
                  <span class="ops-entity-selector__chevron">${props.isEntitySelectorOpen ? "▲" : "▼"}</span>
                </button>
                
                ${props.isEntitySelectorOpen
                  ? html`
                      <div class="ops-entity-selector__dropdown">
                        ${props.entityGroupsLoading
                          ? html`
                              <div style="padding: 16px; font-size: 13px; color: var(--text-muted);">
                                加载集群列表…
                              </div>
                            `
                          : entityGroups.length === 0
                            ? html`
                                <div style="padding: 16px;">
                                  ${renderOpsEmpty({
                                    icon: "server",
                                    title: "暂无已登记集群",
                                    description: `请先在「集群资产管理」中为 ${props.domainName} 登记集群。`,
                                    actionLabel: "前往登记",
                                    onAction: props.onOpenAssets,
                                    compact: true,
                                  })}
                                </div>
                              `
                            : entityGroups.map(
                                (group) => html`
                                  <div
                                    style="padding: 8px 12px; font-size: 11px; font-weight: 600; color: var(--text-muted); background: rgba(0,0,0,0.2);"
                                  >
                                    ${group.groupLabel}
                                  </div>
                                  ${group.options.map((opt) => {
                                    const active = props.selectedEntityId === opt.id;
                                    const padLeft = opt.indent ? "24px" : "12px";
                                    return html`
                                      <button
                                        type="button"
                                        class="ops-entity-dropdown-item"
                                        style="padding: 10px 12px 10px ${padLeft}; border: none; background: ${active ? "var(--bg-hover)" : "transparent"}; color: ${opt.indent ? "var(--text-secondary)" : "var(--text-primary)"}; text-align: left; cursor: pointer; font-size: 13px; border-left: 2px solid ${active ? "var(--accent)" : "transparent"};"
                                        @click=${() => props.onSelectEntity(opt.id)}
                                      >
                                        ${opt.indent ? "• " : ""}${opt.label}
                                      </button>
                                    `;
                                  })}
                                `,
                              )}
                      </div>
                    `
                  : nothing}
              </div>
            </div>
            

          </div>
          
          <div style="flex: 1; overflow: hidden; position: relative;">
          ${props.activeSubTab === "agent"
            ? html`
                <div class="ops-agent-view">
                  ${renderChat(props.chatProps)}
                </div>
              `
            : props.activeSubTab === "alerts"
            ? renderAlertsSubTab(props)
            : renderInspectionsSubTab(props)}
          </div>
        </div>
      </div>
    </div>
  `;
}

// 告警降噪评估子 Tab
function renderMarkdownBlock(content: string) {
  const trimmed = content.trim();
  if (!trimmed) {
    return html`<p class="muted">暂无内容</p>`;
  }
  return html`<div class="ops-markdown">${unsafeHTML(toSanitizedMarkdownHtml(trimmed))}</div>`;
}

function renderAlertsSubTab(props: TechOpsDomainProps) {
  const apiReady = props.alertsApiAvailable === true;
  const activeGroup =
    props.alertGroups.find((g) => g.id === props.selectedAlertGroupId) || props.alertGroups[0];
  const originalTotal =
    props.alertStats?.originalTotal ??
    props.alertGroups.reduce((acc, g) => acc + g.originalCount, 0);
  const reducedTotal = props.alertStats?.mergedTotal ?? props.alertGroups.length;
  const reductionRate = (
    props.alertStats?.reductionRate ??
    (originalTotal > 0 ? (1 - reducedTotal / originalTotal) * 100 : 0)
  ).toFixed(1);

  if (!apiReady && !props.alertsLoading) {
    return html`
      <div class="ops-alerts-grid" style="padding: 24px;">
        ${renderOpsEmpty({
          icon: "bell",
          title: "告警 API 未就绪",
          description: "请升级网关以启用告警组存储接口。",
          compact: true,
        })}
      </div>
    `;
  }

  return html`
    <div class="ops-alerts-grid">
      ${props.alertsError
        ? html`<div class="ops-panel" style="margin-bottom: 12px;">${renderOpsError({ message: props.alertsError })}</div>`
        : nothing}
      <div class="ops-summary-cards">
        <div class="ops-card stat-card">
          <div class="stat-label">未合并原始告警</div>
          <div class="stat-value warning">${originalTotal}</div>
          <div class="muted">已入库原始事件条数</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">降噪合并比</div>
          <div class="stat-value ok">${reductionRate}%</div>
          <div class="muted">基于已合并告警组统计</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">已合并告警组</div>
          <div class="stat-value info">${reducedTotal}</div>
          <div class="muted">待处理的核心故障组</div>
        </div>
      </div>

      <div class="ops-main-columns">
        <div class="ops-card list-column">
          <div class="column-header">已合并告警列表</div>
          ${props.alertsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : props.alertGroups.length === 0
            ? html`<div class="empty-placeholder">暂无活动告警。</div>`
            : html`
                <div class="alert-list">
                  ${props.alertGroups.map(
                    g => html`
                      <div 
                        class="alert-item ${g.id === props.selectedAlertGroupId ? "alert-item--active" : ""} alert-item--${g.severity}"
                        @click=${() => props.onSelectAlertGroup(g.id)}
                      >
                        <div class="alert-item__meta">
                          <span class="alert-badge alert-badge--${g.severity}">
                            ${g.severity === "critical" ? "严重" : g.severity === "warning" ? "警告" : "通知"}
                          </span>
                          <span class="alert-time">${g.timestamp}</span>
                        </div>
                        <div class="alert-item__title">${g.title}</div>
                        <div class="alert-item__noise">
                          <span>原始告警: <strong>${g.originalCount}</strong> 次</span>
                          <span class="divider">|</span>
                          <span class="rate">降噪比: <strong>${((1 - g.reducedTo / g.originalCount) * 100).toFixed(0)}%</strong></span>
                        </div>
                      </div>
                    `
                  )}
                </div>
              `}
        </div>

        <!-- 右侧：AI 诊断详情 -->
        <div class="ops-card detail-column">
          <div class="column-header">AI 智能根因与影响评估</div>
          ${!activeGroup
            ? html`<div class="empty-placeholder">请从左侧选择一个告警组以查看诊断报告。</div>`
            : html`
                <div class="alert-detail">
                  <div class="detail-section">
                    <div class="detail-section__title">${activeGroup.title}</div>
                    <div class="detail-meta">
                      <span class="detail-time">生成时间: ${activeGroup.timestamp}</span>
                      <span class="detail-count">关联原始事件: ${activeGroup.originalCount} 次</span>
                    </div>
                  </div>

                  <div class="detail-section">
                    <div class="detail-section__header">${icons.zap} 根因分析</div>
                    <div class="detail-section__content highlight">
                      ${renderMarkdownBlock(activeGroup.analysisMarkdown || activeGroup.rootCause)}
                    </div>
                  </div>

                  ${activeGroup.impact && activeGroup.impact !== "—"
                    ? html`
                        <div class="detail-section">
                          <div class="detail-section__header">${icons.overviewGrid} 影响范围</div>
                          <div class="detail-section__content">
                            ${renderMarkdownBlock(activeGroup.impact)}
                          </div>
                        </div>
                      `
                    : nothing}

                  ${props.canAckAlerts && activeGroup.status === "active" && props.onAckAlert
                    ? html`
                        <div class="detail-section">
                          <button
                            type="button"
                            class="ops-btn ops-btn--primary"
                            @click=${() => props.onAckAlert!(activeGroup.id)}
                          >
                            标记为已处理
                          </button>
                        </div>
                      `
                    : nothing}
                  <div class="detail-section">
                    <div class="detail-section__header">${icons.scrollText} 降噪说明</div>
                    <div class="detail-section__content">
                      <p>
                        本组合并了 <strong>${activeGroup.originalCount}</strong> 条原始告警（降噪比
                        <strong>${activeGroup.originalCount > 0 ? ((1 - activeGroup.reducedTo / activeGroup.originalCount) * 100).toFixed(0) : 0}%</strong>）。
                        完整分析由 Agent 会话生成，可在左侧切换其他告警组。
                      </p>
                    </div>
                  </div>
                </div>
              `}
        </div>
      </div>
    </div>
  `;
}

function renderInspectionsSubTab(props: TechOpsDomainProps) {
  const activeInspection = props.inspections.find(ins => ins.id === props.selectedInspectionId) || props.inspections[0];
  const lastScore = props.inspections[0]?.score;
  const hasScore = lastScore !== undefined && lastScore !== null && lastScore >= 0;
  const scoreDisplay = hasScore ? `${lastScore}` : "未知";
  const statusClass = hasScore ? (lastScore >= 90 ? "ok" : lastScore >= 75 ? "warning" : "danger") : "unknown";

  return html`
    <div class="ops-inspections-grid">
      ${props.inspectionImStatus && !props.inspectionImStatus.imConfigured
        ? html`
            <div class="ops-banner" style="margin-bottom: 12px;">
              <span class="ops-banner__icon">${icons.info}</span>
              <span>
                ${props.inspectionImStatus.hint ??
                `健康分低于 ${props.inspectionImStatus.lowScoreThreshold} 时将尝试推送 IM，当前未启用飞书/钉钉。`}
                ${props.onOpenChannels
                  ? html`
                      <button
                        type="button"
                        class="ops-btn"
                        style="margin-left: 8px;"
                        @click=${() => props.onOpenChannels?.()}
                      >
                        前往通道配置
                      </button>
                    `
                  : nothing}
              </span>
            </div>
          `
        : nothing}
      <!-- 顶部控制面板 -->
      <div class="ops-inspection-ctrl ops-card">
        <div class="ctrl-left">
          <div class="health-index-badge health-index-badge--${statusClass}">
            <div class="health-score">${scoreDisplay}</div>
            <div class="health-label">健康度得分</div>
          </div>
          <div class="ctrl-meta">
            <h3>巡检自动化配置</h3>
            <p>定时策略：每天 08:00 和 20:00 自动触发深度健康巡检。</p>
            <p>最近一次执行：${props.inspections[0]?.time ?? "暂无执行记录"}</p>
          </div>
        </div>
        <div class="ctrl-right">
          <button 
            class="btn primary large ${props.isInspecting ? "btn--loading" : ""}" 
            type="button" 
            ?disabled=${props.isInspecting || props.canInspect === false}
            title=${props.canInspect === false ? "当前账号无 ops:inspect 权限" : ""}
            @click=${props.onRunInspection}
          >
            ${props.isInspecting ? html`${icons.loader} 正在进行深度巡检...` : html`${icons.zap} 一键手动巡检`}
          </button>
        </div>
      </div>

      <!-- 下方历史报告列表与详情 -->
      <div class="ops-main-columns">
        <!-- 左侧：历史巡检报告列表 -->
        <div class="ops-card list-column">
          <div class="column-header">历史巡检报告</div>
          ${props.inspectionsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : props.inspections.length === 0
            ? html`<div class="empty-placeholder">暂无巡检记录，请点击上方按钮开始首次巡检。</div>`
            : html`
                <div class="inspection-list">
                  ${props.inspections.map(
                    ins => html`
                      <div 
                        class="inspection-item ${ins.id === props.selectedInspectionId ? "inspection-item--active" : ""}"
                        @click=${() => props.onSelectInspection(ins.id)}
                      >
                        <div class="inspection-item__meta">
                          <span class="score-badge score-badge--${ins.score !== undefined && ins.score !== null && ins.score >= 90 ? "ok" : ins.score !== undefined && ins.score !== null && ins.score >= 75 ? "warning" : ins.score !== undefined && ins.score !== null && ins.score >= 0 ? "danger" : "unknown"}">
                            ${ins.score !== undefined && ins.score !== null && ins.score >= 0 ? `${ins.score}分` : "未知"}
                          </span>
                          <span class="inspection-time">${ins.time}</span>
                        </div>
                        <div class="inspection-summary">${ins.reportSummary}</div>
                      </div>
                    `
                  )}
                </div>
              `}
        </div>

        <!-- 右侧：报告详情 -->
        <div class="ops-card detail-column">
          <div class="column-header">健康巡检报告详情</div>
          ${!activeInspection
            ? html`<div class="empty-placeholder">请从左侧选择一份巡检报告以查看完整指标。</div>`
            : html`
                <div class="inspection-detail-report">
                  <div class="report-header">
                    <h3>${props.domainName} 巡检报告 (${activeInspection.time})</h3>
                    <span class="report-score score-badge--${activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 90 ? "ok" : activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 75 ? "warning" : activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 0 ? "danger" : "unknown"}">
                      健康得分：${activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 0 ? `${activeInspection.score} / 100` : "未知"}
                    </span>
                  </div>
                  <div class="report-body">
                    <!-- 使用简单的段落渲染巡检总结与指标 -->
                    <div class="report-section">
                      <div class="report-section__title">巡检发现摘要</div>
                      <p>${activeInspection.reportSummary}</p>
                    </div>

                    ${(activeInspection as any).result?.errors && (activeInspection as any).result.errors.length > 0
                      ? html`
                          <div class="report-section report-section--errors">
                            <div class="report-section__title">异常错误日志</div>
                            <ul class="error-list">
                              ${(activeInspection as any).result.errors.map((err: string) => html`<li>${err}</li>`)}
                            </ul>
                          </div>
                        `
                      : ""}

                    ${(activeInspection as any).result?.toolRuns && (activeInspection as any).result.toolRuns.length > 0
                      ? html`
                          <div class="report-section">
                            <div class="report-section__title">工具执行证据</div>
                            <div class="tool-runs-container">
                              ${(activeInspection as any).result.toolRuns.map((run: any) => html`
                                <div class="tool-run-card ${run.success ? "tool-run-card--success" : "tool-run-card--fail"}">
                                  <div class="tool-run-card__header">
                                    <span class="tool-run-card__name">${run.toolName}</span>
                                    <span class="tool-run-card__badge">${run.success ? "成功" : "失败"}</span>
                                  </div>
                                  ${run.error
                                    ? html`<div class="tool-run-card__error">${run.error}</div>`
                                    : run.output
                                      ? html`<pre class="tool-run-card__output"><code>${run.output}</code></pre>`
                                      : ""}
                                </div>
                              `)}
                            </div>
                          </div>
                        `
                      : ""}

                    <div class="report-section">
                      <div class="report-section__title">完整巡检报告</div>
                      ${renderMarkdownBlock(activeInspection.reportMarkdown)}
                    </div>
                  </div>
                </div>
              `}
        </div>
      </div>
    </div>
  `;
}
