import { html, nothing, type TemplateResult } from "lit";
import { icons } from "../icons.ts";
import { renderOpsError } from "../components/ops-status.ts";

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

export type WorkbenchProps = {
  domainName: string;
  domainFilter?: TemplateResult;
  alertsLoading?: boolean;
  alertsError?: string | null;
  alertGroups: WorkbenchAlertGroup[];
  selectedAlertGroupId?: string | null;
  aiPanelOpen?: boolean;
  aiPanelMode?: "root-cause" | "similar" | "action";
  onRefreshAlerts?: () => void;
  onSelectAlertGroup?: (id: string) => void;
  onOpenAiPanel?: (mode: "root-cause" | "similar" | "action") => void;
  onSendToCopilot?: (id: string, mode: "root-cause" | "similar" | "action") => void;
  onCloseAiPanel?: () => void;
  onAcceptSuggestion?: (id: string) => void;
  onRejectSuggestion?: (id: string) => void;
  onOpenTasks?: (id: string) => void;
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
    <aside class="ops-card" style="min-width: 320px; max-width: 380px;">
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

export function renderWorkbench(props: WorkbenchProps) {
  const active =
    props.alertGroups.find((g) => g.id === props.selectedAlertGroupId) || props.alertGroups[0];
  const originalTotal = props.alertGroups.reduce((acc, g) => acc + (g.originalCount || 0), 0);
  const criticalCount = props.alertGroups.filter((g) => g.severity === "critical").length;
  const warningCount = props.alertGroups.filter((g) => g.severity === "warning").length;

  return html`
    <div class="ops-domain-page">
      <div class="ops-domain-hero">
        <div>
          <div class="ops-domain-kicker">运维工作台 · ${props.domainName}</div>
          <h1>事件中心</h1>
          <p>从待处理告警进入根因分析、处置建议和执行记录，AI 能力嵌入当前工作流。</p>
        </div>
        <div style="display:flex; flex-direction:column; align-items:flex-end; gap:10px;">
          ${props.domainFilter ?? nothing}
          <button class="ops-btn ops-btn--primary" type="button" @click=${props.onRefreshAlerts}>
            ${icons.loader} 刷新告警
          </button>
        </div>
      </div>

      ${props.alertsError
        ? html`<div class="ops-panel" style="margin-bottom:12px;">${renderOpsError({ message: props.alertsError })}</div>`
        : nothing}

      <div class="ops-summary-cards">
        <div class="ops-card stat-card">
          <div class="stat-label">待处理告警组</div>
          <div class="stat-value warning">${props.alertGroups.length}</div>
          <div class="muted">合并后的核心故障组</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">严重告警</div>
          <div class="stat-value critical">${criticalCount}</div>
          <div class="muted">需优先处理</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">原始事件</div>
          <div class="stat-value info">${originalTotal}</div>
          <div class="muted">降噪前事件数</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">巡检/治理</div>
          <div class="stat-value ok">${warningCount}</div>
          <div class="muted">警告级任务候选</div>
        </div>
      </div>

      <div class="ops-main-columns" style="align-items: stretch;">
        <div class="ops-card list-column">
          <div class="column-header">BCH 告警列表</div>
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
    </div>
  `;
}
