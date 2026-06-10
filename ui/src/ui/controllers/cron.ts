import type { GatewayBrowserClient } from "../gateway.ts";
import type { CronJob, CronRunLogEntry, CronStatus } from "../types.ts";
import type { CronFormState } from "../ui-types.ts";
import { toNumber } from "../format.ts";
import { fetchOpsJobRuns, type OpsJobRunRecord } from "./ops-job-runs.ts";

export type CronState = {
  client: GatewayBrowserClient | null;
  connected: boolean;
  cronLoading: boolean;
  cronJobs: CronJob[];
  cronStatus: CronStatus | null;
  cronError: string | null;
  cronForm: CronFormState;
  cronRunsJobId: string | null;
  cronRuns: CronRunLogEntry[];
  cronJobRuns?: OpsJobRunRecord[];
  cronBusy: boolean;
  gatewayHttpUrl?: string;
  settings?: { token: string };
};

function normalizeEmployeeIdForCron(raw: string): string {
  let s = (raw ?? "").trim();
  if (!s) return "";
  // 兼容市场/目录层的 local:<id> 表示：存储与会话 key 只使用去前缀后的稳定 id。
  if (s.toLowerCase().startsWith("local:")) {
    s = s.slice("local:".length);
  }
  // cron 的 employee 会话 key 以 ":" 分隔，id 内不能包含冒号。
  s = s.replaceAll(":", "-");
  return s.trim().toLowerCase();
}

export async function loadCronStatus(state: CronState) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    const res = await state.client.request<CronStatus>("cron.status", {});
    state.cronStatus = res;
  } catch (err) {
    state.cronError = String(err);
  }
}

export async function loadCronJobs(state: CronState) {
  if (!state.client || !state.connected) {
    return;
  }
  if (state.cronLoading) {
    return;
  }
  state.cronLoading = true;
  state.cronError = null;
  try {
    const res = await state.client.request<{ jobs?: Array<CronJob> }>("cron.list", {
      includeDisabled: true,
    });
    state.cronJobs = Array.isArray(res.jobs) ? res.jobs : [];
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronLoading = false;
  }
}

export function buildCronSchedule(form: CronFormState) {
  if (form.scheduleKind === "at") {
    const ms = Date.parse(form.scheduleAt);
    if (!Number.isFinite(ms)) {
      throw new Error("Invalid run time.");
    }
    return { kind: "at" as const, at: new Date(ms).toISOString() };
  }
  if (form.scheduleKind === "every") {
    const amount = toNumber(form.everyAmount, 0);
    if (amount <= 0) {
      throw new Error("Invalid interval amount.");
    }
    const unit = form.everyUnit;
    const mult = unit === "minutes" ? 60_000 : unit === "hours" ? 3_600_000 : 86_400_000;
    return { kind: "every" as const, everyMs: amount * mult };
  }
  const expr = form.cronExpr.trim();
  if (!expr) {
    throw new Error("Cron expression required.");
  }
  return { kind: "cron" as const, expr, tz: form.cronTz.trim() || undefined };
}

export function buildCronPayload(form: CronFormState) {
  if (form.payloadKind === "systemEvent") {
    const text = form.payloadText.trim();
    if (!text) {
      throw new Error("System event text required.");
    }
    return { kind: "systemEvent" as const, text };
  }
  const message = form.payloadText.trim();
  if (!message) {
    throw new Error("Agent message required.");
  }
  const payload: {
    kind: "agentTurn";
    message: string;
    timeoutSeconds?: number;
  } = { kind: "agentTurn", message };
  const timeoutSeconds = toNumber(form.timeoutSeconds, 0);
  if (timeoutSeconds > 0) {
    payload.timeoutSeconds = timeoutSeconds;
  }
  return payload;
}

function pad2(n: number): string {
  return String(n).padStart(2, "0");
}

function toLocalDateTimeInputValue(ms: number): string {
  const d = new Date(ms);
  if (Number.isNaN(d.getTime())) return "";
  // datetime-local expects local time "YYYY-MM-DDTHH:mm"
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

export function cronFormFromJob(job: CronJob, prev: CronFormState): CronFormState {
  const scheduleKind = (job.schedule?.kind ?? prev.scheduleKind) as CronFormState["scheduleKind"];
  const next: CronFormState = { ...prev };
  next.name = job.name ?? "";
  next.description = job.description ?? "";
  next.agentId = job.agentId ?? "";
  next.digitalEmployeeId = normalizeEmployeeIdForCron(job.digitalEmployeeId ?? "");
  next.enabled = !!job.enabled;
  next.sessionTarget = (job.sessionTarget ?? prev.sessionTarget) as CronFormState["sessionTarget"];
  next.wakeMode = (job.wakeMode ?? prev.wakeMode) as CronFormState["wakeMode"];

  // Schedule
  next.scheduleKind = scheduleKind;
  if (scheduleKind === "at") {
    const atMs = Date.parse(job.schedule?.at ?? "");
    next.scheduleAt = Number.isFinite(atMs) ? toLocalDateTimeInputValue(atMs) : "";
  } else if (scheduleKind === "every") {
    const everyMs = Number(job.schedule?.everyMs ?? 0);
    const minute = 60_000;
    const hour = 3_600_000;
    const day = 86_400_000;
    if (everyMs > 0 && everyMs % day === 0) {
      next.everyUnit = "days";
      next.everyAmount = String(Math.max(1, Math.round(everyMs / day)));
    } else if (everyMs > 0 && everyMs % hour === 0) {
      next.everyUnit = "hours";
      next.everyAmount = String(Math.max(1, Math.round(everyMs / hour)));
    } else {
      next.everyUnit = "minutes";
      next.everyAmount = String(Math.max(1, Math.round((everyMs || minute) / minute)));
    }
  } else {
    next.cronExpr = String(job.schedule?.expr ?? "").trim() || prev.cronExpr;
    next.cronTz = String((job.schedule as any)?.tz ?? "").trim();
  }

  // Payload
  const pk = (job.payload?.kind ?? prev.payloadKind) as CronFormState["payloadKind"];
  next.payloadKind = pk;
  if (pk === "systemEvent") {
    next.payloadText = String((job.payload as any)?.text ?? "");
  } else {
    next.payloadText = String((job.payload as any)?.message ?? "");
  }

  // Delivery (only relevant for agentTurn in UI, but keep mapping)
  const mode = (job.delivery?.mode ?? "none") as CronFormState["deliveryMode"];
  next.deliveryMode = mode === "announce" ? "announce" : "none";
  next.deliveryChannel = job.delivery?.channel ?? "last";
  next.deliveryTo = job.delivery?.to ?? "";

  return next;
}

export async function addCronJob(state: CronState) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    const schedule = buildCronSchedule(state.cronForm);
    const payload = buildCronPayload(state.cronForm);
    const delivery =
      state.cronForm.sessionTarget === "isolated" &&
      state.cronForm.payloadKind === "agentTurn" &&
      state.cronForm.deliveryMode
        ? {
            mode: state.cronForm.deliveryMode === "announce" ? "announce" : "none",
            channel: state.cronForm.deliveryChannel.trim() || "last",
            to: state.cronForm.deliveryTo.trim() || undefined,
          }
        : undefined;
    const agentId = state.cronForm.agentId.trim();
    const digitalEmployeeId = normalizeEmployeeIdForCron(state.cronForm.digitalEmployeeId.trim());
    const job = {
      name: state.cronForm.name.trim(),
      description: state.cronForm.description.trim() || undefined,
      agentId: agentId || undefined,
      digitalEmployeeId: digitalEmployeeId || undefined,
      enabled: state.cronForm.enabled,
      schedule,
      sessionTarget: state.cronForm.sessionTarget,
      wakeMode: state.cronForm.wakeMode,
      payload,
      delivery,
    };
    if (!job.name) {
      throw new Error("Name required.");
    }
    await state.client.request("cron.add", job);
    state.cronForm = {
      ...state.cronForm,
      name: "",
      description: "",
      payloadText: "",
    };
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function updateCronJob(state: CronState, jobId: string) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    const schedule = buildCronSchedule(state.cronForm);
    const payload = buildCronPayload(state.cronForm);
    const delivery =
      state.cronForm.sessionTarget === "isolated" &&
      state.cronForm.payloadKind === "agentTurn" &&
      state.cronForm.deliveryMode
        ? {
            mode: state.cronForm.deliveryMode === "announce" ? "announce" : "none",
            channel: state.cronForm.deliveryChannel.trim() || "last",
            to: state.cronForm.deliveryTo.trim() || undefined,
          }
        : undefined;
    const patch = {
      enabled: state.cronForm.enabled,
      name: state.cronForm.name.trim(),
      description: state.cronForm.description.trim(),
      agentId: state.cronForm.agentId.trim() || undefined,
      digitalEmployeeId: normalizeEmployeeIdForCron(state.cronForm.digitalEmployeeId.trim()) || "",
      schedule,
      sessionTarget: state.cronForm.sessionTarget,
      wakeMode: state.cronForm.wakeMode,
      payload,
      delivery,
    };
    if (!patch.name) {
      throw new Error("Name required.");
    }
    await state.client.request("cron.update", { id: jobId, patch });
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function toggleCronJob(state: CronState, job: CronJob, enabled: boolean) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.update", { id: job.id, patch: { enabled } });
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function runCronJob(state: CronState, job: CronJob) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.run", { id: job.id, mode: "force" });
    await loadCronRuns(state, job.id);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function removeCronJob(state: CronState, job: CronJob) {
  if (!state.client || !state.connected || state.cronBusy) {
    return;
  }
  state.cronBusy = true;
  state.cronError = null;
  try {
    await state.client.request("cron.remove", { id: job.id });
    if (state.cronRunsJobId === job.id) {
      state.cronRunsJobId = null;
      state.cronRuns = [];
    }
    await loadCronJobs(state);
    await loadCronStatus(state);
  } catch (err) {
    state.cronError = String(err);
  } finally {
    state.cronBusy = false;
  }
}

export async function loadCronRuns(state: CronState, jobId: string) {
  if (!state.client || !state.connected) {
    return;
  }
  try {
    const res = await state.client.request<{ entries?: Array<CronRunLogEntry> }>("cron.runs", {
      id: jobId,
      limit: 200,
    });
    state.cronRunsJobId = jobId;
    state.cronRuns = Array.isArray(res.entries) ? res.entries : [];
    if (state.gatewayHttpUrl) {
      try {
        const jobRuns = await fetchOpsJobRuns(
          {
            gatewayHttpUrl: state.gatewayHttpUrl,
            settings: state.settings ?? { token: "" },
          },
          { jobId, limit: 200 },
        );
        state.cronJobRuns = jobRuns.runs;
      } catch {
        state.cronJobRuns = [];
      }
    }
  } catch (err) {
    state.cronError = String(err);
  }
}
