import { html } from "lit";
import { icons } from "../icons.ts";

const CAPABILITIES = [
  {
    title: "资产与拓扑中心",
    desc: "跨技术域统一资产、依赖、拓扑、责任人和配置基线。",
    icon: "server",
  },
  {
    title: "可观测中心",
    desc: "汇聚指标、日志、事件和链路，为告警与诊断提供统一信号。",
    icon: "overviewGrid",
  },
  {
    title: "告警事件中心",
    desc: "承载告警接入、聚合、降噪、影响分析和事件闭环。",
    icon: "zap",
  },
  {
    title: "巡检中心",
    desc: "管理巡检模板、巡检计划、执行历史、风险清单和巡检报告。",
    icon: "historyClock",
  },
  {
    title: "诊断中心",
    desc: "统一故障诊断任务、根因分析、影响面分析和应急建议。",
    icon: "messageSquare",
  },
  {
    title: "治理中心",
    desc: "跟踪重复告警、稳定性问题、配置偏差和长期优化事项。",
    icon: "layout",
  },
  {
    title: "容量性能中心",
    desc: "沉淀容量预测、性能瓶颈、资源利用率和成本归因能力。",
    icon: "usageBars",
  },
  {
    title: "变更护航中心",
    desc: "覆盖变更前评估、变更中观测、变更后验证和复盘。",
    icon: "settings",
  },
  {
    title: "自动化处置中心",
    desc: "管理 Runbook、脚本、审批策略、执行记录和安全边界。",
    icon: "wrench",
  },
];

export function renderOpsCapabilityCenter() {
  return html`
    <main class="ops-dashboard">
      <div class="ops-dashboard-header">
        <div>
          <h1>运维能力中心</h1>
          <p class="muted">
            从能力视角组织跨技术域的资产、观测、告警、巡检、诊断、治理、容量、变更与自动化。
          </p>
        </div>
      </div>

      <section class="domain-status-section">
        <div class="section-title">
          <span class="section-title__icon">${icons.overviewGrid}</span>
          <span>平台能力域</span>
        </div>
        <div class="domain-grid">
          ${CAPABILITIES.map((item) => {
            const iconName = item.icon as keyof typeof icons;
            return html`
              <article class="domain-card">
                <div class="domain-card-header">
                  <div class="domain-icon-wrapper">${icons[iconName] ?? icons.folder}</div>
                  <div>
                    <div class="domain-name">${item.title}</div>
                    <div class="domain-card-hint">${item.desc}</div>
                  </div>
                </div>
              </article>
            `;
          })}
        </div>
      </section>
    </main>
  `;
}
