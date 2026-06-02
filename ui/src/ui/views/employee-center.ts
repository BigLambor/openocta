import { html } from "lit";
import { icons } from "../icons.ts";
import type { DigitalEmployee } from "./digital-employee.ts";

export type EmployeeCenterProps = {
  employees: DigitalEmployee[];
  loading: boolean;
  onOpenMarket: () => void;
  onOpenEmployees: () => void;
  onOpenSwarm: () => void;
  onOpenTasks: () => void;
  onOpenEffectiveness: () => void;
  onCreateEmployee: () => void;
};

const ROLE_CARDS = [
  {
    title: "值班运维数字员工",
    desc: "负责告警接收、降噪、聚合、初判和升级通知。",
    domain: "可观测与告警",
  },
  {
    title: "巡检数字员工",
    desc: "负责日常巡检、深度巡检、健康评分和风险清单。",
    domain: "健康度与巡检",
  },
  {
    title: "诊断数字员工",
    desc: "负责故障定位、根因分析、影响面分析和处置建议。",
    domain: "故障诊断与应急",
  },
  {
    title: "治理数字员工",
    desc: "负责重复告警、作业稳定性、配置偏差和 SLA 风险治理。",
    domain: "治理与优化",
  },
  {
    title: "容量与成本数字员工",
    desc: "负责容量预测、资源利用率、性能瓶颈和成本归因。",
    domain: "容量性能与成本",
  },
  {
    title: "变更护航数字员工",
    desc: "负责变更前评估、变更中观测、变更后验证和回滚建议。",
    domain: "变更配置与合规",
  },
];

export function renderEmployeeCenter(props: EmployeeCenterProps) {
  const employees = props.employees ?? [];
  const enabled = employees.filter((emp) => emp.enabled !== false).length;
  const withSkills = employees.filter((emp) => (emp.skillNames?.length ?? emp.skillIds?.length ?? 0) > 0).length;
  const withTools = employees.filter((emp) => (emp.mcpServerKeys?.length ?? 0) > 0).length;

  return html`
    <main class="ops-dashboard">
      <div class="ops-dashboard-header">
        <div>
          <h1>数字员工中心</h1>
          <p class="muted">
            将 Agent 产品化为有岗位职责、技能工具、任务产出、协同编排和效能评估的数字员工体系。
          </p>
        </div>
        <div class="ops-dashboard-actions__inner">
          <button class="ops-dashboard-actions__btn" type="button" @click=${props.onOpenMarket}>
            ${icons.users} 员工市场
          </button>
          <button class="ops-dashboard-actions__btn" type="button" @click=${props.onCreateEmployee}>
            ${icons.plus} 新建员工
          </button>
        </div>
      </div>

      <section class="stats-grid">
        <article class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.users}</div>
          <div class="stat-content">
            <h3>员工资产</h3>
            <div class="stat-value">${props.loading ? "..." : employees.length}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--ok">${icons.overviewGrid}</div>
          <div class="stat-content">
            <h3>启用员工</h3>
            <div class="stat-value">${props.loading ? "..." : enabled}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--warn">${icons.zap}</div>
          <div class="stat-content">
            <h3>配置技能</h3>
            <div class="stat-value">${props.loading ? "..." : withSkills}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--danger">${icons.wrench}</div>
          <div class="stat-content">
            <h3>配置工具</h3>
            <div class="stat-value">${props.loading ? "..." : withTools}</div>
          </div>
        </article>
      </section>

      <section class="domain-status-section">
        <div class="section-title">
          <span class="section-title__icon">${icons.layout}</span>
          <span>中心工作台</span>
        </div>
        <div class="domain-grid">
          ${renderActionCard("员工市场", "发现、安装和上架面向运维岗位的数字员工。", "users", props.onOpenMarket)}
          ${renderActionCard("我的员工", "管理已安装和自建员工，维护 Prompt、Skill、MCP 和启停状态。", "server", props.onOpenEmployees)}
          ${renderActionCard("员工编排", "组织多个数字员工协同处理告警、巡检、诊断、治理和变更任务。", "messageSquare", props.onOpenSwarm)}
          ${renderActionCard("任务记录", "承载告警分析、巡检、诊断、治理、容量预测和变更护航任务。", "historyClock", props.onOpenTasks)}
          ${renderStaticCard("权限与审计", "后续管理只读/执行边界、工具调用审批、执行日志和安全策略。", "sandbox")}
          ${renderActionCard("效能评估", "量化任务数、自动闭环率、诊断采纳率、节省人时和成本消耗。", "usageBars", props.onOpenEffectiveness)}
        </div>
      </section>

      <section class="domain-status-section">
        <div class="section-title">
          <span class="section-title__icon">${icons.users}</span>
          <span>岗位体系</span>
        </div>
        <div class="domain-grid">
          ${ROLE_CARDS.map((role) => html`
            <article class="domain-card">
              <div class="domain-card-header">
                <div class="domain-icon-wrapper">${icons.users}</div>
                <div>
                  <div class="domain-name">${role.title}</div>
                  <div class="domain-card-hint">${role.desc}</div>
                  <div class="muted" style="margin-top: 8px; font-size: 12px;">对应能力域：${role.domain}</div>
                </div>
              </div>
            </article>
          `)}
        </div>
      </section>
    </main>
  `;
}

function renderActionCard(title: string, desc: string, iconName: keyof typeof icons, onClick: () => void) {
  return html`
    <button class="domain-card domain-card-link" type="button" @click=${onClick}>
      <div class="domain-card-header">
        <div class="domain-icon-wrapper">${icons[iconName] ?? icons.folder}</div>
        <div>
          <div class="domain-name">${title}</div>
          <div class="domain-card-hint">${desc}</div>
        </div>
      </div>
    </button>
  `;
}

function renderStaticCard(title: string, desc: string, iconName: keyof typeof icons) {
  return html`
    <article class="domain-card domain-card--muted">
      <div class="domain-card-header">
        <div class="domain-icon-wrapper">${icons[iconName] ?? icons.folder}</div>
        <div>
          <div class="domain-name">${title}</div>
          <div class="domain-card-hint">${desc}</div>
        </div>
      </div>
    </article>
  `;
}
