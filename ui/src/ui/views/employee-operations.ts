import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import type { DigitalEmployee } from "./digital-employee.ts";
import type { EmployeeTask, EmployeeEffectiveness } from "../controllers/employee-tasks.ts";
import { nativeConfirm } from "../native-dialog-bridge.ts";
import { opsCapabilityLabel, opsDomainLabel } from "../ops/taxonomy.ts";

export type EmployeeOperationsProps = {
  employees: DigitalEmployee[];
  mode: "tasks" | "effectiveness";
  onOpenEmployees: () => void;
  // Tasks state
  tasks: EmployeeTask[];
  tasksLoading: boolean;
  tasksError: string | null;
  activeTask: EmployeeTask | null;
  filterEmployee: string;
  filterStatus: string;
  filterQuery: string;
  // Effectiveness state
  effectiveness: EmployeeEffectiveness | null;
  effectivenessLoading: boolean;
  effectivenessError: string | null;
  // Actions
  onFilterEmployeeChange: (val: string) => void;
  onFilterStatusChange: (val: string) => void;
  onFilterQueryChange: (val: string) => void;
  onSetActiveTask: (task: EmployeeTask | null) => void;
  onRateTask: (id: string, evaluation: "accepted" | "rejected") => void;
  onDeleteTask: (id: string) => void;
  onOpenChat: (sessionKey: string) => void;
};

export function renderEmployeeOperations(props: EmployeeOperationsProps) {
  const isTasks = props.mode === "tasks";

  return html`
    <main class="ops-dashboard">
      <div class="ops-dashboard-header">
        <div>
          <h1>${isTasks ? "员工任务记录" : "员工效能评估"}</h1>
          <p class="muted">
            ${isTasks
              ? "将数字员工从一次会话升级为可追踪、可确认、可复盘的工作任务。"
              : "用任务量、闭环率、采纳率、降噪率、人时节省和成本消耗量化数字员工价值。"}
          </p>
        </div>
        <div class="ops-dashboard-actions__inner">
          <button class="ops-dashboard-actions__btn" type="button" @click=${props.onOpenEmployees}>
            ${icons.users} 我的员工
          </button>
        </div>
      </div>

      ${isTasks ? renderTasksView(props) : renderEffectivenessView(props)}
    </main>

    ${props.activeTask ? renderTaskDetailsModal(props.activeTask, props) : nothing}
  `;
}

function renderTasksView(props: EmployeeOperationsProps) {
  if (props.tasksLoading && props.tasks.length === 0) {
    return html`<div class="loading-state">${icons.loader} 加载任务记录中...</div>`;
  }

  if (props.tasksError) {
    return html`<div class="callout danger" style="margin-top: 16px;">${props.tasksError}</div>`;
  }

  return html`
    <!-- 过滤器栏 -->
    <section class="filters-bar" style="margin-top: 16px; display: flex; gap: 12px; flex-wrap: wrap; align-items: flex-end; background: var(--bg-card); padding: 12px; border-radius: 8px; border: 1px solid var(--border-color);">
      <div class="field" style="margin: 0; min-width: 160px;">
        <span style="font-size: 12px; font-weight: 500; margin-bottom: 4px;">筛选员工</span>
        <span class="select">
          <select
            .value=${props.filterEmployee}
            @change=${(e: Event) => props.onFilterEmployeeChange((e.target as HTMLSelectElement).value)}
          >
            <option value="">全部员工</option>
            ${props.employees.map(
              (emp) => html`<option value=${emp.id}>${emp.name || emp.id}</option>`,
            )}
          </select>
        </span>
      </div>

      <div class="field" style="margin: 0; min-width: 140px;">
        <span style="font-size: 12px; font-weight: 500; margin-bottom: 4px;">任务状态</span>
        <span class="select">
          <select
            .value=${props.filterStatus}
            @change=${(e: Event) => props.onFilterStatusChange((e.target as HTMLSelectElement).value)}
          >
            <option value="">全部状态</option>
            <option value="success">执行成功</option>
            <option value="failed">执行失败</option>
            <option value="running">进行中</option>
          </select>
        </span>
      </div>

      <div class="field" style="margin: 0; flex: 1; min-width: 200px;">
        <span style="font-size: 12px; font-weight: 500; margin-bottom: 4px;">搜索任务</span>
        <span class="input">
          <input
            type="text"
            .value=${props.filterQuery}
            @input=${(e: Event) => props.onFilterQueryChange((e.target as HTMLInputElement).value)}
            placeholder="搜索输入/结论/任务 ID"
          />
        </span>
      </div>
    </section>

    <!-- 任务数据列表 -->
    <section class="card" style="margin-top: 16px; padding: 0; overflow: hidden; border: 1px solid var(--border-color);">
      <div style="overflow-x: auto;">
        <table class="ops-table" style="width: 100%; border-collapse: collapse; text-align: left;">
          <thead>
            <tr style="background: var(--bg-header); border-bottom: 1px solid var(--border-color); font-size: 12px; color: var(--text-muted);">
              <th style="padding: 12px 16px; font-weight: 600;">任务 ID</th>
              <th style="padding: 12px 16px; font-weight: 600;">执行员工</th>
              <th style="padding: 12px 16px; font-weight: 600;">归属技术域</th>
              <th style="padding: 12px 16px; font-weight: 600;">服务能力域</th>
              <th style="padding: 12px 16px; font-weight: 600;">关联对象</th>
              <th style="padding: 12px 16px; font-weight: 600;">执行状态</th>
              <th style="padding: 12px 16px; font-weight: 600;">采纳反馈</th>
              <th style="padding: 12px 16px; font-weight: 600;">操作</th>
            </tr>
          </thead>
          <tbody>
            ${props.tasks.length === 0
              ? html`
                  <tr>
                    <td colspan="8" style="padding: 32px; text-align: center; color: var(--text-muted);">
                      暂无匹配的任务记录
                    </td>
                  </tr>
                `
              : props.tasks.map((task) => {
                  const emp = props.employees.find((e) => e.id === task.employeeId);
                  const empName = emp ? (emp.name || emp.id) : task.employeeId;

                  const executionStatus = task.executionStatus || task.status;
                  let statusBadge = html`<span class="badge badge--blue">进行中</span>`;
                  if (executionStatus === "succeeded" || executionStatus === "success") {
                    statusBadge = html`<span class="badge badge--green">成功</span>`;
                  } else if (executionStatus === "failed") {
                    statusBadge = html`<span class="badge badge--red">失败</span>`;
                  } else if (executionStatus === "pending") {
                    statusBadge = html`<span class="badge badge--muted">待执行</span>`;
                  }

                  let evalBadge = html`<span class="badge badge--muted">${workflowLabel(task.workflowStatus)}</span>`;
                  if (task.evaluation === "accepted") {
                    evalBadge = html`<span class="badge badge--green" style="font-weight: 500;">已采纳</span>`;
                  } else if (task.evaluation === "rejected") {
                    evalBadge = html`<span class="badge badge--red" style="font-weight: 500;">已驳回</span>`;
                  }

                  const startedStr = task.startedAt
                    ? new Date(task.startedAt).toLocaleString("zh-CN", { hour12: false })
                    : "-";

                  return html`
                    <tr style="border-bottom: 1px solid var(--border-color); font-size: 13px; cursor: pointer; transition: background 0.2s;" @click=${() => props.onSetActiveTask(task)}>
                      <td style="padding: 12px 16px;" title=${task.id}>
                        <code style="font-family: monospace; font-size: 11px;">${task.id.slice(0, 8)}...</code>
                      </td>
                      <td style="padding: 12px 16px; font-weight: 500;">${empName}</td>
                      <td style="padding: 12px 16px;">
                        <span class="badge badge--purple">${opsDomainLabel(task.domainKey) || "通用"}</span>
                      </td>
                      <td style="padding: 12px 16px;">
                        <span class="badge badge--blue">${opsCapabilityLabel(task.capabilityKey) || "其它"}</span>
                      </td>
                      <td style="padding: 12px 16px; color: var(--text-muted); font-size: 12px;">
                        ${task.objectRef || "-"}
                      </td>
                      <td style="padding: 12px 16px;">${statusBadge}</td>
                      <td style="padding: 12px 16px;">${evalBadge}</td>
                      <td style="padding: 12px 16px;" @click=${(e: Event) => e.stopPropagation()}>
                        <div class="row" style="gap: 6px;">
                          <button class="btn btn--sm" type="button" @click=${() => props.onSetActiveTask(task)}>
                            详情
                          </button>
                          <button class="btn btn--sm" type="button" @click=${() => props.onOpenChat(taskSessionKey(task))}>
                            会话
                          </button>
                          <button class="btn btn--sm btn--danger" type="button" @click=${async () => {
                            if (await nativeConfirm("确定要删除这条任务记录吗？")) {
                              props.onDeleteTask(task.id);
                            }
                          }}>
                            删除
                          </button>
                        </div>
                      </td>
                    </tr>
                  `;
                })}
          </tbody>
        </table>
      </div>
    </section>
  `;
}

function renderEffectivenessView(props: EmployeeOperationsProps) {
  if (props.effectivenessLoading && !props.effectiveness) {
    return html`<div class="loading-state">${icons.loader} 计算效能数据中...</div>`;
  }

  if (props.effectivenessError) {
    return html`<div class="callout danger" style="margin-top: 16px;">${props.effectivenessError}</div>`;
  }

  const eff = props.effectiveness;
  if (!eff) {
    return html`<div class="loading-state">暂无效能评估数据</div>`;
  }

  return html`
    ${eff.metricConfidence === "insufficient_data"
      ? html`<div class="callout" style="margin-top: 16px;">
          当前效能仅统计任务闭环和采纳反馈；告警降噪、人时、成本、MTTR 需要任务写入 metrics 后才展示真实值。
        </div>`
      : nothing}
    <!-- 指标卡片 -->
    <section class="stats-grid" style="margin-top: 16px;">
      <article class="stat-card">
        <div class="stat-icon stat-icon--blue">${icons.historyClock}</div>
        <div class="stat-content">
          <h3>累计处理任务</h3>
          <div class="stat-value">${eff.taskCount}</div>
          <div class="stat-sub">个</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon stat-icon--ok">${icons.overviewGrid}</div>
        <div class="stat-content">
          <h3>任务闭环率</h3>
          <div class="stat-value">${(eff.autoCloseRate * 100).toFixed(1)}</div>
          <div class="stat-sub">%</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon stat-icon--purple">${icons.check}</div>
        <div class="stat-content">
          <h3>诊断采纳率</h3>
          <div class="stat-value">${(eff.adoptionRate * 100).toFixed(1)}</div>
          <div class="stat-sub">%</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon stat-icon--warn">${icons.zap}</div>
        <div class="stat-content">
          <h3>告警降噪比</h3>
          <div class="stat-value">${(eff.noiseReductionRate * 100).toFixed(1)}</div>
          <div class="stat-sub">%</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon stat-icon--blue">${icons.historyClock}</div>
        <div class="stat-content">
          <h3>累计节省人时</h3>
          <div class="stat-value">${eff.savedHours.toFixed(1)}</div>
          <div class="stat-sub">小时</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon stat-icon--danger">${icons.wrench}</div>
        <div class="stat-content">
          <h3>累计计算成本</h3>
          <div class="stat-value">${eff.costSpent.toFixed(2)}</div>
          <div class="stat-sub">USD</div>
        </div>
      </article>
    </section>

    <!-- 图标/对比网格 -->
    <div class="row" style="margin-top: 16px; gap: 16px; align-items: flex-start; flex-wrap: wrap;">
      
      <!-- 能力域分布 -->
      <section class="card" style="flex: 1; min-width: 340px; border: 1px solid var(--border-color);">
        <div class="card-title">运维能力域任务分布</div>
        <div style="display: grid; gap: 12px; margin-top: 12px;">
          ${Object.entries(eff.taskBreakdown).map(([key, val]) => {
            const pct = eff.taskCount > 0 ? (val / eff.taskCount) * 100 : 0;
            return html`
              <div>
                <div class="row" style="justify-content: space-between; font-size: 13px; margin-bottom: 4px;">
                  <span>${opsCapabilityLabel(key)}</span>
                  <span class="muted">${val} 次 (${pct.toFixed(0)}%)</span>
                </div>
                <div style="background: rgba(0,0,0,0.06); height: 8px; border-radius: 4px; overflow: hidden;">
                  <div style="background: var(--blue-color); width: ${pct}%; height: 100%; border-radius: 4px;"></div>
                </div>
              </div>
            `;
          })}
        </div>
      </section>

      <!-- 技术域分布 -->
      <section class="card" style="flex: 1; min-width: 340px; border: 1px solid var(--border-color);">
        <div class="card-title">技术域任务覆盖分布</div>
        <div style="display: grid; gap: 12px; margin-top: 12px;">
          ${Object.entries(eff.domainBreakdown).map(([key, val]) => {
            const pct = eff.taskCount > 0 ? (val / eff.taskCount) * 100 : 0;
            return html`
              <div>
                <div class="row" style="justify-content: space-between; font-size: 13px; margin-bottom: 4px;">
                  <span>${opsDomainLabel(key)}</span>
                  <span class="muted">${val} 次 (${pct.toFixed(0)}%)</span>
                </div>
                <div style="background: rgba(0,0,0,0.06); height: 8px; border-radius: 4px; overflow: hidden;">
                  <div style="background: var(--purple-color); width: ${pct}%; height: 100%; border-radius: 4px;"></div>
                </div>
              </div>
            `;
          })}
        </div>
      </section>

    </div>
  `;
}

function renderTaskDetailsModal(task: EmployeeTask, props: EmployeeOperationsProps) {
  const emp = props.employees.find((e) => e.id === task.employeeId);
  const empName = emp ? (emp.name || emp.id) : task.employeeId;
  const executionStatus = task.executionStatus || task.status;

  return html`
    <div class="modal-overlay" @click=${() => props.onSetActiveTask(null)}>
      <div class="modal card" style="max-width: 800px; width: 90%; display: flex; flex-direction: column; max-height: 85vh; padding: 24px;" @click=${(e: Event) => e.stopPropagation()}>
        
        <div class="row" style="justify-content: space-between; align-items: flex-start; border-bottom: 1px solid var(--border-color); padding-bottom: 12px;">
          <div>
            <div class="card-title" style="font-size: 18px; font-weight: 600;">任务详细记录</div>
            <div class="muted" style="font-size: 12px; margin-top: 4px;">任务 ID: <code>${task.id}</code></div>
          </div>
          <button class="btn btn--sm" type="button" @click=${() => props.onSetActiveTask(null)}>
            ${icons.folder} 关闭
          </button>
        </div>

        <div style="flex: 1; overflow-y: auto; margin-top: 16px; display: grid; gap: 16px; padding-right: 8px;">
          <!-- 元数据网格 -->
          <div style="display: grid; grid-template-columns: repeat(auto-fit, minmax(180px, 1fr)); gap: 12px; background: var(--bg-header); padding: 12px; border-radius: 8px; border: 1px solid var(--border-color);">
            <div>
              <span class="muted" style="font-size: 11px; display: block;">执行员工</span>
              <span style="font-size: 13px; font-weight: 500;">${empName}</span>
            </div>
            <div>
              <span class="muted" style="font-size: 11px; display: block;">运维技术域</span>
              <span class="badge badge--purple" style="margin-top: 2px;">${opsDomainLabel(task.domainKey) || "通用"}</span>
            </div>
            <div>
              <span class="muted" style="font-size: 11px; display: block;">服务能力域</span>
              <span class="badge badge--blue" style="margin-top: 2px;">${opsCapabilityLabel(task.capabilityKey) || "其它"}</span>
            </div>
            <div>
              <span class="muted" style="font-size: 11px; display: block;">关联对象引用</span>
              <span style="font-size: 12px;">${task.objectRef || "-"}</span>
            </div>
            <div>
              <span class="muted" style="font-size: 11px; display: block;">触发方式 / 操作人</span>
              <span style="font-size: 12px;">${task.triggerType} (${task.operator})</span>
            </div>
            <div>
              <span class="muted" style="font-size: 11px; display: block;">状态 / 开始时间</span>
              <span style="font-size: 12px;">
                ${executionStatusLabel(executionStatus)} / ${workflowLabel(task.workflowStatus)}
                (${new Date(task.startedAt).toLocaleTimeString("zh-CN", { hour12: false })})
              </span>
            </div>
          </div>

          <!-- 用户输入 -->
          <div>
            <h4 style="font-size: 13px; font-weight: 600; margin-bottom: 6px;">输入指令 (User Query)</h4>
            <div style="background: rgba(0,0,0,0.03); border: 1px solid var(--border-color); border-radius: 6px; padding: 12px; font-family: monospace; font-size: 12px; white-space: pre-wrap; max-height: 120px; overflow-y: auto;">
              ${task.input || "无指令"}
            </div>
          </div>

          <!-- 结论概要 -->
          <div>
            <h4 style="font-size: 13px; font-weight: 600; margin-bottom: 6px;">结论摘要 (Conclusion Summary)</h4>
            <div style="background: rgba(0,0,0,0.03); border: 1px solid var(--border-color); border-radius: 6px; padding: 12px; font-family: monospace; font-size: 12px; white-space: pre-wrap; max-height: 120px; overflow-y: auto;">
              ${task.conclusion || "无摘要结论"}
            </div>
          </div>

          <!-- 完整输出 -->
          <div>
            <h4 style="font-size: 13px; font-weight: 600; margin-bottom: 6px;">完整分析报告 (Full Report)</h4>
            <div style="background: var(--bg-card); border: 1px solid var(--border-color); border-radius: 6px; padding: 16px; font-family: monospace; font-size: 12px; white-space: pre-wrap; max-height: 300px; overflow-y: auto; line-height: 1.5; color: var(--text-primary);">
              ${task.output || "无输出内容"}
            </div>
          </div>
        </div>

        <!-- 反馈区 -->
        <div class="row" style="margin-top: 16px; border-top: 1px solid var(--border-color); padding-top: 16px; justify-content: space-between; align-items: center;">
          <div class="row" style="gap: 8px; align-items: center;">
            <span class="muted" style="font-size: 12px;">评价与反馈：</span>
            <button class="btn ${task.evaluation === "accepted" ? "primary" : ""}" type="button" @click=${() => props.onRateTask(task.id, "accepted")}>
              采纳此报告结论
            </button>
            <button class="btn btn--danger ${task.evaluation === "rejected" ? "primary" : ""}" type="button" @click=${() => props.onRateTask(task.id, "rejected")}>
              驳回此诊断结论
            </button>
          </div>
          <div class="row" style="gap: 8px;">
            <button class="btn" type="button" @click=${() => props.onSetActiveTask(null)}>
              返回
            </button>
            <button class="btn primary" type="button" @click=${() => {
              props.onOpenChat(taskSessionKey(task));
              props.onSetActiveTask(null);
            }}>
              进入会话追问
            </button>
          </div>
        </div>

      </div>
    </div>
  `;
}

function executionStatusLabel(status?: string) {
  switch (status) {
    case "succeeded":
    case "success":
      return "执行成功";
    case "failed":
      return "执行失败";
    case "running":
      return "执行中";
    case "pending":
      return "待执行";
    default:
      return status || "未知";
  }
}

function workflowLabel(status?: string) {
  switch (status) {
    case "closed":
      return "已闭环";
    case "rejected":
      return "已驳回";
    case "waiting_approval":
      return "待确认";
    case "open":
      return "处理中";
    default:
      return status || "待确认";
  }
}

function taskSessionKey(task: EmployeeTask) {
  return `agent:main:employee:${task.employeeId}:run:${task.sessionId || task.id}`;
}
