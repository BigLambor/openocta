import { html, nothing } from "lit";
import type { ChannelUiMetaEntry, CronJob, CronRunLogEntry, CronStatus } from "../types.ts";
import type { CronFormState } from "../ui-types.ts";
import { formatAgo, formatMs } from "../format.ts";
import { icons } from "../icons.js";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { pathForTab } from "../navigation.ts";
import { cronJobSessionKeyFromRunSessionKey } from "../sessions/session-key-utils.js";
import {
  formatCronSchedule,
  formatNextRun,
} from "../presenter.ts";
import { t } from "../strings.js";
import { resolveJobRunIdForCronEntry } from "../controllers/ops-job-runs.ts";

export type CronProps = {
  basePath: string;
  loading: boolean;
  status: CronStatus | null;
  jobs: CronJob[];
  error: string | null;
  busy: boolean;
  form: CronFormState;
  addModalOpen: boolean;
  editModalOpen?: boolean;
  editJobId?: string | null;
  digitalEmployees?: Array<{ id: string; name?: string; enabled?: boolean }>;
  digitalEmployeesLoading?: boolean;
  channels: string[];
  channelLabels?: Record<string, string>;
  channelMeta?: ChannelUiMetaEntry[];
  runsJobId: string | null;
  runs: CronRunLogEntry[];
  jobRuns?: import("../controllers/ops-job-runs.ts").OpsJobRunRecord[];
  onOpenJobRun?: (runId: string) => void;
  onFormChange: (patch: Partial<CronFormState>) => void;
  onRefresh: () => void;
  onOpenAddModal: () => void;
  onCloseAddModal: () => void;
  onAdd: () => void;
  onOpenEditModal?: (job: CronJob) => void;
  onCloseEditModal?: () => void;
  onUpdate?: (jobId: string) => void;
  onToggle: (job: CronJob, enabled: boolean) => void;
  onRun: (job: CronJob) => void;
  onRemove: (job: CronJob) => void;
  confirmRemove?: boolean;
  onLoadRuns: (jobId: string) => void;
  onShowHistory?: (jobId: string) => void;
};

function normalizeEmployeeIdForCron(raw: string): string {
  let s = (raw ?? "").trim();
  if (!s) return "";
  if (s.toLowerCase().startsWith("local:")) {
    s = s.slice("local:".length);
  }
  s = s.replaceAll(":", "-");
  return s.trim().toLowerCase();
}

function showCronToTooltip(e: Event) {
  const target = e.currentTarget as HTMLElement;
  const hint = target.classList.contains("mcp-field-hint")
    ? target
    : (target.closest(".mcp-field-hint") as HTMLElement);
  if (!hint) return;
  const tooltip = hint.querySelector(".mcp-field-hint__tooltip") as HTMLElement;
  if (!tooltip) return;
  const rect = hint.getBoundingClientRect();
  tooltip.style.left = rect.left + "px";
  tooltip.style.top = rect.bottom + 6 + "px";
  tooltip.classList.add("is-visible");
}

function hideCronToTooltip(e: Event) {
  const target = e.currentTarget as HTMLElement;
  const hint = target.classList.contains("mcp-field-hint")
    ? target
    : (target.closest(".mcp-field-hint") as HTMLElement);
  if (!hint) return;
  const tooltip = hint.querySelector(".mcp-field-hint__tooltip") as HTMLElement;
  if (!tooltip) return;
  window.setTimeout(() => {
    if (!hint.matches(":hover") && !tooltip.matches(":hover")) {
      tooltip.classList.remove("is-visible");
    }
  }, 50);
}

function buildChannelOptions(props: CronProps): string[] {
  const options = ["last", ...props.channels.filter(Boolean)];
  const current = props.form.deliveryChannel?.trim();
  if (current && !options.includes(current)) {
    options.push(current);
  }
  const seen = new Set<string>();
  return options.filter((value) => {
    if (seen.has(value)) {
      return false;
    }
    seen.add(value);
    return true;
  });
}

function resolveChannelLabel(props: CronProps, channel: string): string {
  if (channel === "last") {
    return t("cronLast");
  }
  const meta = props.channelMeta?.find((entry) => entry.id === channel);
  if (meta?.label) {
    return meta.label;
  }
  return props.channelLabels?.[channel] ?? channel;
}

function formatCronStatDateTime(ms?: number | null): string {
  if (!ms) {
    return "n/a";
  }
  const date = new Date(ms);
  if (Number.isNaN(date.getTime())) {
    return "n/a";
  }
  const pad = (value: number) => String(value).padStart(2, "0");
  return `${date.getFullYear()}/${date.getMonth() + 1}/${date.getDate()} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`;
}

export function renderCronConfig(props: CronProps) {
  return html`
    <section class="stack cron-config-stack">
      <div class="card">
        <div class="card-title">${t("cronScheduler")}</div>
        <div class="card-sub">${t("cronSchedulerSub")}</div>
        <div class="stat-grid">
          <div class="stat">
            <div class="stat-label">${t("cronEnabled")}</div>
            <div class="stat-value">
              ${props.status ? (props.status.enabled ? t("commonYes") : t("commonNo")) : t("commonNA")}
            </div>
          </div>
          <div class="stat">
            <div class="stat-label">${t("cronJobs")}</div>
            <div class="stat-value">${props.status?.jobs ?? t("commonNA")}</div>
          </div>
          <div class="stat">
            <div class="stat-label">${t("overviewCronNext")}</div>
            <div class="stat-value">${formatCronStatDateTime(props.status?.nextWakeAtMs ?? null)}</div>
          </div>
        </div>
        ${props.error ? html`<div class="muted" style="margin-top: 12px;">${props.error}</div>` : nothing}
      </div>
    </section>

    <section class="card cron-jobs-card">
      ${
        props.jobs.length === 0
          ? html`
              <div class="muted">${t("cronNoJobsYet")}</div>
            `
          : html`
            <div class="list">
              ${props.jobs.map((job) => renderJob(job, props, { mode: "config" }))}
            </div>
          `
      }
    </section>
    ${props.addModalOpen ? renderCronAddModal(props) : nothing}
    ${props.editModalOpen && props.editJobId ? renderCronEditModal(props, props.editJobId) : nothing}
  `;
}

function renderCronAddModal(props: CronProps) {
  return html`
    <div class="modal-overlay" @click=${props.onCloseAddModal}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large cron-config-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 class="emp-detail-title" style="margin: 0;">${t("cronNewJob")}</h1>
            <div class="emp-detail-summary cron-config-modal__sub">${t("cronNewJobSub")}</div>
          </div>
          <button
            class="emp-detail-modal__close"
            type="button"
            aria-label=${t("commonCancel")}
            @click=${props.onCloseAddModal}
          >
            ${icons.x}
          </button>
        </div>
        <div class="cron-config-modal__body">
          ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}
          ${renderCronForm(props)}
        </div>
        <div class="modal__actions cron-config-modal__actions">
          <button class="btn" @click=${props.onCloseAddModal}>${t("commonCancel")}</button>
          <button
            class="btn primary"
            ?disabled=${props.busy ||
            (props.form.payloadKind === "agentTurn" && !props.form.payloadText.trim())}
            @click=${props.onAdd}
          >
            ${props.busy ? t("commonSaving") : t("cronAddJob")}
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderCronEditModal(props: CronProps, jobId: string) {
  return html`
    <div class="modal-overlay" @click=${props.onCloseEditModal ?? (() => {})}>
      <div
        class="modal card emp-detail-modal emp-detail-modal--large cron-config-modal"
        @click=${(e: Event) => e.stopPropagation()}
      >
        <div class="emp-detail-modal__header">
          <div class="emp-detail-header" style="flex: 1; min-width: 0;">
            <h1 class="emp-detail-title" style="margin: 0;">编辑定时任务</h1>
            <div class="emp-detail-summary cron-config-modal__sub">修改后将立即保存到本地任务配置。</div>
          </div>
          <button
            class="emp-detail-modal__close"
            type="button"
            aria-label=${t("commonCancel")}
            @click=${props.onCloseEditModal ?? (() => {})}
          >
            ${icons.x}
          </button>
        </div>
        <div class="cron-config-modal__body">
          ${props.error ? html`<div class="callout danger">${props.error}</div>` : nothing}
          ${renderCronForm(props)}
        </div>
        <div class="modal__actions cron-config-modal__actions">
          <button class="btn" @click=${props.onCloseEditModal ?? (() => {})}>${t("commonCancel")}</button>
          <button
            class="btn primary"
            ?disabled=${props.busy ||
            (props.form.payloadKind === "agentTurn" && !props.form.payloadText.trim())}
            @click=${() => props.onUpdate?.(jobId)}
          >
            ${props.busy ? t("commonSaving") : "保存修改"}
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderCronForm(props: CronProps) {
  const channelOptions = buildChannelOptions(props);
  const employeeOptions = (props.digitalEmployees ?? []).slice().sort((a, b) => {
    const an = (a.name ?? a.id ?? "").toLowerCase();
    const bn = (b.name ?? b.id ?? "").toLowerCase();
    return an.localeCompare(bn);
  });
  const selectedEmployeeId = normalizeEmployeeIdForCron(props.form.digitalEmployeeId ?? "");
  const isEditing = !!props.editModalOpen && !!props.editJobId;
  const selectedEmployee =
    selectedEmployeeId && employeeOptions.length
      ? employeeOptions.find((e) => normalizeEmployeeIdForCron(e.id ?? "").toLowerCase() === selectedEmployeeId.toLowerCase())
      : undefined;
  const employeeMissing = selectedEmployeeId !== "" && !selectedEmployee && !props.digitalEmployeesLoading;
  return html`
    <div class="form-grid">
      <label class="field">
        <span>${t("cronName")}</span>
        <span class="input"><input
          .value=${props.form.name}
          @input=${(e: Event) =>
            props.onFormChange({ name: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
      <label class="field">
        <span>${t("cronDescription")}</span>
        <span class="input"><input
          .value=${props.form.description}
          @input=${(e: Event) =>
            props.onFormChange({ description: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
      <label class="field">
        <span>${t("cronAgentId")}</span>
        <span class="input"><input
          .value=${props.form.agentId}
          @input=${(e: Event) =>
            props.onFormChange({ agentId: (e.target as HTMLInputElement).value })}
          placeholder="default"
        /></span>
      </label>
      <label class="field">
        <span>数字员工</span>
        <span class="select"><select
          @change=${async (e: Event) => {
            const el = e.target as HTMLSelectElement;
            const next = normalizeEmployeeIdForCron(el.value ?? "");
            const prev = normalizeEmployeeIdForCron(props.form.digitalEmployeeId ?? "");
            if (isEditing && prev && !next) {
              const ok = await nativeConfirm(
                `你正在移除该定时任务绑定的数字员工（${prev}）。\n\n移除后：后续触发将不再使用该数字员工会话上下文。\n\n确认移除吗？`,
              );
              if (!ok) {
                el.value = prev;
                return;
              }
            }
            props.onFormChange({ digitalEmployeeId: next });
          }}
        >
          <option value="" ?selected=${!selectedEmployeeId}>（不选择）</option>
          ${employeeOptions.map((emp) => {
            const idNorm = normalizeEmployeeIdForCron(emp.id ?? "");
            return html`<option value=${idNorm} ?selected=${idNorm === selectedEmployeeId}>${emp.name ? `${emp.name}（${idNorm}）` : idNorm}</option>`;
          })}
          ${
            employeeMissing
              ? html`<option value=${selectedEmployeeId} ?selected=${true}>—（已删除：${selectedEmployeeId}）</option>`
              : nothing
          }
        </select></span>
      </label>
      <label class="field checkbox">
        <span>${t("cronEnabled")}</span>
        <span class="checkbox"><input
          type="checkbox"
          .checked=${props.form.enabled}
          @change=${(e: Event) =>
            props.onFormChange({ enabled: (e.target as HTMLInputElement).checked })}
        /></span>
      </label>
      <label class="field">
        <span>${t("cronSchedule")}</span>
        <span class="select"><select
          .value=${props.form.scheduleKind}
          @change=${(e: Event) =>
            props.onFormChange({
              scheduleKind: (e.target as HTMLSelectElement).value as CronFormState["scheduleKind"],
            })}
        >
          <option value="every">${t("cronEvery")}</option>
          <option value="at">${t("cronAt")}</option>
          <option value="cron">${t("cronCron")}</option>
        </select></span>
      </label>
    </div>
    ${
      employeeMissing
        ? html`<div class="callout danger" style="margin-top: 12px;">
            该定时任务已配置的数字员工（${selectedEmployeeId}）不存在（可能已被删除）。请将“数字员工”改为有效项或清空后保存。
          </div>`
        : nothing
    }
    ${renderScheduleFields(props)}
    <div class="form-grid" style="margin-top: 12px;">
      <label class="field">
        <span>${t("cronSession")}</span>
        <span class="select"><select
          .value=${props.form.sessionTarget}
          @change=${(e: Event) =>
            props.onFormChange({
              sessionTarget: (e.target as HTMLSelectElement).value as CronFormState["sessionTarget"],
            })}
        >
          <option value="main">${t("cronMain")}</option>
          <option value="isolated">${t("cronIsolated")}</option>
        </select></span>
      </label>
      <label class="field">
        <span>${t("cronWakeMode")}</span>
        <span class="select"><select
          .value=${props.form.wakeMode}
          @change=${(e: Event) =>
            props.onFormChange({
              wakeMode: (e.target as HTMLSelectElement).value as CronFormState["wakeMode"],
            })}
        >
          <option value="next-heartbeat">${t("cronNextHeartbeat")}</option>
          <option value="now">${t("cronNow")}</option>
        </select></span>
      </label>
      <label class="field">
        <span>${t("cronPayload")}</span>
        <span class="select"><select
          .value=${props.form.payloadKind}
          @change=${(e: Event) =>
            props.onFormChange({
              payloadKind: (e.target as HTMLSelectElement).value as CronFormState["payloadKind"],
            })}
        >
          <option value="systemEvent">${t("cronSystemEvent")}</option>
          <option value="agentTurn">${t("cronAgentTurn")}</option>
        </select></span>
      </label>
    </div>
    <label class="field" style="margin-top: 12px;">
      <span>
        ${props.form.payloadKind === "systemEvent" ? t("cronSystemText") : t("cronAgentMessage")}${props
          .form.payloadKind === "agentTurn"
          ? html`<span style="color: var(--danger-color);"> *</span>`
          : nothing}
      </span>
      <span class="textarea"><textarea
        .value=${props.form.payloadText}
        @input=${(e: Event) =>
          props.onFormChange({
            payloadText: (e.target as HTMLTextAreaElement).value,
          })}
        rows="4"
        ?required=${props.form.payloadKind === "agentTurn"}
      ></textarea></span>
    </label>
    ${
      props.form.payloadKind === "agentTurn"
        ? html`
            <div class="form-grid" style="margin-top: 12px;">
              <label class="field">
                <span>${t("cronDelivery")}</span>
                <span class="select"><select
                  .value=${props.form.deliveryMode}
                  @change=${(e: Event) =>
                    props.onFormChange({
                      deliveryMode: (e.target as HTMLSelectElement).value as CronFormState["deliveryMode"],
                    })}
                >
                  <option value="announce">${t("cronAnnounceSummary")}</option>
                  <option value="none">${t("cronNoneInternal")}</option>
                </select></span>
              </label>
              <label class="field">
                <span>${t("cronTimeoutSeconds")}</span>
                <span class="input"><input
                  .value=${props.form.timeoutSeconds}
                  @input=${(e: Event) =>
                    props.onFormChange({
                      timeoutSeconds: (e.target as HTMLInputElement).value,
                    })}
                /></span>
              </label>
              ${
                props.form.deliveryMode === "announce"
                  ? html`
                      <label class="field">
                        <span>${t("cronChannel")}</span>
                        <span class="select"><select
                          .value=${props.form.deliveryChannel || "last"}
                          @change=${(e: Event) =>
                            props.onFormChange({
                              deliveryChannel: (e.target as HTMLSelectElement).value,
                            })}
                        >
                          ${channelOptions.map(
                            (channel) =>
                              html`<option value=${channel}>
                                ${resolveChannelLabel(props, channel)}
                              </option>`,
                          )}
                        </select></span>
                      </label>
                      <label class="field">
                        <span style="display: inline-flex; align-items: center; gap: 4px;">
                          ${t("cronTo")}
                          <span class="mcp-field-hint" @mouseenter=${showCronToTooltip} @mouseleave=${hideCronToTooltip}>
                            ${icons.info}
                            <span class="mcp-field-hint__tooltip" @mouseenter=${showCronToTooltip} @mouseleave=${hideCronToTooltip}>${t("cronToTooltip")}</span>
                          </span>
                        </span>
                        <span class="input"><input
                          .value=${props.form.deliveryTo}
                          @input=${(e: Event) =>
                            props.onFormChange({
                              deliveryTo: (e.target as HTMLInputElement).value,
                            })}
                          placeholder=${t("cronToPlaceholder")}
                        /></span>
                        <div class="muted" style="font-size: 11px; margin-top: 2px;">${t("cronToHint")}</div>
                      </label>
                    `
                  : nothing
              }
            </div>
          `
        : nothing
    }
  `;
}

export function renderCronHistory(props: CronProps) {
  const selectedJob =
    props.runsJobId == null ? undefined : props.jobs.find((job) => job.id === props.runsJobId);
  const orderedRuns = props.runs.toSorted((a, b) => b.ts - a.ts);

  return html`
    <section class="stack cron-config-stack cron-config-stack-history">
      <div class="card">
        ${
          props.jobs.length === 0
            ? nothing
            : html`
                <label class="field">
                  <span>${t("agentsTabCron")}</span>
                  <span class="select"><select @change=${(e: Event) => {
                    const value = (e.target as HTMLSelectElement).value;
                    if (value) props.onLoadRuns(value);
                  }}>
                    <option value="" ?selected=${props.runsJobId == null}>${t("cronSelectJob")}</option>
                    ${props.jobs.map(
                      (job) =>
                        html`<option value=${job.id} ?selected=${props.runsJobId === job.id}>
                          ${job.name}
                        </option>`,
                    )}
                  </select></span>
                </label>
              `
        }
        ${
          props.runsJobId == null
            ? props.jobs.length === 0
              ? html`<div class="muted" style="margin-top: 12px">${t("cronNoJobsYet")}</div>`
              : html`<div class="muted" style="margin-top: 12px">${t("cronSelectJobToInspect")}</div>`
            : orderedRuns.length === 0
              ? html`<div class="muted" style="margin-top: 12px">${t("cronNoRunsYet")}</div>`
              : html`
                  <div class="list" style="margin-top: 12px;">
                    ${orderedRuns.map((entry) => renderRun(entry, props))}
                  </div>
                `
        }
      </div>
    </section>
  `;
}

function renderScheduleFields(props: CronProps) {
  const form = props.form;
  if (form.scheduleKind === "at") {
    return html`
      <label class="field" style="margin-top: 12px;">
        <span>${t("cronRunAt")}</span>
        <input
          type="datetime-local"
          .value=${form.scheduleAt}
          @input=${(e: Event) =>
            props.onFormChange({
              scheduleAt: (e.target as HTMLInputElement).value,
            })}
        />
      </label>
    `;
  }
  if (form.scheduleKind === "every") {
    return html`
      <div class="form-grid" style="margin-top: 12px;">
        <label class="field">
          <span>${t("cronEvery")}</span>
          <span class="input"><input
            .value=${form.everyAmount}
            @input=${(e: Event) =>
              props.onFormChange({
                everyAmount: (e.target as HTMLInputElement).value,
              })}
          /></span>
        </label>
        <label class="field">
          <span>${t("cronUnit")}</span>
          <span class="select"><select
            .value=${form.everyUnit}
            @change=${(e: Event) =>
              props.onFormChange({
                everyUnit: (e.target as HTMLSelectElement).value as CronFormState["everyUnit"],
              })}
          >
            <option value="minutes">${t("cronMinutes")}</option>
            <option value="hours">${t("cronHours")}</option>
            <option value="days">${t("cronDays")}</option>
          </select></span>
        </label>
      </div>
    `;
  }
  return html`
    <div class="form-grid" style="margin-top: 12px;">
      <label class="field">
        <span>${t("cronExpression")}</span>
        <span class="input"><input
          .value=${form.cronExpr}
          @input=${(e: Event) =>
            props.onFormChange({ cronExpr: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
      <label class="field">
        <span>Timezone (optional)</span>
        <span class="input"><input
          .value=${form.cronTz}
          @input=${(e: Event) =>
            props.onFormChange({ cronTz: (e.target as HTMLInputElement).value })}
        /></span>
      </label>
    </div>
  `;
}

function renderJob(
  job: CronJob,
  props: CronProps,
  opts: { mode: "config" | "history" },
) {
  const isSelected = props.runsJobId === job.id;
  const itemClass = `list-item list-item-clickable cron-job${isSelected ? " list-item-selected" : ""}`;
  return html`
    <div
      class=${itemClass}
      @click=${() => {
        if (opts.mode === "config") {
          props.onShowHistory?.(job.id);
          return;
        }
        props.onLoadRuns(job.id);
      }}
    >
      <div class="list-main">
        <div class="list-title">${job.name}</div>
        <div class="list-sub">${formatCronSchedule(job)}</div>
        ${renderJobPayload(job)}
        ${renderJobEmployee(job, props)}
        ${job.agentId ? html`<div class="muted cron-job-agent">Agent: ${job.agentId}</div>` : nothing}
      </div>
      <div class="list-meta">
        ${renderJobState(job)}
      </div>
      <div class="cron-job-footer">
        <div class="chip-row cron-job-chips">
          <span class=${`chip ${job.enabled ? "chip-ok" : "chip-danger"}`}>
            ${job.enabled ? "enabled" : "disabled"}
          </span>
          <span class="chip">${job.sessionTarget}</span>
          <span class="chip">${job.wakeMode}</span>
        </div>
        <div class="row cron-job-actions">
          ${
            opts.mode === "config"
              ? html`
                <button
                  class="btn"
                  ?disabled=${props.busy}
                  @click=${(event: Event) => {
                    event.stopPropagation();
                    props.onOpenEditModal?.(job);
                  }}
                >
                  Edit
                </button>
              `
              : nothing
          }
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              props.onToggle(job, !job.enabled);
            }}
          >
            ${job.enabled ? "Disable" : "Enable"}
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              props.onRun(job);
            }}
          >
            Run
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${(event: Event) => {
              event.stopPropagation();
              if (opts.mode === "config") {
                props.onShowHistory?.(job.id);
                return;
              }
              props.onLoadRuns(job.id);
            }}
          >
            History
          </button>
          <button
            class="btn"
            ?disabled=${props.busy}
            @click=${async (event: Event) => {
              event.stopPropagation();
              if (props.confirmRemove && !(await nativeConfirm(t("cronDeleteConfirm")))) {
                return;
              }
              props.onRemove(job);
            }}
          >
            Remove
          </button>
        </div>
      </div>
    </div>
  `;
}

function renderJobEmployee(job: CronJob, props: CronProps) {
  const raw = normalizeEmployeeIdForCron(job.digitalEmployeeId ?? "");
  if (!raw) return nothing;
  const list = props.digitalEmployees ?? [];
  const hit = list.find((e) => normalizeEmployeeIdForCron(e.id ?? "").toLowerCase() === raw.toLowerCase());
  if (!hit) {
    return html`<div class="cron-job-detail">
      <span class="cron-job-detail-label">数字员工</span>
      <span class="muted cron-job-detail-value">—（已删除：${raw}，建议移除该配置）</span>
    </div>`;
  }
  const label = hit.name ? `${hit.name}（${raw}）` : raw;
  return html`<div class="cron-job-detail">
    <span class="cron-job-detail-label">数字员工</span>
    <span class="muted cron-job-detail-value">${label}</span>
  </div>`;
}

function renderJobPayload(job: CronJob) {
  if (job.payload.kind === "systemEvent") {
    return html`<div class="cron-job-detail">
      <span class="cron-job-detail-label">System</span>
      <span class="muted cron-job-detail-value">${job.payload.text}</span>
    </div>`;
  }

  const delivery = job.delivery;
  const deliveryTarget =
    delivery?.channel || delivery?.to
      ? ` (${delivery.channel ?? "last"}${delivery.to ? ` -> ${delivery.to}` : ""})`
      : "";

  return html`
    <div class="cron-job-detail">
      <span class="cron-job-detail-label">Prompt</span>
      <span class="muted cron-job-detail-value">${job.payload.message}</span>
    </div>
    ${
      delivery
        ? html`<div class="cron-job-detail">
            <span class="cron-job-detail-label">Delivery</span>
            <span class="muted cron-job-detail-value">${delivery.mode}${deliveryTarget}</span>
          </div>`
        : nothing
    }
  `;
}

function formatStateRelative(ms?: number) {
  if (typeof ms !== "number" || !Number.isFinite(ms)) {
    return "n/a";
  }
  return formatAgo(ms);
}

function renderJobState(job: CronJob) {
  const status = job.state?.lastStatus ?? "n/a";
  const statusClass =
    status === "ok"
      ? "cron-job-status-ok"
      : status === "error"
        ? "cron-job-status-error"
        : status === "skipped"
          ? "cron-job-status-skipped"
          : "cron-job-status-na";
  const nextRunAtMs = job.state?.nextRunAtMs;
  const lastRunAtMs = job.state?.lastRunAtMs;

  return html`
    <div class="cron-job-state">
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Status</span>
        <span class=${`cron-job-status-pill ${statusClass}`}>${status}</span>
      </div>
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Next</span>
        <span class="cron-job-state-value" title=${formatMs(nextRunAtMs)}>
          ${formatStateRelative(nextRunAtMs)}
        </span>
      </div>
      <div class="cron-job-state-row">
        <span class="cron-job-state-key">Last</span>
        <span class="cron-job-state-value" title=${formatMs(lastRunAtMs)}>
          ${formatStateRelative(lastRunAtMs)}
        </span>
      </div>
    </div>
  `;
}

function cronRunChatSessionKey(entry: CronRunLogEntry): string | null {
  const fromKey = cronJobSessionKeyFromRunSessionKey(entry.sessionKey);
  if (fromKey) {
    return fromKey;
  }
  const jid = entry.jobId?.trim();
  return jid ? `agent:main:cron:${jid}` : null;
}

function renderRun(entry: CronRunLogEntry, props: CronProps) {
  const chatSessionKey = cronRunChatSessionKey(entry);
  const chatUrl = chatSessionKey
    ? `${pathForTab("message", props.basePath)}?session=${encodeURIComponent(chatSessionKey)}`
    : null;
  const jobRunId = resolveJobRunIdForCronEntry(entry, props.jobRuns ?? []);
  return html`
    <div class="list-item">
      <div class="list-main">
        <div class="list-title">${entry.status}</div>
        <div class="list-sub">${entry.summary ?? ""}</div>
      </div>
      <div class="list-meta">
        <div>${formatMs(entry.ts)}</div>
        <div class="muted">${entry.durationMs ?? 0}ms</div>
        ${
          jobRunId && props.onOpenJobRun
            ? html`<div><button type="button" class="btn btn--link session-link" @click=${() => props.onOpenJobRun?.(jobRunId)}>执行链路</button></div>`
            : nothing
        }
        ${
          chatUrl
            ? html`<div><a class="session-link" href=${chatUrl}>Open run chat</a></div>`
            : nothing
        }
        ${entry.error ? html`<div class="muted">${entry.error}</div>` : nothing}
      </div>
    </div>
  `;
}
