import { html } from "lit";
import { icons } from "../icons.ts";
import type { DigitalEmployee } from "./digital-employee.ts";

export type AutomationHubProps = {
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
    title: "值班运维助手",
    desc: "负责告警接收、降噪、聚合、初判和升级通知。",
    domain: "可观测与告警",
  },
  {
    title: "巡检助手",
    desc: "负责日常巡检、深度巡检、健康评分和风险清单。",
    domain: "健康度与巡检",
  },
  {
    title: "诊断助手",
    desc: "负责故障定位、根因分析、影响面分析和处置建议。",
    domain: "故障诊断与应急",
  },
  {
    title: "治理助手",
    desc: "负责重复告警、作业稳定性、配置偏差和 SLA 风险治理。",
    domain: "治理与优化",
  },
  {
    title: "容量与成本助手",
    desc: "负责容量预测、资源利用率、性能瓶颈和成本归因。",
    domain: "容量性能与成本",
  },
  {
    title: "变更护航助手",
    desc: "负责变更前评估、变更中观测、变更后验证和回滚建议。",
    domain: "变更配置与合规",
  },
];

export function renderAutomationHub(props: AutomationHubProps) {
  const employees = props.employees ?? [];
  const enabled = employees.filter((emp) => emp.enabled !== false).length;
  const withSkills = employees.filter((emp) => (emp.skillNames?.length ?? emp.skillIds?.length ?? 0) > 0).length;
  const withTools = employees.filter((emp) => (emp.mcpServerKeys?.length ?? 0) > 0).length;

  return html`
    <main class="ops-dashboard">
      <div class="ops-dashboard-header">
        <div>
          <h1>自动化配置</h1>
          <p class="muted">
            管理数字员工模板、工作流编排、触发规则和执行记录
          </p>
        </div>
        <div class="ops-dashboard-actions__inner">
          <button class="ops-dashboard-actions__btn" type="button" @click=${props.onOpenMarket}>
            <span class="ops-dashboard-actions__icon" aria-hidden="true">${icons.users}</span>
            <span>助手模板库</span>
          </button>
          <button class="ops-dashboard-actions__btn" type="button" @click=${props.onCreateEmployee}>
            <span class="ops-dashboard-actions__icon" aria-hidden="true">${icons.plus}</span>
            <span>创建助手</span>
          </button>
        </div>
      </div>

      <section class="stats-grid">
        <article class="stat-card">
          <div class="stat-icon stat-icon--blue">${icons.users}</div>
          <div class="stat-content">
            <h3>助手模板</h3>
            <div class="stat-value">${props.loading ? "..." : employees.length}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--ok">${icons.overviewGrid}</div>
          <div class="stat-content">
            <h3>已启用</h3>
            <div class="stat-value">${props.loading ? "..." : enabled}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--warn">${icons.zap}</div>
          <div class="stat-content">
            <h3>关联技能</h3>
            <div class="stat-value">${props.loading ? "..." : withSkills}</div>
          </div>
        </article>
        <article class="stat-card">
          <div class="stat-icon stat-icon--danger">${icons.wrench}</div>
          <div class="stat-content">
            <h3>关联工具</h3>
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
          ${renderActionCard("助手模板库", "发现、安装和上架面向运维场景的数字员工模板。", "users", props.onOpenMarket)}
          ${renderActionCard("我的助手", "管理已安装和自建数字员工，配置 Prompt、技能和工具。", "server", props.onOpenEmployees)}
          ${renderActionCard("工作流编排", "编排自动化步骤、触发条件、审批节点和工具调用。", "messageSquare", props.onOpenSwarm)}
          ${renderActionCard("执行记录", "数字员工处理告警、巡检、诊断、治理等场景的执行记录。", "historyClock", props.onOpenTasks)}
          ${renderStaticCard("权限与审计", "后续管理只读/执行边界、工具调用审批、执行日志和安全策略。", "sandbox")}
          ${renderActionCard("自动化效果", "量化自动化执行效果：任务量、成功率、闭环率和成本。", "usageBars", props.onOpenEffectiveness)}
        </div>
      </section>

      <section class="domain-status-section">
        <div class="section-title">
          <span class="section-title__icon">${icons.users}</span>
          <span>专家角色模板</span>
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
