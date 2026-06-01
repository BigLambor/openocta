# OpenOcta 改造任务跟踪表

> **主路线图**（优先级、Cursor 风格、验收标准）：见 [ops-roadmap.md](./ops-roadmap.md)  
> 本文件保留「阶段」视图，并与路线图 ID（P0-x、P1-x…）对齐。

---

## 阶段一：UI 重构、技术域页面及二级 Tab 骨架

- [x] 全局 UI 视觉与 `ui/src/styles/` 主题
- [x] 重构 `navigation.ts`：一级技术域 + 二级场景 Tab
- [x] 各大技术域 views 骨架（BCH / FI / GBase / 治理 / 数据 App）
- [x] 前端渲染验证
- [x] **补充（路线图 P3）**：ops-tokens、ui-style-guide.md、通用空状态组件

---

## 阶段二：RBAC 与企业 IM 通道

- [x] 后端 SQLite RBAC + 迁移
- [x] JWT 登录 / 网关鉴权中间件
- [x] 前端登录、用户/角色/权限管理
- [x] 菜单 `menu:*` 权限过滤
- [x] 企业 IM 凭证配置后台
- [x] **补充（路线图 P4-2）**：ops:inspect / ops:ack 前后端一致

---

## 阶段三：深度巡检（VictoriaMetrics）

- [x] 五大技术域默认 Cron 任务 + PromQL 提示词（`cron/service.go`）
- [x] `query_vm_metrics` 真实 PromQL 查询
- [x] 前端 Cron 历史 + 一键手动巡检 + 轮询（`controllers/ops-inspection-run.ts`）
- [/] 健康得分 &lt; 85 与 IM 推送（`cron_delivery.go`，依赖通道已启用）
- [x] **P2-A1** `query_gbase_slow_sql` 真实 DSN（`GBASE_DSN`）
- [x] **P2-A2** `query_governance_lineage` 真实 API（`GOVERNANCE_API_URL`）
- [x] **P2-A3** 巡检报告 Cursor 式 UI 统一
- [x] **P2-A4** Hadoop JMX / FI Manager 专用 tool
- [x] **P2-A5** 巡检 &lt;85 IM 推送可观测（im-status + UI 提示）
- [x] **P1-4** 集群上下文注入 Agent / 巡检 Prompt

---

## 阶段四：告警降噪、ChatOps、集成交付

- [x] **后端** `POST /hooks/alert` 滑动窗口合并 + Agent 分析触发（`hooks.go`）
- [x] **P2-B1–B4** 告警持久化 API + 前端告警 Tab
- [x] **P2-B5** Alert Studio：保持域内子 Tab（见 ops-api.md）
- [x] **P2-C2** ChatOps `/help` `/ack` `/diagnose`
- [x] **P4** E2E 清单（e2e-ops-smoke.md）与按钮 RBAC；Vitest 单元冒烟已补

---

## 当前冲刺

- [x] **P0**（2026-06-01）：路由、去 Mock、空状态、按域实体选择器、按钮文案对齐
- [x] **P1**：P1-1～P1-6 已完成 — 见 [ops-roadmap.md](./ops-roadmap.md)、[ops-api.md](./ops-api.md)
- [x] **P2**：核心闭环已完成（含 ChatOps、IM 卡片）
- [x] **P3**：顶栏下拉样式迁入 layout.css；其余 P3 已落地
- [/] **P4-4 自动化**：`ops/rbac.test.ts`、`ops/deeplink.test.ts`、`ops/ops-paths.browser.test.ts`；浏览器全链路 E2E 待补
- [x] **P1-2 CMDB**：`POST /api/ops/clusters/sync-cmdb` + 资产页「同步 CMDB」

---

## 状态图例

| 标记 | 含义 |
|------|------|
| `[x]` | 已完成 |
| `[/]` | 进行中 |
| `[ ]` | 未开始 |

更新任务时请同时勾选 [ops-roadmap.md](./ops-roadmap.md) 中对应 P0–P4 条目。
