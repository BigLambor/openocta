import { html, nothing } from "lit";
import { icons } from "../icons.ts";
import { renderChat, type ChatProps } from "./chat.ts";

export type TechOpsDomainProps = {
  domainKey: "hadoop" | "fi" | "gbase" | "governance" | "dataapps";
  domainName: string;
  activeSubTab: "agent" | "alerts" | "inspections";
  onSubTabChange: (tab: "agent" | "alerts" | "inspections") => void;
  // Chat Props
  chatProps: ChatProps;
  // Alert Props
  alertsLoading: boolean;
  alertGroups: Array<{
    id: string;
    title: string;
    severity: "critical" | "warning" | "info";
    timestamp: string;
    originalCount: number;
    reducedTo: number;
    rootCause: string;
    impact: string;
    status: "resolved" | "active";
  }>;
  selectedAlertGroupId: string | null;
  onSelectAlertGroup: (id: string) => void;
  // Inspection Props
  inspectionsLoading: boolean;
  inspections: Array<{
    id: string;
    time: string;
    score: number;
    status: "healthy" | "warning" | "critical";
    reportSummary: string;
    reportMarkdown: string;
  }>;
  selectedInspectionId: string | null;
  onSelectInspection: (id: string) => void;
  onRunInspection: () => void;
  isInspecting: boolean;
};

export function renderTechOpsDomain(props: TechOpsDomainProps) {
  return html`
    <div class="ops-domain-container">
      <!-- 页面顶部横幅和二级 Tab 开关 -->
      <div class="ops-header">
        <div class="ops-header__title-area">
          <div class="ops-header__title">
            <span class="ops-header__icon">${icons[props.domainKey === "hadoop" ? "overviewGrid" : props.domainKey === "fi" ? "brain" : props.domainKey === "gbase" ? "monitor" : props.domainKey === "governance" ? "scrollText" : "folder"]}</span>
            <h2>${props.domainName} 智能运维中心</h2>
          </div>
          <div class="ops-header__desc">针对 ${props.domainName} 提供全方位的智能诊断、告警风暴抑制与深度周期巡检。</div>
        </div>

        <div class="ops-tabs">
          <button 
            class="ops-tab ${props.activeSubTab === "agent" ? "ops-tab--active" : ""}" 
            type="button"
            @click=${() => props.onSubTabChange("agent")}
          >
            ${icons.messageSquare} 智能诊断 Agent
          </button>
          <button 
            class="ops-tab ${props.activeSubTab === "alerts" ? "ops-tab--active" : ""}" 
            type="button"
            @click=${() => props.onSubTabChange("alerts")}
          >
            ${icons.zap} 告警降噪与影响评估
          </button>
          <button 
            class="ops-tab ${props.activeSubTab === "inspections" ? "ops-tab--active" : ""}" 
            type="button"
            @click=${() => props.onSubTabChange("inspections")}
          >
            ${icons.historyClock} 深度健康巡检
          </button>
        </div>
      </div>

      <!-- 页面主要内容区分发 -->
      <div class="ops-content">
        ${props.activeSubTab === "agent"
          ? html`
              <div class="ops-agent-view">
                ${renderChat(props.chatProps)}
              </div>
            `
          : props.activeSubTab === "alerts"
          ? renderAlertsSubTab(props)
          : renderInspectionsSubTab(props)}
      </div>
    </div>
  `;
}

// 告警降噪评估子 Tab
function renderAlertsSubTab(props: TechOpsDomainProps) {
  const activeGroup = props.alertGroups.find(g => g.id === props.selectedAlertGroupId) || props.alertGroups[0];
  const originalTotal = props.alertGroups.reduce((acc, g) => acc + g.originalCount, 0);
  const reducedTotal = props.alertGroups.length;
  const reductionRate = originalTotal > 0 ? ((1 - reducedTotal / originalTotal) * 100).toFixed(1) : "0.0";

  return html`
    <div class="ops-alerts-grid">
      <!-- 顶部统计栏 -->
      <div class="ops-summary-cards">
        <div class="ops-card stat-card">
          <div class="stat-label">未合并原始告警</div>
          <div class="stat-value warning">${originalTotal}</div>
          <div class="muted">最近 24 小时内的总报警事件数</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">AI 智能降噪比</div>
          <div class="stat-value ok">${reductionRate}%</div>
          <div class="muted">通过滑动窗口及拓扑关联算法合并</div>
        </div>
        <div class="ops-card stat-card">
          <div class="stat-label">已合并告警组</div>
          <div class="stat-value info">${reducedTotal}</div>
          <div class="muted">已派发 SRE 团队处理的核心故障组</div>
        </div>
      </div>

      <!-- 下方列表与分析两栏布局 -->
      <div class="ops-main-columns">
        <!-- 左侧：告警组列表 -->
        <div class="ops-card list-column">
          <div class="column-header">已合并告警列表</div>
          ${props.alertsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : props.alertGroups.length === 0
            ? html`<div class="empty-placeholder">暂无活动告警，系统运行平稳。</div>`
            : html`
                <div class="alert-list">
                  ${props.alertGroups.map(
                    g => html`
                      <div 
                        class="alert-item ${g.id === props.selectedAlertGroupId ? "alert-item--active" : ""} alert-item--${g.severity}"
                        @click=${() => props.onSelectAlertGroup(g.id)}
                      >
                        <div class="alert-item__meta">
                          <span class="alert-badge alert-badge--${g.severity}">
                            ${g.severity === "critical" ? "严重" : g.severity === "warning" ? "警告" : "通知"}
                          </span>
                          <span class="alert-time">${g.timestamp}</span>
                        </div>
                        <div class="alert-item__title">${g.title}</div>
                        <div class="alert-item__noise">
                          <span>原始告警: <strong>${g.originalCount}</strong> 次</span>
                          <span class="divider">|</span>
                          <span class="rate">降噪比: <strong>${((1 - g.reducedTo / g.originalCount) * 100).toFixed(0)}%</strong></span>
                        </div>
                      </div>
                    `
                  )}
                </div>
              `}
        </div>

        <!-- 右侧：AI 诊断详情 -->
        <div class="ops-card detail-column">
          <div class="column-header">AI 智能根因与影响评估</div>
          ${!activeGroup
            ? html`<div class="empty-placeholder">请从左侧选择一个告警组以查看诊断报告。</div>`
            : html`
                <div class="alert-detail">
                  <div class="detail-section">
                    <div class="detail-section__title">${activeGroup.title}</div>
                    <div class="detail-meta">
                      <span class="detail-time">生成时间: ${activeGroup.timestamp}</span>
                      <span class="detail-count">关联原始事件: ${activeGroup.originalCount} 次</span>
                    </div>
                  </div>

                  <div class="detail-section">
                    <div class="detail-section__header">${icons.zap} 根因推导 (Root Cause Analysis)</div>
                    <div class="detail-section__content highlight">
                      ${activeGroup.rootCause}
                    </div>
                  </div>

                  <div class="detail-section">
                    <div class="detail-section__header">${icons.overviewGrid} 业务受损评估 (Impact Scope)</div>
                    <div class="detail-section__content">
                      ${activeGroup.impact}
                    </div>
                  </div>

                  <div class="detail-section">
                    <div class="detail-section__header">${icons.scrollText} 智能降噪与处置建议</div>
                    <div class="detail-section__content">
                      <p>1. <strong>告警合并机制</strong>：该事件中包含的 ${activeGroup.originalCount - 1} 条重复告警已自动静默抑制，避免告警风暴。</p>
                      <p>2. <strong>排查优先级</strong>：请优先登录相关物理节点检查监控指标；若网络阻塞，建议启动备用路由并重启组件服务。</p>
                      <p>3. <strong>ChatOps 联动</strong>：您可以在“智能诊断 Agent”中 @机器人 快速执行指令诊断（如输入“<code>查看 ${props.domainName} 活跃死锁状态</code>”）。</p>
                    </div>
                  </div>
                </div>
              `}
        </div>
      </div>
    </div>
  `;
}

// 深度健康巡检子 Tab
function renderInspectionsSubTab(props: TechOpsDomainProps) {
  const activeInspection = props.inspections.find(ins => ins.id === props.selectedInspectionId) || props.inspections[0];
  const lastScore = props.inspections[0]?.score ?? 100;
  const statusClass = lastScore >= 90 ? "ok" : lastScore >= 75 ? "warning" : "danger";

  return html`
    <div class="ops-inspections-grid">
      <!-- 顶部控制面板 -->
      <div class="ops-inspection-ctrl ops-card">
        <div class="ctrl-left">
          <div class="health-index-badge health-index-badge--${statusClass}">
            <div class="health-score">${lastScore}</div>
            <div class="health-label">健康度得分</div>
          </div>
          <div class="ctrl-meta">
            <h3>巡检自动化配置</h3>
            <p>定时策略：每天 08:00 和 20:00 自动触发深度健康巡检。</p>
            <p>最近一次执行：${props.inspections[0]?.time ?? "暂无执行记录"}</p>
          </div>
        </div>
        <div class="ctrl-right">
          <button 
            class="btn primary large ${props.isInspecting ? "btn--loading" : ""}" 
            type="button" 
            ?disabled=${props.isInspecting}
            @click=${props.onRunInspection}
          >
            ${props.isInspecting ? html`${icons.loader} 正在进行深度巡检...` : html`${icons.zap} 一键手动巡检`}
          </button>
        </div>
      </div>

      <!-- 下方历史报告列表与详情 -->
      <div class="ops-main-columns">
        <!-- 左侧：历史巡检报告列表 -->
        <div class="ops-card list-column">
          <div class="column-header">历史巡检报告</div>
          ${props.inspectionsLoading
            ? html`<div class="loading-placeholder">${icons.loader} 加载中...</div>`
            : props.inspections.length === 0
            ? html`<div class="empty-placeholder">暂无巡检记录，请点击上方按钮开始首次巡检。</div>`
            : html`
                <div class="inspection-list">
                  ${props.inspections.map(
                    ins => html`
                      <div 
                        class="inspection-item ${ins.id === props.selectedInspectionId ? "inspection-item--active" : ""}"
                        @click=${() => props.onSelectInspection(ins.id)}
                      >
                        <div class="inspection-item__meta">
                          <span class="score-badge score-badge--${ins.score >= 90 ? "ok" : ins.score >= 75 ? "warning" : "danger"}">
                            ${ins.score}分
                          </span>
                          <span class="inspection-time">${ins.time}</span>
                        </div>
                        <div class="inspection-summary">${ins.reportSummary}</div>
                      </div>
                    `
                  )}
                </div>
              `}
        </div>

        <!-- 右侧：报告详情 -->
        <div class="ops-card detail-column">
          <div class="column-header">健康巡检报告详情</div>
          ${!activeInspection
            ? html`<div class="empty-placeholder">请从左侧选择一份巡检报告以查看完整指标。</div>`
            : html`
                <div class="inspection-detail-report">
                  <div class="report-header">
                    <h3>${props.domainName} 巡检报告 (${activeInspection.time})</h3>
                    <span class="report-score score-badge--${activeInspection.score >= 90 ? "ok" : activeInspection.score >= 75 ? "warning" : "danger"}">
                      健康得分：${activeInspection.score} / 100
                    </span>
                  </div>
                  <div class="report-body">
                    <!-- 使用简单的段落渲染巡检总结与指标 -->
                    <div class="report-section">
                      <div class="report-section__title">🔍 巡检发现摘要</div>
                      <p>${activeInspection.reportSummary}</p>
                    </div>

                    <div class="report-section">
                      <div class="report-section__title">📊 监控指标明细 (数据源: VictoriaMetrics)</div>
                      <table class="ops-metrics-table">
                        <thead>
                          <tr>
                            <th>监控指标项</th>
                            <th>状态</th>
                            <th>当前观测值</th>
                            <th>标准阈值</th>
                          </tr>
                        </thead>
                        <tbody>
                          <tr>
                            <td>系统 CPU 平均负载</td>
                            <td><span class="indicator indicator--ok">● 正常</span></td>
                            <td>34.2%</td>
                            <td>&lt; 80%</td>
                          </tr>
                          <tr>
                            <td>系统 内存空闲余量</td>
                            <td><span class="indicator indicator--ok">● 正常</span></td>
                            <td>64 GB</td>
                            <td>&gt; 8 GB</td>
                          </tr>
                          <tr>
                            <td>系统 磁盘 I/O 队列深度</td>
                            <td><span class="indicator indicator--${activeInspection.score < 90 ? "warning" : "ok"}">
                              ● ${activeInspection.score < 90 ? "偏高" : "正常"}
                            </span></td>
                            <td>${activeInspection.score < 90 ? "4.2" : "0.5"}</td>
                            <td>&lt; 2.0</td>
                          </tr>
                          <tr>
                            <td>关键 服务响应时延</td>
                            <td><span class="indicator indicator--ok">● 正常</span></td>
                            <td>12 ms</td>
                            <td>&lt; 200 ms</td>
                          </tr>
                        </tbody>
                      </table>
                    </div>

                    <div class="report-section">
                      <div class="report-section__title">💡 AI 优化建议</div>
                      <pre class="report-markdown">${activeInspection.reportMarkdown}</pre>
                    </div>
                  </div>
                </div>
              `}
        </div>
      </div>
    </div>
  `;
}
