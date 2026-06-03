import { LitElement, html, css, nothing } from "lit";
import { customElement, property, state } from "lit/decorators.js";
import { fetchBchEmployees, BchEmployee } from "../../controllers/bch-client.ts";

@customElement("bch-employee-workstation")
export class BchEmployeeWorkstation extends LitElement {
  @property({ type: Object }) host: any = null;

  @state() private employees: BchEmployee[] = [];
  @state() private loading = false;
  @state() private error: string | null = null;

  static styles = css`
    :host {
      display: block;
      padding: 24px;
      font-family: var(--font-family, sans-serif);
      color: var(--text-primary);
      overflow-y: auto;
      height: 100%;
      box-sizing: border-box;
    }

    .ws-header {
      margin-bottom: 24px;
    }

    .ws-header h2 {
      margin: 0 0 6px 0;
      font-size: 18px;
      font-weight: 600;
    }

    .ws-header p {
      margin: 0;
      font-size: 13px;
      color: var(--text-muted);
    }

    .employee-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
      gap: 20px;
    }

    .employee-card {
      background: var(--bg-content);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 20px;
      box-shadow: var(--shadow-sm, 0 4px 20px rgba(0, 0, 0, 0.08));
      display: flex;
      flex-direction: column;
      gap: 16px;
      transition: all 0.2s;
    }

    .employee-card:hover {
      border-color: var(--accent, #3b82f6);
      transform: translateY(-2px);
      box-shadow: var(--shadow-md, 0 6px 24px rgba(0, 0, 0, 0.12));
    }

    .emp-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      border-bottom: 1px solid var(--border);
      padding-bottom: 12px;
    }

    .emp-meta {
      display: flex;
      gap: 12px;
      align-items: center;
    }

    .emp-avatar {
      width: 42px;
      height: 42px;
      border-radius: 8px;
      background: rgba(59, 130, 246, 0.1);
      border: 1px solid rgba(59, 130, 246, 0.3);
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 20px;
    }

    .emp-title {
      font-size: 15px;
      font-weight: 600;
    }

    .status-badge {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 10px;
      color: var(--text-muted);
      margin-top: 2px;
    }

    .status-dot {
      width: 6px;
      height: 6px;
      border-radius: 50%;
      position: relative;
    }

    .status-dot.idle {
      background: #10b981;
    }

    .status-dot.working {
      background: #3b82f6;
    }

    .status-dot.pulse::after {
      content: "";
      position: absolute;
      inset: -2px;
      border-radius: 50%;
      animation: pulse 1.5s infinite;
    }

    .status-dot.idle.pulse::after {
      border: 1px solid #10b981;
    }

    .status-dot.working.pulse::after {
      border: 1px solid #3b82f6;
    }

    @keyframes pulse {
      0% { transform: scale(1); opacity: 1; }
      100% { transform: scale(2.2); opacity: 0; }
    }

    .emp-desc {
      font-size: 12px;
      color: var(--text-secondary);
      line-height: 1.5;
    }

    .tags-section-header {
      font-size: 10px;
      font-weight: 600;
      text-transform: uppercase;
      letter-spacing: 0.5px;
      color: var(--text-muted);
      margin-bottom: 6px;
    }

    .skills-box, .tools-box {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
    }

    .skill-pill {
      background: rgba(16, 185, 129, 0.08);
      border: 1px solid rgba(16, 185, 129, 0.15);
      color: #10b981;
      font-size: 10px;
      padding: 2px 6px;
      border-radius: 4px;
    }

    .tool-pill {
      background: var(--bg, rgba(0, 0, 0, 0.04));
      border: 1px solid var(--border);
      color: var(--text-secondary);
      font-family: monospace;
      font-size: 10px;
      padding: 2px 6px;
      border-radius: 4px;
    }

    .recent-tasks-panel {
      border-top: 1px solid var(--border);
      padding-top: 12px;
      font-size: 11px;
    }

    .task-timeline {
      display: flex;
      flex-direction: column;
      gap: 10px;
      margin-top: 8px;
    }

    .task-timeline-item {
      display: flex;
      gap: 8px;
    }

    .task-time {
      color: var(--text-muted);
      width: 50px;
      flex-shrink: 0;
    }

    .task-content {
      flex: 1;
    }

    .task-title {
      font-weight: 600;
      color: var(--text-primary);
    }

    .task-res {
      color: var(--text-muted);
      margin-top: 2px;
    }

    .loading-container {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 200px;
      color: var(--text-muted);
      gap: 12px;
      font-size: 13px;
    }

    .spinner {
      width: 24px;
      height: 24px;
      border: 2px solid rgba(255, 255, 255, 0.1);
      border-top-color: var(--accent, #3b82f6);
      border-radius: 50%;
      animation: spin 0.8s linear infinite;
    }

    @keyframes spin {
      to {
        transform: rotate(360deg);
      }
    }
  `;

  connectedCallback() {
    super.connectedCallback();
    this.loadData();
  }

  async loadData() {
    if (!this.host) return;
    this.loading = true;
    this.error = null;
    try {
      this.employees = await fetchBchEmployees(this.host);
    } catch (err: any) {
      this.error = err.message || String(err);
    } finally {
      this.loading = false;
    }
  }

  render() {
    if (this.loading) {
      return html`
        <div class="loading-container">
          <div class="spinner"></div>
          <div>正在加载数字员工状态...</div>
        </div>
      `;
    }

    if (this.error) {
      return html`
        <div style="padding: 16px; color: #ef4444;">
          ${this.error}
        </div>
      `;
    }

    return html`
      <div class="ws-header">
        <h2>BCH 数字员工工作站</h2>
        <p>本技术域已部署 3 名专属大数据运维智能体，协同进行自动化值班、巡检与故障诊断。</p>
      </div>

      <div class="employee-grid">
        ${this.employees.map((emp) => {
          const isWorking = emp.status === "working";
          const avatar = emp.id === "emp_bch_inspect" ? "🔍" : emp.id === "emp_bch_diagnose" ? "🩺" : "🛡️";

          return html`
            <div class="employee-card">
              <div class="emp-header">
                <div class="emp-meta">
                  <div class="emp-avatar">${avatar}</div>
                  <div>
                    <div class="emp-title">${emp.name}</div>
                    <div class="status-badge">
                      <span class="status-dot ${emp.status} pulse"></span>
                      <span>${emp.statusDesc}</span>
                    </div>
                  </div>
                </div>
              </div>

              <div class="emp-desc">${emp.description}</div>

              <div>
                <div class="tags-section-header">掌握的 SOP 运维技能 (Skills)</div>
                <div class="skills-box">
                  ${emp.skills.map((skill) => html`<span class="skill-pill">${skill}</span>`)}
                </div>
              </div>

              <div>
                <div class="tags-section-header">授权调用的 MCP 工具 (Tools)</div>
                <div class="tools-box">
                  ${emp.tools.map((tool) => html`<span class="tool-pill">${tool}</span>`)}
                </div>
              </div>

              <div class="recent-tasks-panel">
                <div class="tags-section-header">最近工作历史成果流水</div>
                <div class="task-timeline">
                  ${emp.recentTasks.map((t) => html`
                    <div class="task-timeline-item">
                      <div class="task-time">${t.time}</div>
                      <div class="task-content">
                        <div class="task-title">${t.task}</div>
                        <div class="task-res">${t.result}</div>
                      </div>
                    </div>
                  `)}
                </div>
              </div>
            </div>
          `;
        })}
      </div>
    `;
  }
}
