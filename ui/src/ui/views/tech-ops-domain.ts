import { html, nothing } from "lit";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { icons } from "../icons.ts";
import "./ops/bch-cluster-overview.ts";
import "./ops/bch-flink-diagnosis.ts";
import "./ops/bch-fsimage-dashboard.ts";
import "./ops/bch-employee-workstation.ts";
import { renderOpsEmpty, renderOpsError } from "../components/ops-status.ts";
import { buildInspectionListPreview } from "../ops/inspection-report.ts";
import { toSanitizedMarkdownHtml } from "../markdown.ts";
import {
  formatEntityContextFromClusters,
  type OpsEntityGroup,
} from "../ops/entity-config.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import { renderChat, type ChatProps } from "./chat.ts";
import { opsDomainIcon } from "../components/domain-filter.ts";
import type { TechOpsCapabilityTab } from "../ops/navigation.ts";

export type { TechOpsCapabilityTab } from "../ops/navigation.ts";

export type TechOpsDomainProps = {
  domainKey: "hadoop" | "fi" | "gbase" | "governance" | "dataapps";
  domainName: string;
  activeSubTab: TechOpsCapabilityTab;
  onSubTabChange: (tab: TechOpsCapabilityTab) => void;
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
    status: "healthy" | "warning" | "critical" | "unknown";
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
  canAckAlerts?: boolean;
  onAckAlert?: (groupId: string) => void;
  host?: any;
};

type CapabilityNavItem = {
  id: TechOpsCapabilityTab;
  label: string;
  icon: string;
};

const DOMAIN_CAPABILITIES: Record<string, CapabilityNavItem[]> = {
  hadoop: [
    { id: "overview", label: "概览", icon: "overviewGrid" },
    { id: "assetTopology", label: "资产与拓扑", icon: "server" },
    { id: "observability", label: "可观测与告警", icon: "zap" },
    { id: "inspection", label: "健康度与巡检", icon: "historyClock" },
    { id: "jobGovernance", label: "作业治理", icon: "activity" },
    { id: "diagnosis", label: "故障诊断与应急", icon: "messageSquare" },
    { id: "governance", label: "治理与优化", icon: "layout" },
    { id: "capacity", label: "容量性能与成本", icon: "usageBars" },
    { id: "change", label: "变更配置与合规", icon: "settings" },
    { id: "employees", label: "数字员工", icon: "users" },
  ],
  default: [
    { id: "overview", label: "概览", icon: "overviewGrid" },
    { id: "assetTopology", label: "资产与拓扑", icon: "server" },
    { id: "observability", label: "可观测与告警", icon: "zap" },
    { id: "inspection", label: "健康度与巡检", icon: "historyClock" },
    { id: "diagnosis", label: "故障诊断与应急", icon: "messageSquare" },
    { id: "governance", label: "治理与优化", icon: "layout" },
    { id: "capacity", label: "容量性能与成本", icon: "usageBars" },
    { id: "change", label: "变更配置与合规", icon: "settings" },
    { id: "employees", label: "数字员工", icon: "users" },
  ],
};

export function renderTechOpsDomain(props: TechOpsDomainProps) {
  const entityCtx = formatEntityContextFromClusters(
    props.domainClusters,
    props.selectedEntityId,
  );
  const entityGroups = props.entityGroups;
  const capabilities = DOMAIN_CAPABILITIES[props.domainKey] || DOMAIN_CAPABILITIES.default;
  const activeCapability = capabilities.find((item) => item.id === props.activeSubTab) ?? capabilities[0];

  return html`
    <div class="ops-domain-container">
      <div class="ops-layout-wrapper">
        <div class="ops-sidebar">
          <div class="ops-sidebar__header">
            <div class="ops-sidebar__domain-card ops-sidebar__domain-title">
              <span class="ops-nav-icon" aria-hidden="true">${icons[opsDomainIcon(props.domainKey)]}</span>
              <span class="ops-sidebar__domain-name">${props.domainName}</span>
            </div>
          </div>
          <div class="ops-sidebar__menu">
            <div class="ops-sidebar-section__label">运维能力域</div>
            <nav class="ops-sidebar-nav">
              ${capabilities.map((capability) => {
                const active = props.activeSubTab === capability.id;
                const iconSvg = (icons as Record<string, typeof icons.globe>)[capability.icon] ?? icons.globe;
                return html`
                  <button
                    type="button"
                    class="ops-sidebar-nav-item ${active ? "ops-sidebar-nav-item--active" : ""}"
                    @click=${() => props.onSubTabChange(capability.id)}
                  >
                    <span class="ops-nav-icon" aria-hidden="true">${iconSvg}</span>
                    <span class="ops-sidebar-nav-item__label">${capability.label}</span>
                  </button>
                `;
              })}
            </nav>
          </div>
        </div>

        <div class="ops-main-content">
          <!-- Migrated Warning Banner -->
          <div class="ops-banner warning" style="margin: 12px 16px; border-radius: 6px; padding: 10px 16px; background: rgba(255, 152, 0, 0.08); border: 1px dashed rgba(255, 152, 0, 0.4); display: flex; justify-content: space-between; align-items: center; box-sizing: border-box;">
            <div style="display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 500; color: #e65100;">
              <span style="display: flex;">${icons.info}</span>
              <span><strong>升级提示</strong>：该独立入口已合并。技术域的告警、巡检、优化和数字员工等能力，已全面升级迁移至<strong>“运维工作台”</strong>与<strong>“技术域详情”</strong>。</span>
            </div>
            <div style="display: flex; gap: 8px; flex-shrink: 0;">
              <button type="button" class="ops-btn ops-btn--primary" style="padding: 4px 10px; font-size: 12px;" @click=${() => props.host?.setTab?.("domainInsight")}>进入域详情</button>
              <button type="button" class="ops-btn" style="padding: 4px 10px; font-size: 12px;" @click=${() => props.host?.setTab?.("workbench")}>进入工作台</button>
            </div>
          </div>

          <div class="ops-main-header">
            <div class="ops-main-header__left">
              <span class="ops-main-header__breadcrumb-domain">${props.domainName}</span>
              <span class="ops-main-header__breadcrumb-separator">/</span>
              <span class="ops-main-header__breadcrumb-domain">${activeCapability?.label ?? "能力域"}</span>
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
          
          <div style="flex: 1; overflow: visible; position: relative;">
          ${props.activeSubTab === "diagnosis"
            ? html`
                <div class="ops-agent-view">
                  ${renderChat(props.chatProps)}
                </div>
              `
            : props.activeSubTab === "observability"
            ? renderAlertsSubTab(props)
            : props.activeSubTab === "inspection"
            ? renderInspectionsSubTab(props)
            : props.domainKey === "hadoop" && props.activeSubTab === "overview"
            ? html`<bch-cluster-overview .host=${props.host}></bch-cluster-overview>`
            : props.domainKey === "hadoop" && props.activeSubTab === "jobGovernance"
            ? html`<bch-flink-diagnosis .host=${props.host}></bch-flink-diagnosis>`
            : props.domainKey === "hadoop" && props.activeSubTab === "capacity"
            ? html`<bch-fsimage-dashboard .host=${props.host}></bch-fsimage-dashboard>`
            : props.domainKey === "hadoop" && props.activeSubTab === "employees"
            ? html`<bch-employee-workstation .host=${props.host}></bch-employee-workstation>`
            : renderCapabilityPlaceholder(props)}
          </div>
        </div>
      </div>
    </div>
  `;
}

function renderCapabilityPlaceholder(props: TechOpsDomainProps) {
  const content = getCapabilityPlaceholder(props.domainKey, props.activeSubTab);
  return html`
    <div class="ops-alerts-grid" style="padding: 24px;">
      <div class="ops-card detail-column">
        <div class="detail-section__header">${content.title}</div>
        <p class="muted" style="margin-top: 10px;">${content.description}</p>
        <div class="ops-summary-cards" style="margin-top: 16px;">
          ${content.scenarios.map(
            (scenario) => html`
              <div class="ops-card stat-card">
                <div class="stat-label">${scenario.title}</div>
                <div class="muted" style="margin-top: 6px; line-height: 1.6;">${scenario.desc}</div>
              </div>
            `,
          )}
        </div>
      </div>
    </div>
  `;
}

function getCapabilityPlaceholder(domainKey: string, tab: TechOpsCapabilityTab) {
  const bch = domainKey === "hadoop";
  const common: Record<TechOpsCapabilityTab, { title: string; description: string; scenarios: Array<{ title: string; desc: string }> }> = {
    overview: {
      title: "技术域概览",
      description: "聚合该技术域的健康度、风险、告警、巡检、容量和自动化执行状态。",
      scenarios: [
        { title: "健康度矩阵", desc: "按集群、组件、作业或实例展示健康评分与风险等级。" },
        { title: "待处理风险", desc: "汇总告警、巡检、容量和变更风险，形成统一待办。" },
        { title: "自动化执行状态", desc: "展示本域自动化任务正在处理的任务、产出和异常。" },
      ],
    },
    assetTopology: {
      title: "资产与拓扑",
      description: "管理对象、关系、归属、配置和依赖，是观测、诊断、治理的基础。",
      scenarios: bch
        ? [
            { title: "集群资产", desc: "纳管 BCH 集群、区域、责任人、版本和状态。" },
            { title: "组件拓扑", desc: "维护 HDFS、YARN、Hive、Spark、Flink 等组件关系。" },
            { title: "队列与租户", desc: "沉淀 YARN 队列、租户、资源配额和责任归属。" },
          ]
        : [
            { title: "对象资产", desc: "纳管该技术域的核心实例、组件、服务和责任人。" },
            { title: "依赖拓扑", desc: "维护上下游依赖，为告警关联和影响分析提供基础。" },
          ],
    },
    observability: {
      title: "可观测与告警",
      description: "该能力域已接入当前告警降噪与影响评估页面。",
      scenarios: [],
    },
    inspection: {
      title: "健康度与巡检",
      description: "该能力域已接入当前深度巡检与报告页面。",
      scenarios: [],
    },
    jobGovernance: {
      title: "作业治理",
      description: "面向大数据作业稳定性、性能、SLA 和资源使用的治理能力。该能力域主要适用于 BCH 生态。",
      scenarios: [
        { title: "Flink 作业健康度", desc: "评估 checkpoint、延迟、反压、失败率和资源水位。" },
        { title: "Spark 作业诊断", desc: "分析失败原因、资源倾斜、长尾 task 和执行效率。" },
        { title: "作业 SLA 风险", desc: "识别即将超时、频繁失败或资源异常的关键作业。" },
      ],
    },
    diagnosis: {
      title: "故障诊断与应急",
      description: "该能力域承载当前专家对话，并逐步升级为诊断任务、根因分析和应急处置闭环。",
      scenarios: [],
    },
    governance: {
      title: "治理与优化",
      description: "面向长期问题治理，而不是单次告警或单次故障处理。",
      scenarios: [
        { title: "重复告警治理", desc: "识别长期重复、低价值、无责任归属的告警。" },
        { title: "稳定性治理", desc: "跟踪高频失败对象、弱依赖和反复出现的风险。" },
        { title: "配置治理", desc: "发现配置漂移、基线偏差和不合理参数。" },
      ],
    },
    capacity: {
      title: "容量性能与成本",
      description: "面向资源水位、容量预测、性能瓶颈和成本归因。",
      scenarios: [
        { title: "容量预测", desc: "基于增长趋势预测存储、计算、连接数或任务量风险。" },
        { title: "性能瓶颈", desc: "识别热点、队列拥塞、慢 SQL、长尾任务或资源争用。" },
        { title: "成本归因", desc: "按集群、租户、作业或应用归因资源消耗。" },
      ],
    },
    change: {
      title: "变更配置与合规",
      description: "覆盖变更前评估、变更中护航、变更后验证和配置合规。",
      scenarios: [
        { title: "变更前风险评估", desc: "检查影响范围、依赖、历史风险和回滚条件。" },
        { title: "变更中观测", desc: "自动关注关键指标、告警和用户影响。" },
        { title: "变更后验证", desc: "验证服务恢复、容量稳定、关键链路正常。" },
      ],
    },
    employees: {
      title: "数字员工",
      description: "按该技术域展示值班、巡检、诊断、治理、容量和变更护航数字员工。",
      scenarios: bch
        ? [
            { title: "BCH 值班运维数字员工", desc: "负责告警接收、聚合、初判和升级。" },
            { title: "BCH 深度巡检数字员工", desc: "负责周期巡检、健康评分和风险清单。" },
            { title: "BCH 作业诊断数字员工", desc: "负责 Spark、Flink 和离线任务诊断。" },
          ]
        : [
            { title: "值班运维数字员工", desc: "负责该技术域告警和事件的初步处理。" },
            { title: "巡检数字员工", desc: "负责该技术域健康检查与风险发现。" },
            { title: "诊断数字员工", desc: "负责故障定位、根因分析和处置建议。" },
          ],
    },
  };
  return common[tab] ?? common.overview;
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

  const imHint =
    props.inspectionImStatus && !props.inspectionImStatus.imConfigured
      ? props.inspectionImStatus.hint ??
        `健康分低于 ${props.inspectionImStatus.lowScoreThreshold} 时将推送 IM，需先配置飞书/钉钉。`
      : null;

  return html`
    <div class="ops-inspections-grid">
      <div class="ops-inspection-toolbar ops-inspection-toolbar--bar">
        <div class="ops-inspection-toolbar__status">
          <span class="health-index-badge health-index-badge--compact health-index-badge--${statusClass}">
            <span class="health-score">${scoreDisplay}</span>
            <span class="health-label">最新健康分</span>
          </span>
          <span class="ops-inspection-toolbar__meta muted">
            最近：${props.inspections[0]?.time ?? "—"} · 定时 08:00 / 20:00
          </span>
        </div>
        <div class="ops-inspection-toolbar">
          ${imHint && props.onOpenChannels
            ? html`
                <button
                  type="button"
                  class="ops-btn ops-btn--ghost ops-inspection-toolbar__hint"
                  title=${imHint}
                  @click=${() => props.onOpenChannels?.()}
                >
                  ${icons.info} IM 通道
                </button>
              `
            : nothing}
          <button
            class="btn primary ${props.isInspecting ? "btn--loading" : ""}"
            type="button"
            ?disabled=${props.isInspecting || props.canInspect === false}
            title=${props.canInspect === false ? "当前账号无 ops:inspect 权限" : ""}
            @click=${props.onRunInspection}
          >
            ${props.isInspecting ? html`${icons.loader} 巡检中...` : html`${icons.zap} 一键巡检`}
          </button>
        </div>
      </div>

      <div class="ops-main-columns workbench-inspection-layout">
        <div class="list-column ops-inspection-list-column">
          <div class="minimal-column-header">
            <span>巡检报告</span>
            <span class="minimal-column-stats">${props.inspections.length} 条记录</span>
          </div>
          ${props.inspectionsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : props.inspections.length === 0
            ? html`<div class="empty-placeholder">暂无巡检记录，点击「一键巡检」开始。</div>`
            : html`
                <div class="inspection-list minimal-inspection-list">
                  ${props.inspections.map(
                    ins => html`
                      <div 
                        class="inspection-item minimal-inspection-item ${ins.id === props.selectedInspectionId ? "inspection-item--active" : ""}"
                        @click=${() => props.onSelectInspection(ins.id)}
                      >
                        <div class="inspection-item__meta">
                          <span class="score-badge score-badge--${ins.score !== undefined && ins.score !== null && ins.score >= 90 ? "ok" : ins.score !== undefined && ins.score !== null && ins.score >= 75 ? "warning" : ins.score !== undefined && ins.score !== null && ins.score >= 0 ? "danger" : "unknown"}">
                            ${ins.score !== undefined && ins.score !== null && ins.score >= 0 ? `${ins.score}分` : "未知"}
                          </span>
                          <span class="inspection-time">${ins.time}</span>
                        </div>
                        <div class="inspection-summary">${buildInspectionListPreview(ins.reportSummary)}</div>
                      </div>
                    `
                  )}
                </div>
              `}
        </div>

        <div class="detail-column minimal-detail-column ops-inspection-detail">
          <div class="minimal-column-header">
            <span>报告详情</span>
            ${activeInspection
              ? html`
                  <span class="score-badge score-badge--${activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 90 ? "ok" : activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 75 ? "warning" : activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 0 ? "danger" : "unknown"}">
                    ${activeInspection.score !== undefined && activeInspection.score !== null && activeInspection.score >= 0 ? `${activeInspection.score}/100` : "未知"}
                  </span>
                `
              : nothing}
          </div>
          ${!activeInspection
            ? html`<div class="empty-placeholder">从左侧选择一份巡检报告。</div>`
            : html`
                <div class="ops-inspection-detail__body">
                  <div class="detail-section ops-inspection-detail__summary">
                    <div class="detail-section__header">巡检结论</div>
                    <div class="detail-section__content">
                      ${(() => {
                        const bullets = activeInspection.reportSummary
                          .split("\n")
                          .map((line) => line.trim())
                          .filter(Boolean);
                        if (bullets.length <= 1) {
                          return html`<p class="ops-inspection-detail__summary-text">${activeInspection.reportSummary}</p>`;
                        }
                        return html`
                          <ul class="ops-inspection-summary-list">
                            ${bullets.map((line) => html`<li>${line}</li>`)}
                          </ul>
                        `;
                      })()}
                    </div>
                  </div>

                  ${(activeInspection as any).result?.errors && (activeInspection as any).result.errors.length > 0
                    ? html`
                        <div class="detail-section report-section--errors">
                          <div class="detail-section__header">异常错误</div>
                          <ul class="error-list">
                            ${(activeInspection as any).result.errors.map((err: string) => html`<li>${err}</li>`)}
                          </ul>
                        </div>
                      `
                    : nothing}

                  <div class="detail-section ops-inspection-detail__report">
                    <div class="detail-section__header">完整报告</div>
                    <div class="detail-section__content ops-inspection-detail__report-body">
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
