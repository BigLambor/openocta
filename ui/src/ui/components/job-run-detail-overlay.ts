import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import {
  formatJobRunTimestamp,
  jobRunStatusLabel,
  type OpsJobRunDetail,
} from "../controllers/ops-job-runs.ts";

export type JobRunDetailOverlayProps = {
  open: boolean;
  loading: boolean;
  error: string | null;
  detail: OpsJobRunDetail | null;
  onClose: () => void;
};

function renderJsonBlock(label: string, value: Record<string, unknown> | undefined) {
  if (!value || Object.keys(value).length === 0) {
    return nothing;
  }
  return html`
    <div class="detail-section">
      <div class="detail-section__header">${label}</div>
      <pre class="job-run-json">${JSON.stringify(value, null, 2)}</pre>
    </div>
  `;
}

function statusClass(status: string): string {
  switch (status) {
    case "succeeded":
    case "ok":
      return "cron-job-status-ok";
    case "failed":
    case "error":
      return "cron-job-status-error";
    case "running":
      return "cron-job-status-skipped";
    case "cancelled":
      return "cron-job-status-na";
    default:
      return "cron-job-status-na";
  }
}

export function renderJobRunDetailOverlay(props: JobRunDetailOverlayProps) {
  if (!props.open) {
    return nothing;
  }
  const run = props.detail?.run;
  const steps = props.detail?.steps ?? [];
  const toolInvocations = props.detail?.toolInvocations ?? [];

  return html`
    <div
      class="channel-panel-overlay"
      role="dialog"
      aria-modal="true"
      aria-labelledby="job-run-detail-title"
      @click=${(e: Event) => {
        const el = e.target as HTMLElement;
        if (el.classList.contains("channel-panel-overlay")) {
          props.onClose();
        }
      }}
    >
      <div class="card channel-panel job-run-detail-panel" @click=${(e: Event) => e.stopPropagation()}>
        <div class="row" style="justify-content: space-between; align-items: flex-start; gap: 12px; margin-bottom: 12px;">
          <div>
            <div class="card-title" id="job-run-detail-title">执行链路</div>
            <div class="card-sub muted">JobRun · steps · tools · output</div>
          </div>
          <button type="button" class="btn" @click=${props.onClose}>关闭</button>
        </div>

        ${props.loading
          ? html`<div class="muted">${icons.loader} 加载中...</div>`
          : props.error
            ? html`<div class="callout danger">${props.error}</div>`
            : !run
              ? html`<div class="muted">未找到执行记录。</div>`
              : html`
                  <div class="job-run-detail-body">
                    <div class="detail-section">
                      <div class="detail-section__header">运行概览</div>
                      <div class="job-run-meta-grid">
                        <div><span class="muted">Run ID</span><div class="mono">${run.id}</div></div>
                        <div><span class="muted">Job</span><div>${run.jobId || "—"}</div></div>
                        <div><span class="muted">触发</span><div>${run.triggerType}${run.triggerRef ? ` · ${run.triggerRef}` : ""}</div></div>
                        <div>
                          <span class="muted">状态</span>
                          <div><span class=${`cron-job-status-pill ${statusClass(run.status)}`}>${jobRunStatusLabel(run.status)}</span></div>
                        </div>
                        <div><span class="muted">开始</span><div>${formatJobRunTimestamp(run.startedAt)}</div></div>
                        <div><span class="muted">结束</span><div>${formatJobRunTimestamp(run.finishedAt)}</div></div>
                      </div>
                      ${run.error
                        ? html`<div class="callout danger" style="margin-top: 12px;">${run.error}</div>`
                        : nothing}
                    </div>

                    <div class="detail-section">
                      <div class="detail-section__header">执行步骤 (${steps.length})</div>
                      ${steps.length === 0
                        ? html`<div class="muted">暂无步骤记录。</div>`
                        : html`
                            <div class="list job-run-steps-list">
                              ${steps.map(
                                (step) => html`
                                  <div class="list-item job-run-step">
                                    <div class="list-main">
                                      <div class="list-title">
                                        #${step.stepOrder} ${step.name || step.kind}
                                        <span class=${`cron-job-status-pill ${statusClass(step.status)}`} style="margin-left: 8px;">
                                          ${jobRunStatusLabel(step.status)}
                                        </span>
                                      </div>
                                      <div class="list-sub muted">${step.kind}</div>
                                      ${step.inputSummary
                                        ? html`<div class="list-sub">输入：${step.inputSummary}</div>`
                                        : nothing}
                                      ${step.outputSummary
                                        ? html`<div class="list-sub">输出：${step.outputSummary}</div>`
                                        : nothing}
                                      ${step.error ? html`<div class="list-sub danger">${step.error}</div>` : nothing}
                                    </div>
                                    <div class="list-meta muted">${formatJobRunTimestamp(step.finishedAt || step.startedAt)}</div>
                                  </div>
                                `,
                              )}
                            </div>
                          `}
                    </div>

                    ${renderJsonBlock("输入", run.input)}
                    ${renderJsonBlock("输出", run.output)}

                    ${toolInvocations.length > 0
                      ? html`
                          <div class="detail-section">
                            <div class="detail-section__header">Tool Invocations (${toolInvocations.length})</div>
                            <div class="list job-run-steps-list">
                              ${toolInvocations.map(
                                (inv) => html`
                                  <div class="list-item job-run-step">
                                    <div class="list-main">
                                      <div class="list-title">
                                        ${inv.toolName}
                                        <span class="muted" style="margin-left: 8px;">${inv.provider || "agent"}</span>
                                      </div>
                                      ${inv.inputSummary ? html`<div class="list-sub">输入：${inv.inputSummary}</div>` : nothing}
                                      ${inv.outputSummary ? html`<div class="list-sub">输出：${inv.outputSummary}</div>` : nothing}
                                      ${inv.error ? html`<div class="list-sub danger">${inv.error}</div>` : nothing}
                                    </div>
                                    <div class="list-meta muted">
                                      ${inv.durationMs ? `${inv.durationMs}ms` : ""}
                                    </div>
                                  </div>
                                `,
                              )}
                            </div>
                          </div>
                        `
                      : nothing}
                  </div>
                `}
      </div>
    </div>
  `;
}
