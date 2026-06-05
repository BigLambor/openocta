import { html, nothing, type TemplateResult } from "lit";
import { icons } from "../icons.ts";
import { renderOpsError } from "../components/ops-status.ts";
import {
  renderOpsShellHeader,
  renderOpsShellStatGrid,
  renderOpsViewNav,
  type OpsViewNavItem,
} from "../components/ops-shell.ts";

export type WorkbenchAlertGroup = {
  id: string;
  title: string;
  severity: "critical" | "warning" | "info";
  timestamp: string;
  originalCount: number;
  reducedTo: number;
  rootCause: string;
  analysisMarkdown?: string;
  impact?: string;
  status?: string;
};

export type WorkbenchView = "events" | "inspection" | "diagnosis" | "governance" | "capacity" | "change";

export type WorkbenchInspection = {
  id: string;
  time: string;
  score: number | null;
  status: "healthy" | "warning" | "critical" | "unknown";
  reportSummary: string;
  reportMarkdown: string;
};

export type WorkbenchProps = {
  domainName: string;
  domainFilter?: TemplateResult;
  activeView?: WorkbenchView;
  alertsLoading?: boolean;
  alertsError?: string | null;
  alertGroups: WorkbenchAlertGroup[];
  inspectionsLoading?: boolean;
  inspections?: WorkbenchInspection[];
  selectedInspectionId?: string | null;
  inspectionImStatus?: {
    imConfigured: boolean;
    lowScoreThreshold: number;
    hint?: string;
  } | null;
  isInspecting?: boolean;
  canInspect?: boolean;
  selectedAlertGroupId?: string | null;
  aiPanelOpen?: boolean;
  aiPanelMode?: "root-cause" | "similar" | "action";
  onViewChange?: (view: WorkbenchView) => void;
  onRefreshAlerts?: () => void;
  onSelectAlertGroup?: (id: string) => void;
  onSelectInspection?: (id: string) => void;
  onRunInspection?: () => void;
  onOpenChannels?: () => void;
  onOpenAiPanel?: (mode: "root-cause" | "similar" | "action") => void;
  onSendToCopilot?: (id: string, mode: "root-cause" | "similar" | "action") => void;
  onCloseAiPanel?: () => void;
  onAcceptSuggestion?: (id: string) => void;
  onRejectSuggestion?: (id: string) => void;
  onOpenTasks?: (id: string) => void;
};

const WORKBENCH_VIEWS: OpsViewNavItem<WorkbenchView>[] = [
  { id: "events", label: "事件中心", icon: "messageSquare" },
  { id: "inspection", label: "巡检中心", icon: "historyClock" },
  { id: "diagnosis", label: "诊断中心", icon: "bug" },
  { id: "governance", label: "治理中心", icon: "layout" },
  { id: "capacity", label: "容量性能", icon: "usageBars" },
  { id: "change", label: "变更护航", icon: "settings" },
];

const WORKBENCH_VIEW_META: Record<
  WorkbenchView,
  { title: string; description: string }
> = {
  events: {
    title: "事件中心",
    description: "从待处理告警进入根因分析、处置建议和执行记录，AI 能力嵌入当前工作流。",
  },
  inspection: {
    title: "巡检中心",
    description: "触发健康巡检、查看报告与风险摘要，低分结果可推送 IM 通道。",
  },
  diagnosis: {
    title: "诊断中心",
    description: "汇聚告警、指标、日志与拓扑，输出根因候选与验证步骤。",
  },
  governance: {
    title: "治理中心",
    description: "识别重复告警、稳定性与配置漂移，沉淀治理任务与采纳效果。",
  },
  capacity: {
    title: "容量性能",
    description: "关注资源利用率、容量水位与性能瓶颈，给出扩缩容与优化建议。",
  },
  change: {
    title: "变更护航",
    description: "变更前评估、变更中观测与回滚建议，对接审批与执行记录。",
  },
};

function severityLabel(severity: WorkbenchAlertGroup["severity"]): string {
  switch (severity) {
    case "critical":
      return "严重";
    case "warning":
      return "警告";
    default:
      return "通知";
  }
}

function markdownToLines(text: string): string[] {
  return (text || "")
    .replace(/```[\s\S]*?```/g, "")
    .split(/\n+/)
    .map((line) => line.replace(/^[-#*\s]+/, "").trim())
    .filter(Boolean)
    .slice(0, 6);
}

function renderAiPanel(props: WorkbenchProps, active: WorkbenchAlertGroup | undefined) {
  if (!props.aiPanelOpen || !active) {
    return nothing;
  }
  const mode = props.aiPanelMode ?? "root-cause";
  const lines = markdownToLines(active.analysisMarkdown || active.rootCause);
  const title =
    mode === "similar"
      ? "聚合相似告警"
      : mode === "action"
        ? "处置建议"
        : "分析根因";
  const recommendation =
    mode === "similar"
      ? `本组已聚合 ${active.originalCount} 条原始告警，建议优先按共同根因处理，避免逐条关闭。`
      : mode === "action"
        ? "建议先确认影响范围，再按 Runbook 执行只读检查；涉及变更或脚本执行时需人工审批。"
        : active.rootCause || "当前告警缺少明确根因，建议补充指标、日志和拓扑上下文后再次分析。";

  return html`
    <aside class="ops-card ops-shell-side detail-column">
      <div class="column-header" style="display:flex; align-items:center; justify-content:space-between; gap:12px;">
        <span>${icons.messageSquare} ${title}</span>
        <button class="ops-btn ops-btn--ghost" type="button" @click=${props.onCloseAiPanel}>关闭</button>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.users} 数字员工模板</div>
        <div class="detail-section__content">
          <strong>BCH 值班数字员工</strong>
          <p class="muted">专家人设：BCH 告警降噪、根因候选、影响面判断、处置建议。</p>
        </div>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.zap} 结论</div>
        <div class="detail-section__content highlight">${recommendation}</div>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.scrollText} 证据</div>
        <div class="detail-section__content">
          ${lines.length
            ? html`<ul>${lines.map((line) => html`<li>${line}</li>`)}</ul>`
            : html`<p class="muted">暂无可用分析证据。</p>`}
        </div>
      </div>
      <div class="detail-section">
        <div style="display:flex; gap:8px; flex-wrap:wrap;">
          <button class="ops-btn ops-btn--primary" type="button" @click=${() => props.onAcceptSuggestion?.(active.id)}>
            确认建议并记录
          </button>
          <button class="ops-btn" type="button" @click=${() => props.onRejectSuggestion?.(active.id)}>
            驳回建议并记录
          </button>
          <button class="ops-btn ops-btn--ghost" type="button" @click=${() => props.onOpenTasks?.(active.id)}>
            查看执行记录
          </button>
        </div>
      </div>
    </aside>
  `;
}


function scoreClass(score: number | null | undefined): string {
  if (score == null || score < 0) return "unknown";
  if (score >= 90) return "ok";
  if (score >= 75) return "warning";
  return "danger";
}

function renderInspectionView(props: WorkbenchProps) {
  const inspections = props.inspections ?? [];
  const active =
    inspections.find((item) => item.id === props.selectedInspectionId) || inspections[0];
  return html`
    <div class="ops-inspections-grid">
      ${props.inspectionImStatus && !props.inspectionImStatus.imConfigured
        ? html`
            <div class="ops-banner" style="margin-bottom: 12px;">
              <span class="ops-banner__icon">${icons.info}</span>
              <span>
                ${props.inspectionImStatus.hint ??
                `健康分低于 ${props.inspectionImStatus.lowScoreThreshold} 时将尝试推送 IM，当前未配置通道。`}
                ${props.onOpenChannels
                  ? html`<button type="button" class="ops-btn" style="margin-left: 8px;" @click=${props.onOpenChannels}>前往通道配置</button>`
                  : nothing}
              </span>
            </div>
          `
        : nothing}

      <div class="ops-shell-panel" style="margin-bottom: 14px;">
        <div class="ops-shell-panel__head">${icons.zap} 巡检控制台</div>
        <div style="padding:16px;">
        <div style="display:flex; justify-content:space-between; gap:18px; flex-wrap:wrap; align-items:center;">
          <div>
            <div class="detail-section__title">健康巡检</div>
            <p class="muted">支持手动触发当前技术域巡检，巡检报告用于风险解释和治理建议。</p>
          </div>
          <button
            class="ops-btn ops-btn--primary ${props.isInspecting ? "btn--loading" : ""}"
            type="button"
            ?disabled=${props.isInspecting || props.canInspect === false}
            title=${props.canInspect === false ? "当前账号无 ops:inspect 权限" : ""}
            @click=${props.onRunInspection}
          >
            ${props.isInspecting ? html`${icons.loader} 巡检中...` : html`${icons.zap} 一键巡检`}
          </button>
        </div>
        </div>
      </div>

      <div class="ops-main-columns ops-shell-columns">
        <div class="ops-card list-column">
          <div class="column-header">巡检报告</div>
          ${props.inspectionsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : inspections.length === 0
              ? html`<div class="empty-placeholder">暂无巡检记录，可先执行一次手动巡检。</div>`
              : html`
                  <div class="inspection-list">
                    ${inspections.map(
                      (item) => html`
                        <button
                          type="button"
                          class="inspection-item ${item.id === active?.id ? "inspection-item--active" : ""}"
                          @click=${() => props.onSelectInspection?.(item.id)}
                        >
                          <div class="inspection-item__meta">
                            <span class="score-badge score-badge--${scoreClass(item.score)}">
                              ${item.score == null || item.score < 0 ? "未知" : `${item.score}分`}
                            </span>
                            <span class="inspection-time">${item.time}</span>
                          </div>
                          <div class="inspection-summary">${item.reportSummary}</div>
                        </button>
                      `,
                    )}
                  </div>
                `}
        </div>
        <div class="ops-card detail-column">
          <div class="column-header">报告详情</div>
          ${!active
            ? html`<div class="empty-placeholder">请选择一份巡检报告。</div>`
            : html`
                <div class="report-header">
                  <h3>${props.domainName} 巡检报告</h3>
                  <span class="score-badge score-badge--${scoreClass(active.score)}">
                    ${active.score == null || active.score < 0 ? "健康分未知" : `健康分 ${active.score}/100`}
                  </span>
                </div>
                <div class="detail-section">
                  <div class="detail-section__header">${icons.scrollText} 发现摘要</div>
                  <div class="detail-section__content">${active.reportSummary}</div>
                </div>
                <div class="detail-section">
                  <div class="detail-section__header">${icons.messageSquare} AI 解释</div>
                  <div class="detail-section__content highlight">
                    ${active.reportMarkdown || "当前报告暂无完整正文，可结合最近告警和资产健康度继续分析。"}
                  </div>
                </div>
              `}
        </div>
      </div>
    </div>
  `;
}

function renderSkeletonView(props: WorkbenchProps, view: WorkbenchView) {
  const meta: Record<Exclude<WorkbenchView, "events" | "inspection">, { title: string; input: string; output: string; next: string }> = {
    diagnosis: {
      title: "诊断中心",
      input: "告警组、巡检异常、组件指标、日志片段、拓扑关系",
      output: "根因候选、证据链、影响面、验证步骤",
      next: "下一步接入指标/日志证据与 Runbook 推荐。",
    },
    governance: {
      title: "治理中心",
      input: "重复告警、低健康分项、作业失败模式、配置漂移",
      output: "治理建议、优先级、预期收益、责任对象",
      next: "下一步沉淀治理任务和采纳效果。",
    },
    capacity: {
      title: "容量性能",
      input: "CPU/内存/磁盘/HDFS/YARN/数据库容量与趋势",
      output: "容量风险、扩缩容建议、性能瓶颈、成本影响",
      next: "下一步接入趋势预测与容量水位规则。",
    },
    change: {
      title: "变更护航",
      input: "变更窗口、目标集群、组件版本、历史故障和巡检基线",
      output: "变更前检查、风险项、回滚建议、审批点",
      next: "下一步接入变更单和审批链路。",
    },
  };
  const item = meta[view as Exclude<WorkbenchView, "events" | "inspection">];
  return html`
    <div class="ops-card">
      <div class="column-header">${item.title}</div>
      <div class="ops-summary-cards">
        <div class="ops-card stat-card">
          <div class="stat-label">输入上下文</div>
          <div class="muted">${item.input}</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">输出成果</div>
          <div class="muted">${item.output}</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">当前状态</div>
          <div class="stat-value info">骨架</div>
          <div class="muted">不会承诺未接通的自动化动作</div>
        </div>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.info} 后续落地</div>
        <div class="detail-section__content highlight">${item.next}</div>
      </div>
    </div>
  `;
}

function renderEventsView(props: WorkbenchProps, active: WorkbenchAlertGroup | undefined, originalTotal: number, criticalCount: number, warningCount: number) {
  return html`
    ${props.alertsError
      ? html`<div class="ops-panel" style="margin-bottom:12px;">${renderOpsError({ message: props.alertsError })}</div>`
      : nothing}

    ${renderOpsShellStatGrid([
      {
        label: "待处理告警组",
        value: props.alertGroups.length,
        hint: "合并后的核心故障组",
        tone: "warn",
        icon: "bell",
      },
      {
        label: "严重告警",
        value: criticalCount,
        hint: "需优先处理",
        tone: "danger",
        icon: "alertTriangle",
      },
      {
        label: "原始事件",
        value: originalTotal,
        hint: "跨域事件条数",
        tone: "info",
        icon: "zap",
      },
      {
        label: "警告级",
        value: warningCount,
        hint: "巡检/治理候选",
        tone: "ok",
        icon: "historyClock",
      },
    ])}

    <div class="ops-main-columns ops-shell-columns ${props.aiPanelOpen && active ? "ops-shell-columns--with-side" : ""}">
      <div class="ops-card list-column">
        <div class="column-header">${props.domainName} 告警列表</div>
        ${props.alertsLoading
          ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
          : props.alertGroups.length === 0
            ? html`<div class="empty-placeholder">暂无待处理告警。</div>`
            : html`
                <div class="alert-list">
                  ${props.alertGroups.map(
                    (g) => html`
                      <button
                        type="button"
                        class="alert-item ${g.id === active?.id ? "alert-item--active" : ""} alert-item--${g.severity}"
                        @click=${() => props.onSelectAlertGroup?.(g.id)}
                      >
                        <div class="alert-item__meta">
                          <span class="alert-badge alert-badge--${g.severity}">${severityLabel(g.severity)}</span>
                          <span class="alert-time">${g.timestamp}</span>
                        </div>
                        <div class="alert-item__title">${g.title}</div>
                        <div class="alert-item__noise">
                          <span>原始告警: <strong>${g.originalCount}</strong> 次</span>
                          <span class="divider">|</span>
                          <span class="rate">降噪后: <strong>${g.reducedTo}</strong> 组</span>
                        </div>
                      </button>
                    `,
                  )}
                </div>
              `}
      </div>

      <div class="ops-card detail-column">
        <div class="column-header">告警详情与 AI 操作</div>
        ${!active
          ? html`<div class="empty-placeholder">请选择一个告警组。</div>`
          : html`
              <div class="alert-detail">
                <div class="detail-section">
                  <div class="detail-section__title">${active.title}</div>
                  <div class="detail-meta">
                    <span class="detail-time">生成时间: ${active.timestamp}</span>
                    <span class="detail-count">关联原始事件: ${active.originalCount} 次</span>
                  </div>
                </div>
                <div class="detail-section">
                  <div class="detail-section__header">${icons.zap} 根因候选</div>
                  <div class="detail-section__content highlight">${active.rootCause || "暂无根因候选。"}</div>
                </div>
                ${active.impact
                  ? html`
                      <div class="detail-section">
                        <div class="detail-section__header">${icons.overviewGrid} 影响范围</div>
                        <div class="detail-section__content">${active.impact}</div>
                      </div>
                    `
                  : nothing}
                <div class="detail-section" style="display:flex; gap:8px; flex-wrap:wrap;">
                  <button class="ops-btn ops-btn--primary" type="button" @click=${() => props.onOpenAiPanel?.("root-cause")}>
                    ${icons.messageSquare} 分析根因
                  </button>
                  <button class="ops-btn" type="button" @click=${() => props.onOpenAiPanel?.("similar")}>聚合相似</button>
                  <button class="ops-btn" type="button" @click=${() => props.onOpenAiPanel?.("action")}>处置建议</button>
                  <button class="ops-btn" type="button" @click=${() => props.onSendToCopilot?.(active.id, "root-cause")}>
                    ${icons.messageSquare} 发送到 AI 运维助手
                  </button>
                </div>
              </div>
            `}
      </div>

      ${renderAiPanel(props, active)}
    </div>
  `;
}

export function renderWorkbench(props: WorkbenchProps) {
  const active =
    props.alertGroups.find((g) => g.id === props.selectedAlertGroupId) || props.alertGroups[0];
  const originalTotal = props.alertGroups.reduce((acc, g) => acc + (g.originalCount || 0), 0);
  const criticalCount = props.alertGroups.filter((g) => g.severity === "critical").length;
  const warningCount = props.alertGroups.filter((g) => g.severity === "warning").length;
  const activeView = props.activeView ?? "events";

  const meta = WORKBENCH_VIEW_META[activeView];

  return html`
    <main class="ops-dashboard ops-shell">
      ${renderOpsShellHeader({
        kicker: `运维工作台 · ${props.domainName}`,
        title: meta.title,
        description: meta.description,
        toolbar:
          activeView === "events"
            ? html`
                <button
                  class="ops-btn ops-btn--primary"
                  type="button"
                  ?disabled=${props.alertsLoading}
                  @click=${() => props.onRefreshAlerts?.()}
                >
                  ${icons.refreshCw} 刷新告警
                </button>
              `
            : activeView === "inspection"
              ? html`
                  <button
                    class="ops-btn ops-btn--primary ${props.isInspecting ? "btn--loading" : ""}"
                    type="button"
                    ?disabled=${props.isInspecting || props.canInspect === false}
                    title=${props.canInspect === false ? "当前账号无 ops:inspect 权限" : ""}
                    @click=${() => props.onRunInspection?.()}
                  >
                    ${props.isInspecting ? icons.loader : icons.zap}
                    ${props.isInspecting ? "巡检中..." : "一键巡检"}
                  </button>
                `
              : nothing,
      })}
      ${props.domainFilter ?? nothing}
      ${renderOpsViewNav(WORKBENCH_VIEWS, activeView, (view) => props.onViewChange?.(view))}
      ${activeView === "events"
        ? renderEventsView(props, active, originalTotal, criticalCount, warningCount)
        : activeView === "inspection"
          ? renderInspectionView(props)
          : renderSkeletonView(props, activeView)}
    </main>
  `;
}
