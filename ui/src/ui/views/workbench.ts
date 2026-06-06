import { html, nothing } from "lit";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { icons } from "../icons.ts";
import { toSanitizedMarkdownHtml } from "../markdown.ts";
import { renderOpsError } from "../components/ops-status.ts";
import {
  renderOpsShellHeader,
  renderOpsShellStatGrid,
  type OpsViewNavItem,
} from "../components/ops-shell.ts";
import { renderOpsContextSidebar, type SidebarItem } from "../components/ops-context-sidebar.ts";
import {
  canAccessOpsDomain,
  normalizeOpsDomain,
  opsDomainLabel,
  type OpsDomainKey,
} from "../components/domain-filter.ts";
import type { OpsClusterRecord } from "../controllers/ops-clusters.ts";
import {
  findWorkbenchScenario,
  filterWorkbenchScenarios,
  scenarioCatalogStats,
  scenariosForWorkbench,
  WORKBENCH_TIME_RANGES,
  type OpsScenario,
  type OpsScenarioMaturityFilter,
  type WorkbenchTimeRange,
  OPS_SCENARIOS,
} from "../ops/scenario-registry.ts";
import { renderScenarioComponent } from "../ops/scenario-components.ts";
import { buildScenarioResult, type OpsScenarioResult } from "../ops/scenario-results.ts";
import {
  formatWorkbenchObjectScope,
  normalizeWorkbenchObjectScope,
  normalizeWorkbenchTimeRange,
  objectOptionsForScenario,
  type WorkbenchObjectOption,
} from "../ops/workbench-context.ts";

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
  selectedDomain?: string;
  user?: any;
  host?: any;
  onDomainChange?: (domain: string) => void;
  domainSummary?: {
    alertsCount?: number;
    clustersCount?: number;
    score?: number | null;
  };
  assistantName?: string;
  assistantPersona?: string;
  activeView?: WorkbenchView;
  selectedScenarioId?: string | null;
  scenarioSearch?: string | null;
  scenarioMaturityFilter?: string | null;
  selectedObjectScope?: string | null;
  selectedTimeRange?: string | null;
  domainClusters?: OpsClusterRecord[];
  flinkJobs?: any[];
  sparkJobs?: any[];
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
  aiStatus?: "idle" | "loading" | "streaming" | "done" | "error";
  aiStream?: string | null;
  aiResult?: string | null;
  aiError?: string | null;
  onRetryAi?: () => void;
  onViewChange?: (view: WorkbenchView) => void;
  onSelectScenario?: (id: string | null) => void;
  onScenarioSearchChange?: (query: string) => void;
  onScenarioMaturityFilterChange?: (maturity: OpsScenarioMaturityFilter) => void;
  onObjectScopeChange?: (scope: string) => void;
  onTimeRangeChange?: (range: WorkbenchTimeRange) => void;
  onOpenScenarioAi?: (scenario: OpsScenario, mode: "root-cause" | "similar" | "action") => void;
  onRecordScenarioSuggestion?: (scenario: OpsScenario) => void;
  onOpenScenarioTasks?: (scenario: OpsScenario) => void;
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

function renderRichText(text: string) {
  const blocks = (text || "")
    .split(/\n{2,}/)
    .map((b) => b.trim())
    .filter(Boolean);
  if (blocks.length === 0) {
    return nothing;
  }
  return html`${blocks.map(
    (block) => html`<p style="margin:0 0 8px; white-space:pre-wrap;">${block}</p>`,
  )}`;
}

function renderAiConclusion(props: WorkbenchProps, active: WorkbenchAlertGroup, mode: "root-cause" | "similar" | "action") {
  const status = props.aiStatus ?? "idle";
  if (status === "loading") {
    return html`<div class="detail-section__content highlight">${icons.loader} 正在调用数字员工分析当前告警上下文…</div>`;
  }
  if (status === "streaming") {
    return html`
      <div class="detail-section__content highlight">
        ${props.aiStream ? renderRichText(props.aiStream) : html`<span class="muted">${icons.loader} 分析中…</span>`}
        <div class="muted" style="margin-top:6px;">${icons.loader} 正在生成…</div>
      </div>
    `;
  }
  if (status === "done") {
    return html`<div class="detail-section__content highlight">
      ${props.aiResult ? renderRichText(props.aiResult) : html`<span class="muted">本次分析未返回内容。</span>`}
    </div>`;
  }
  if (status === "error") {
    return html`
      <div class="detail-section__content">
        <p style="color: var(--danger, #d33);">${props.aiError ?? "AI 分析失败。"}</p>
        ${props.onRetryAi
          ? html`<button class="ops-btn" type="button" @click=${props.onRetryAi}>${icons.refreshCw} 重试</button>`
          : nothing}
      </div>
    `;
  }
  const fallback =
    mode === "similar"
      ? `本组已聚合 ${active.originalCount} 条原始告警，建议优先按共同根因处理，避免逐条关闭。`
      : mode === "action"
        ? "建议先确认影响范围，再按 Runbook 执行只读检查；涉及变更或脚本执行时需人工审批。"
        : active.rootCause || "当前告警缺少明确根因，点击对应 AI 操作发起实时分析。";
  return html`<div class="detail-section__content highlight">${fallback}</div>`;
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
  const status = props.aiStatus ?? "idle";
  const busy = status === "loading" || status === "streaming";

  return html`
    <aside class="ops-card ops-shell-side detail-column">
      <div class="column-header" style="display:flex; align-items:center; justify-content:space-between; gap:12px;">
        <span>${icons.messageSquare} ${title}</span>
        <button class="ops-btn ops-btn--ghost" type="button" @click=${props.onCloseAiPanel}>关闭</button>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.users} 数字员工模板</div>
        <div class="detail-section__content">
          <strong>${props.assistantName ?? "BCH 值班数字员工"}</strong>
          <p class="muted">${props.assistantPersona ?? "专家人设：BCH 告警降噪、根因候选、影响面判断、处置建议。"}</p>
        </div>
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.zap} 结论</div>
        ${renderAiConclusion(props, active, mode)}
      </div>
      <div class="detail-section">
        <div class="detail-section__header">${icons.scrollText} 告警证据</div>
        <div class="detail-section__content">
          ${lines.length
            ? html`<ul>${lines.map((line) => html`<li>${line}</li>`)}</ul>`
            : html`<p class="muted">暂无可用分析证据。</p>`}
        </div>
      </div>
      <div class="detail-section">
        <div style="display:flex; gap:8px; flex-wrap:wrap;">
          <button
            class="ops-btn ops-btn--primary"
            type="button"
            ?disabled=${busy}
            title=${busy ? "请等待 AI 分析完成后再确认" : ""}
            @click=${() => props.onAcceptSuggestion?.(active.id)}
          >
            确认建议并记录
          </button>
          <button
            class="ops-btn"
            type="button"
            ?disabled=${busy}
            @click=${() => props.onRejectSuggestion?.(active.id)}
          >
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


function renderInspectionMarkdown(content: string) {
  const trimmed = content.trim();
  if (!trimmed) {
    return html`<p class="muted">当前报告暂无完整正文，可结合最近告警和资产健康度继续分析。</p>`;
  }
  return html`<div class="ops-markdown ops-inspection-report">${unsafeHTML(toSanitizedMarkdownHtml(trimmed))}</div>`;
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

      <div class="ops-inspection-layout-hint muted">
        左侧选择巡检记录，右侧查看完整深度报告。
      </div>

      <div class="ops-main-columns ops-shell-columns ops-inspection-columns">
        <div class="ops-card list-column">
          <div class="column-header">巡检报告（左侧）</div>
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
        <div class="ops-card detail-column ops-inspection-detail">
          <div class="column-header">报告详情（右侧）</div>
          ${!active
            ? html`<div class="empty-placeholder">请选择一份巡检报告。</div>`
            : html`
                <div class="ops-inspection-detail__body">
                  <div class="report-header">
                    <h3>${props.domainName} 深度巡检报告</h3>
                    <span class="score-badge score-badge--${scoreClass(active.score)}">
                      ${active.score == null || active.score < 0 ? "健康分未知" : `健康分 ${active.score}/100`}
                    </span>
                  </div>
                  <div class="detail-section">
                    <div class="detail-section__header">发现摘要</div>
                    <div class="detail-section__content">${active.reportSummary}</div>
                  </div>
                  <div class="detail-section ops-inspection-detail__report">
                    <div class="detail-section__header">完整报告</div>
                    <div class="detail-section__content highlight">
                      ${renderInspectionMarkdown(active.reportMarkdown)}
                    </div>
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

function maturityLabel(maturity: OpsScenario["maturity"]): string {
  switch (maturity) {
    case "automated":
      return "自动化闭环";
    case "connected":
      return "已接入";
    case "beta":
      return "Beta";
    default:
      return "规划中";
  }
}

function normalizeScenarioMaturityFilter(value: string | null | undefined): OpsScenarioMaturityFilter {
  if (value === "planned" || value === "beta" || value === "connected" || value === "automated") {
    return value;
  }
  return "all";
}

function automationLabel(level: OpsScenario["automationLevel"]): string {
  switch (level) {
    case "closed-loop":
      return "闭环执行";
    case "approval":
      return "审批后执行";
    case "recommendation":
      return "建议采纳";
    default:
      return "手动分析";
  }
}

function triggerLabel(trigger: OpsScenario["triggers"][number]): string {
  switch (trigger) {
    case "alert":
      return "告警触发";
    case "schedule":
      return "定时巡检";
    case "change":
      return "变更触发";
    default:
      return "手动触发";
  }
}

function renderTagList(label: string, items: string[]) {
  return html`
    <div style="margin-top:8px;">
      <div class="stat-label" style="margin-bottom:4px;">${label}</div>
      <div class="detail-meta" style="flex-wrap:wrap;">
        ${items.map((item) => html`<span>${item}</span>`)}
      </div>
    </div>
  `;
}

function renderAiStatusBlock(props: WorkbenchProps) {
  const status = props.aiStatus ?? "idle";
  if (status === "loading") {
    return html`<div class="detail-section__content highlight">${icons.loader} 正在调用专项数字员工分析...</div>`;
  }
  if (status === "streaming") {
    return html`
      <div class="detail-section__content highlight">
        ${props.aiStream ? renderRichText(props.aiStream) : html`<span class="muted">${icons.loader} 分析中...</span>`}
      </div>
    `;
  }
  if (status === "done") {
    return html`
      <div class="detail-section__content highlight">
        ${props.aiResult ? renderRichText(props.aiResult) : html`<span class="muted">本次专项分析未返回内容。</span>`}
      </div>
    `;
  }
  if (status === "error") {
    return html`<div class="detail-section__content" style="color: var(--danger, #d33);">${props.aiError ?? "AI 分析失败。"}</div>`;
  }
  return html`<div class="detail-section__content highlight">选择一个 AI 操作后，系统会基于当前技术域、对象范围、时间范围和专项证据生成建议。</div>`;
}

function renderScenarioClosurePanel(
  props: WorkbenchProps,
  scenario: OpsScenario,
  result: OpsScenarioResult,
  selectedObjectScope: string,
  selectedTimeRange: WorkbenchTimeRange,
) {
  const busy = props.aiStatus === "loading" || props.aiStatus === "streaming";
  return html`
    <section class="ops-card" style="margin-bottom:14px;">
      <div class="column-header">专项闭环</div>
      <div style="padding:16px; display:grid; grid-template-columns: minmax(0, 1fr) minmax(280px, 0.9fr); gap:16px;">
        <div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.activity} 健康信号</div>
            <div class="detail-section__content highlight">${result.healthSignal}</div>
          </div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.alertTriangle} 风险证据</div>
            <div class="detail-section__content">
              <ul style="margin:0; padding-left:18px;">
                ${result.riskEvidence.map((item) => html`<li>${item}</li>`)}
              </ul>
            </div>
          </div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.scrollText} 输出结构</div>
            <div class="detail-section__content">
              <ul style="margin:0; padding-left:18px;">
                ${result.outputs.map((item) => html`<li>${item}</li>`)}
              </ul>
            </div>
          </div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.zap} 建议动作</div>
            <div class="detail-section__content">
              <ul style="margin:0; padding-left:18px;">
                ${result.recommendedActions.map((item) => html`<li>${item}</li>`)}
              </ul>
            </div>
          </div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.usageBars} 预期收益</div>
            <div class="detail-section__content highlight">${result.expectedBenefit}</div>
          </div>
          <div class="detail-section">
            <div class="detail-section__header">${icons.historyClock} Runbook</div>
            <div class="detail-meta" style="flex-wrap:wrap;">
              ${result.runbooks.map((item) => html`<span>${item}</span>`)}
            </div>
          </div>
        </div>
        <aside class="detail-section" style="margin:0;">
          <div class="detail-section__header">${icons.messageSquare} AI 操作</div>
          ${renderAiStatusBlock(props)}
          <div style="display:flex; gap:8px; flex-wrap:wrap; margin-top:12px;">
            <button
              class="ops-btn ops-btn--primary"
              type="button"
              ?disabled=${busy}
              @click=${() => props.onOpenScenarioAi?.(scenario, "root-cause")}
            >
              分析风险
            </button>
            <button class="ops-btn" type="button" ?disabled=${busy} @click=${() => props.onOpenScenarioAi?.(scenario, "action")}>
              生成建议
            </button>
            <button class="ops-btn" type="button" ?disabled=${busy} @click=${() => props.onRecordScenarioSuggestion?.(scenario)}>
              记录闭环
            </button>
            <button class="ops-btn ops-btn--ghost" type="button" @click=${() => props.onOpenScenarioTasks?.(scenario)}>
              执行记录
            </button>
          </div>
          <div class="muted" style="margin-top:10px;">
            对象 ${selectedObjectScope || "all"} · 时间 ${selectedTimeRange}
          </div>
        </aside>
      </div>
    </section>
  `;
}

function renderWorkbenchContextBar(
  view: WorkbenchView,
  selectedDomain: OpsDomainKey,
  objectOptions: WorkbenchObjectOption[],
  selectedObjectScope: string,
  selectedTimeRange: WorkbenchTimeRange,
  props: WorkbenchProps,
  scenario?: OpsScenario,
) {
  const objectScope = formatWorkbenchObjectScope(selectedObjectScope, objectOptions);
  const scopeLabel = scenario ? scenario.title : WORKBENCH_VIEW_META[view].title;
  return html`
    <section class="ops-card" style="margin-bottom: 14px;">
      <div style="display:grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 12px; padding: 14px;">
        <div>
          <div class="stat-label">技术域</div>
          <div class="detail-section__title" style="font-size:14px;">${opsDomainLabel(selectedDomain)}</div>
        </div>
        <div>
          <div class="stat-label">工作中心</div>
          <div class="detail-section__title" style="font-size:14px;">${WORKBENCH_VIEW_META[view].title}</div>
        </div>
        <div>
          <div class="stat-label">对象范围</div>
          <label class="select" style="display:block; margin-top:4px;">
            <select
              .value=${selectedObjectScope}
              @change=${(e: Event) => props.onObjectScopeChange?.((e.target as HTMLSelectElement).value)}
            >
              ${objectOptions.map(
                (option) => html`<option value=${option.id} ?selected=${option.id === selectedObjectScope}>
                  ${option.label}${option.subtitle ? ` · ${option.subtitle}` : ""}
                </option>`,
              )}
            </select>
          </label>
        </div>
        <div>
          <div class="stat-label">时间范围</div>
          <label class="select" style="display:block; margin-top:4px;">
            <select
              .value=${selectedTimeRange}
              @change=${(e: Event) =>
                props.onTimeRangeChange?.(normalizeWorkbenchTimeRange((e.target as HTMLSelectElement).value))}
            >
              ${WORKBENCH_TIME_RANGES.map(
                (range) => html`<option value=${range.id} ?selected=${range.id === selectedTimeRange}>${range.label}</option>`,
              )}
            </select>
          </label>
        </div>
      </div>
      <div class="detail-section__content highlight" style="margin: 0 14px 14px;">
        当前上下文：${opsDomainLabel(selectedDomain)} / ${WORKBENCH_VIEW_META[view].title} / ${scopeLabel} /
        ${objectScope.title} / ${WORKBENCH_TIME_RANGES.find((range) => range.id === selectedTimeRange)?.label ?? selectedTimeRange}
      </div>
    </section>
  `;
}

function renderScenarioDirectory(props: WorkbenchProps, view: WorkbenchView, selectedDomain: OpsDomainKey) {
  let scenarios = scenariosForWorkbench(selectedDomain, view);
  if (selectedDomain === "all") {
    scenarios = scenarios.filter((scenario) => canAccessOpsDomain(props.user, scenario.domain));
  }
  if (scenarios.length === 0) {
    return renderSkeletonView(props, view);
  }

  let stats;
  if (selectedDomain === "all") {
    const accessible = OPS_SCENARIOS.filter((scenario) => canAccessOpsDomain(props.user, scenario.domain));
    stats = {
      total: accessible.length,
      centers: {} as Record<string, number>,
      maturity: { planned: 0, beta: 0, connected: 0, automated: 0 },
    };
    for (const scenario of accessible) {
      stats.centers[scenario.center] = (stats.centers[scenario.center] ?? 0) + 1;
      stats.maturity[scenario.maturity] += 1;
    }
  } else {
    stats = scenarioCatalogStats(selectedDomain);
  }

  const connectedCount = stats.maturity.beta + stats.maturity.connected + stats.maturity.automated;
  const activeCenterCount = stats.centers[view] ?? scenarios.length;
  const scenarioSearch = props.scenarioSearch ?? "";
  const maturityFilter = normalizeScenarioMaturityFilter(props.scenarioMaturityFilter);
  const filteredScenarios = filterWorkbenchScenarios(scenarios, scenarioSearch, maturityFilter);

  return html`
    <div class="ops-summary-cards" style="margin-bottom: 14px;">
      <div class="ops-card stat-card">
        <div class="stat-icon-slot">${icons.folder}</div>
        <div class="stat-body">
          <div class="stat-label">场景总数</div>
          <div class="stat-value">${stats.total}</div>
          <div class="muted">${selectedDomain === "all" ? "跨域已注册场景" : `${opsDomainLabel(selectedDomain)} 已注册场景`}</div>
        </div>
      </div>
      <div class="ops-card stat-card">
        <div class="stat-icon-slot">${icons.layout}</div>
        <div class="stat-body">
          <div class="stat-label">当前中心</div>
          <div class="stat-value">${activeCenterCount}</div>
          <div class="muted">${WORKBENCH_VIEW_META[view].title} 可用场景</div>
        </div>
      </div>
      <div class="ops-card stat-card">
        <div class="stat-icon-slot">${icons.checkCircle}</div>
        <div class="stat-body">
          <div class="stat-label">试点/已接入</div>
          <div class="stat-value ok">${connectedCount}</div>
          <div class="muted">Beta、已接入或自动化闭环</div>
        </div>
      </div>
      <div class="ops-card stat-card">
        <div class="stat-icon-slot">${icons.info}</div>
        <div class="stat-body">
          <div class="stat-label">规划中</div>
          <div class="stat-value info">${stats.maturity.planned}</div>
          <div class="muted">已有对象、输入、输出和 Runbook 骨架</div>
        </div>
      </div>
    </div>
    <section class="ops-card" style="margin-bottom: 14px;">
      <div class="column-header" style="display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap;">
        <span>场景目录</span>
        <span class="muted">当前显示 ${filteredScenarios.length} / ${scenarios.length}</span>
      </div>
      <div style="display:grid; grid-template-columns: minmax(220px, 1fr) minmax(180px, 240px); gap:12px; padding:14px;">
        <label class="input" style="display:block;">
          <input
            type="search"
            autocomplete="off"
            placeholder="搜索场景、对象、输入证据或输出成果"
            .value=${scenarioSearch}
            @input=${(e: Event) => props.onScenarioSearchChange?.((e.target as HTMLInputElement).value)}
          />
        </label>
        <label class="select" style="display:block;">
          <select
            .value=${maturityFilter}
            @change=${(e: Event) =>
              props.onScenarioMaturityFilterChange?.(
                normalizeScenarioMaturityFilter((e.target as HTMLSelectElement).value),
              )}
          >
            <option value="all" ?selected=${maturityFilter === "all"}>全部成熟度</option>
            <option value="planned" ?selected=${maturityFilter === "planned"}>规划中</option>
            <option value="beta" ?selected=${maturityFilter === "beta"}>Beta</option>
            <option value="connected" ?selected=${maturityFilter === "connected"}>已接入</option>
            <option value="automated" ?selected=${maturityFilter === "automated"}>自动化闭环</option>
          </select>
        </label>
      </div>
      ${filteredScenarios.length === 0
        ? html`<div class="empty-placeholder" style="margin: 0 14px 14px;">没有匹配的场景，可调整关键字或成熟度过滤。</div>`
        : html`
            <div class="ops-summary-cards">
              ${filteredScenarios.map(
                (scenario) => html`
                  <article class="ops-card stat-card" style="align-items: flex-start;">
                    <div class="stat-icon-slot">${icons[scenario.icon] ?? icons.folder}</div>
                    <div class="stat-body">
                      <div style="display:flex; align-items:center; justify-content:space-between; gap:10px; margin-bottom:4px;">
                        <h3 style="margin:0;">${scenario.title}</h3>
                        <span class="score-badge score-badge--${scenario.maturity === "planned" ? "unknown" : "ok"}">
                          ${maturityLabel(scenario.maturity)}
                        </span>
                      </div>
                      <p class="muted" style="margin:0 0 10px;">${scenario.summary}</p>
                      <div class="detail-meta" style="margin-bottom:10px; flex-wrap:wrap;">
                        <span>${opsDomainLabel(scenario.domain, true)}</span>
                        <span>${automationLabel(scenario.automationLevel)}</span>
                        ${scenario.triggers.map((trigger) => html`<span>${triggerLabel(trigger)}</span>`)}
                        <span>${scenario.primaryMetric ?? scenario.objectTypes.join(" / ")}</span>
                      </div>
                      ${renderTagList("输入证据", scenario.inputs)}
                      ${renderTagList("输出成果", scenario.outputs)}
                      <button
                        class="ops-btn ops-btn--primary"
                        type="button"
                        style="margin-top: 12px;"
                        @click=${() => {
                          if (selectedDomain === "all") {
                            props.onDomainChange?.(scenario.domain);
                          }
                          props.onSelectScenario?.(scenario.id);
                        }}
                      >
                        进入专项
                      </button>
                    </div>
                  </article>
                `,
              )}
            </div>
          `}
    </section>
  `;
}

function renderScenarioDetail(
  props: WorkbenchProps,
  view: WorkbenchView,
  selectedDomain: OpsDomainKey,
  selectedObjectScope: string,
  selectedTimeRange: WorkbenchTimeRange,
) {
  const scenario = findWorkbenchScenario(props.selectedScenarioId);
  if (!scenario || scenario.center !== view) {
    return renderScenarioDirectory(props, view, selectedDomain);
  }
  if (selectedDomain === "all") {
    return renderScenarioDirectory(props, view, selectedDomain);
  }
  if (scenario.domain !== selectedDomain) {
    return renderScenarioDirectory(props, view, selectedDomain);
  }

  const back = html`
    <div style="display:flex; justify-content:space-between; align-items:center; gap:12px; margin-bottom:12px;">
      <div class="detail-meta" style="flex-wrap:wrap;">
        <span>${opsDomainLabel(scenario.domain)}</span>
        <span>${scenario.objectTypes.join(" / ")}</span>
        <span>${automationLabel(scenario.automationLevel)}</span>
      </div>
      <button class="ops-btn ops-btn--ghost" type="button" @click=${() => props.onSelectScenario?.(null)}>
        返回场景目录
      </button>
    </div>
  `;

  const result = buildScenarioResult(scenario, selectedObjectScope, selectedTimeRange);
  return html`
    ${back}
    ${renderScenarioClosurePanel(props, scenario, result, selectedObjectScope, selectedTimeRange)}
    ${renderScenarioComponent(scenario, {
      host: props.host,
      objectScope: selectedObjectScope,
      timeRange: selectedTimeRange,
    })}
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
  const selectedDomain = normalizeOpsDomain(props.selectedDomain || "all");
  const selectedScenarioRaw = findWorkbenchScenario(props.selectedScenarioId);
  const selectedScenario =
    selectedScenarioRaw &&
    selectedScenarioRaw.center === activeView &&
    selectedDomain !== "all" &&
    selectedScenarioRaw.domain === selectedDomain
      ? selectedScenarioRaw
      : undefined;
  const objectOptions = objectOptionsForScenario(
    selectedScenario,
    props.domainClusters ?? [],
    props.flinkJobs ?? [],
    props.sparkJobs ?? [],
  );
  const selectedObjectScope = normalizeWorkbenchObjectScope(props.selectedObjectScope, objectOptions);
  const selectedTimeRange = normalizeWorkbenchTimeRange(props.selectedTimeRange);

  const meta = WORKBENCH_VIEW_META[activeView];

  const sidebarItems: SidebarItem<WorkbenchView>[] = WORKBENCH_VIEWS.map((v) => {
    if (v.id === "events") {
      return { ...v, badge: props.alertGroups.length };
    }
    return v;
  });

  return html`
    <div class="ops-workbench-layout" style="display: flex; height: 100%; width: 100%;">
      ${renderOpsContextSidebar({
        selectedDomain: props.selectedDomain || "all",
        user: props.user,
        items: sidebarItems,
        activeItemId: activeView,
        onItemChange: (id) => props.onViewChange?.(id as WorkbenchView),
        onDomainChange: (domain) => props.onDomainChange?.(domain),
        domainSummary: props.domainSummary,
        includeAllDomain: true,
      })}
      <div style="flex: 1; min-width: 0; overflow-y: auto;">
        <main class="ops-dashboard ops-shell" style="height: 100%; box-sizing: border-box; display: flex; flex-direction: column;">
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
                : nothing,
          })}
          ${renderWorkbenchContextBar(
            activeView,
            selectedDomain,
            objectOptions,
            selectedObjectScope,
            selectedTimeRange,
            props,
            selectedScenario,
          )}
          ${activeView === "events"
            ? renderEventsView(props, active, originalTotal, criticalCount, warningCount)
            : activeView === "inspection"
              ? renderInspectionView(props)
              : renderScenarioDetail(props, activeView, selectedDomain, selectedObjectScope, selectedTimeRange)}
        </main>
      </div>
    </div>
  `;
}
