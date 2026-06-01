# OpenOcta 智能运维改造路线图

> 依据 [docs/1.md](./1.md) 战略动机、[implementation_plan.md](./implementation_plan.md) 30 天计划与当前代码现状整理。  
> **视觉与交互基调**：参考 [Cursor Dashboard](https://cursor.com/cn/dashboard) — 克制、工具型、留白充足、低噪声，避免传统「运维大屏」炫技风格。

**进度勾选**：将 `[ ]` 改为 `[x]` 表示完成；`[/]` 表示进行中。

---

## 设计基调：Cursor 风格（全阶段适用）

在改页面前先统一规则，避免反复返工。

| 维度 | Cursor 气质 | OpenOcta 落地要求 |
|------|-------------|------------------|
| 布局 | 顶栏极简 + 主内容区宽、留白足 | 保留「顶栏域切换 + 左侧场景」，减少描边与阴影层数 |
| 色彩 | 中性灰底 + 单一强调色（低饱和） | 收敛渐变/霓虹；健康态用绿/黄/红 **小标签**，不用大面积色块 |
| 字体 | 清晰层级，标题不夸张 | 页面主标题 20–24px；指标数字可略大，副文案 12–13px 灰色 |
| 组件 | 细边框卡片、轻 hover | 卡片 `1px` 边框 + 8–12px 圆角；hover 仅背景微变 |
| 导航 | 设置/账号收拢，主路径突出 | 系统能力收在 ⚙️ 下拉；「AI 运维助手」与业务域 Tab 同级清晰 |
| 数据 | 空状态、加载、错误明确 | **禁止假数据当真相**：无数据时展示「暂无 / 未配置」 |
| 动效 | 几乎静态，过渡极短 | 列表切换 ≤150ms；避免大面积入场动画 |

**设计交付物**

- [x] `ui/src/styles/ops-tokens.css`（或扩展现有 design tokens）
- [x] `docs/ui-style-guide.md`（色板、间距、卡片、空状态文案）

---

## P0 — 必修：正确性 + 诚实 UI（预估 1–3 天）

> 不修这些，后续接真数据也会被路由 Bug 和 Mock 误导。

| ID | 任务 | 说明 | 验收标准 |
|----|------|------|----------|
| P0-1 | 修复「集群资产管理」路由 | 下拉项 `tab: "config"` → `assetManagement`；去重两个「系统配置」 | 点击后进入 `/asset-management`，渲染 `renderAssetManagement()` |
| P0-2 | 清理误导性 Mock | 删除未使用的 `getMockInspections()`；告警 Tab 演示数据或空状态 | 无死函数；用户不把 Mock 当生产告警 |
| P0-3 | 无数据诚实展示 | 大屏、资产、告警：空状态 + 主 CTA | 未配置 VM/CMDB 时不显示硬编码「12 集群 / 85 分」 |
| P0-4 | 实体选择器域隔离（最小版） | 各 `domainKey` 独立选项配置（常量/JSON，后接 API） | FI/GBase 不再出现 BCH 集群名 |
| P0-5 | 文案与能力对齐 | 告警文案与后端能力一致；无 handler 的按钮 disabled + tooltip | 不宣称未实现的「拓扑合并已生效」 |

**主要文件**：`ui/src/ui/app-render.ts`、`views/asset-management.ts`、`views/overview.ts`、`views/tech-ops-domain.ts`、`navigation.ts`

### P0 勾选清单

- [x] P0-1 集群资产管理路由
- [x] P0-2 清理 Mock / 死代码
- [x] P0-3 诚实空状态（大屏 / 资产 / 告警）
- [x] P0-4 实体选择器按域隔离
- [x] P0-5 文案与按钮能力对齐

---

## P1 — 数据契约：资产 / 集群 / 上下文（预估 ~1 周）

> 建立单一数据源，Dashboard 才有真指标可挂。

| ID | 任务 | 说明 | 验收标准 |
|----|------|------|----------|
| P1-1 | 集群资产领域模型 | 后端表或 store：id、name、domain、region、components、owner、status | `GET/POST/PATCH /api/ops/clusters` |
| P1-2 | 资产页接 API | 表格来自 API；「同步 CMDB」可先为配置导入或 webhook stub | 列表可刷新；按钮可点击 |
| P1-3 | 实体选择器接 API | 按 `domain` 过滤；支持全域 / 集群 / 组件（组件可二期） | `opsSelectedEntityIds` 结构稳定 |
| P1-4 | 上下文贯穿 Agent | Prompt 注入 domain + clusterId + component | 换集群后 Agent 能体现环境 |
| P1-5 | 运维大屏 v2（真数据） | 纳管数、健康/异常、待处理告警 ← API；**5 个业务域**卡片齐全 | 与 CMDB/VM 一致 |
| P1-6 | 大屏快捷操作接线 | 全局巡检 / 未处理告警 → 跳转或 `cron.run` | 有 loading / toast 反馈 |

**主要文件**：新建 `src/pkg/ops/`（或 gateway handlers）、`views/overview.ts`、`app-render.ts`、`app.ts`

### P1 勾选清单

- [x] P1-1 集群 API 与模型（见 [ops-api.md](./ops-api.md)）
- [x] P1-2 资产页接 API（列表 / 新增 / 刷新）
- [x] P1-3 实体选择器接 API（按域拉取集群 + 组件级选项）
- [x] P1-4 上下文注入 Agent（发送时前缀 + 输入区横幅）
- [x] P1-5 运维大屏真数据 + 五域卡片（汇总 API + VictoriaMetrics 健康分）
- [x] P1-6 大屏快捷操作接线（全局巡检 cron.run、告警子页跳转 + toast）

---

## P2 — 核心运维能力闭环（预估 ~2 周）

### P2-A 深度巡检（巩固阶段三）

| ID | 任务 | 验收标准 |
|----|------|----------|
| P2-A1 | `query_gbase_slow_sql` 接真实 DSN | 无 DSN 时明确错误，不静默返回模拟数据 |
| P2-A2 | `query_governance_lineage` 接治理 API | 同上 |
| P2-A3 | 巡检报告 Cursor 式排版 | 左列表 + 右 Markdown；顶部分数徽章；失败态可读 |
| P2-A4 | Hadoop JMX / FI Manager 专用 tool（可选） | 至少 1 域有非 VM 直连指标 |
| P2-A5 | 巡检 &lt;85 分 IM 推送可观测 | 未配置渠道时设置页有提示；文档说明 |

**后端现状（供对照）**：`cron/service.go` 已 `ensureDefaultJobs`；`query_vm_metrics` 为真实 HTTP；`direct_plugins.go` 中 GBase/治理仍为模拟。

### P2-B 告警降噪（阶段四 — 前端 + 存储）

| ID | 任务 | 验收标准 |
|----|------|----------|
| P2-B1 | 告警事件持久化 | 合并结果可查询：`GET /api/ops/alerts/groups` |
| P2-B2 | 告警 Tab 接 API | 替换 `getMockAlertGroups`；Cursor 式双栏列表+详情 |
| P2-B3 | 降噪统计真实计算 | 原始条数 / 合并组 / 降噪比来自存储 |
| P2-B4 | Root Cause 展示 | 告警组详情展示 Agent 分析 Markdown |
| P2-B5 | Alert Studio 产品路径 | 独立顶栏 **或** 保持域内子 Tab 并文档化 |

**后端现状**：`POST /hooks/alert` + `enqueueAlert` 滑动窗口合并 **已实现**；前端未消费。

### P2-C ChatOps

| ID | 任务 | 验收标准 |
|----|------|----------|
| P2-C1 | IM 告警卡片 + deep link | 飞书/企微模板；点击打开告警组 |
| P2-C2 | 入站指令 `/diagnose`、`/ack` 等 | 文档 + 至少 1 条 E2E |
| P2-C3 | 外部告警源联调 | `docs/alert-integration.md` checklist |

### P2 勾选清单

- [x] P2-A1 GBase 慢 SQL 真实接入（`GBASE_DSN`，无 DSN 明确报错）
- [x] P2-A2 治理血缘真实接入（`GOVERNANCE_API_URL`，无 URL 明确报错）
- [x] P2-A3 巡检报告 UI 统一（Markdown 详情，去除假指标表）
- [x] P2-A4 专用采集 tool（`query_hadoop_jmx`、`query_fi_manager_metrics`）
- [x] P2-A5 IM 推送可观测性（im-status API + 巡检 Tab 提示 + 未配置时日志）
- [x] P2-B1 告警持久化 API
- [x] P2-B2 告警 Tab 接 API
- [x] P2-B3 降噪统计真实化
- [x] P2-B4 Root Cause 详情（会话 Markdown）
- [x] P2-B5 Alert Studio 路径定稿（域内子 Tab，见 ops-api.md）
- [x] P2-C1 IM 告警卡片（飞书/钉钉 card + header + 深链）
- [x] P2-C2 ChatOps 指令（/help、/ack、/diagnose + 入站路由）
- [x] P2-C3 外部联调文档（alert-integration.md）

---

## P3 — Cursor 式体验统一（可与 P1/P2 并行，预估 ~1 周）

| ID | 任务 | 说明 |
|----|------|------|
| P3-1 | 设计 Token 落地 | 内联 style → CSS 类 + tokens |
| P3-2 | 顶栏与系统下拉 | 间距 8px、轻 hover，与 Cursor 一致 |
| P3-3 | 业务域侧栏 | Active 用左侧细条指示，非整块高亮底 |
| P3-4 | 控制条（集群选择器） | 单行、次要「刷新」按钮、下拉轻阴影 |
| P3-5 | 登录页统一 | 居中卡片、弱背景、单色主按钮 |
| P3-6 | 通用状态组件 | `<OpsEmpty>` `<OpsError>` `<OpsSkeleton>` 全站复用 |
| P3-7 | 去「大屏化」 | 减少过大数字、过多 emoji |

### P3 勾选清单

- [x] P3-1 ops-tokens / 去内联样式（ops-dashboard.css、控制条类）
- [x] P3-2 顶栏与下拉（layout.css：agent-core-dropdown / 圆形触发器）
- [x] P3-3 侧栏 active 样式（左侧细条）
- [x] P3-4 控制条（entity-selector CSS）
- [x] P3-5 登录页（ops-login.css）
- [x] P3-6 OpsEmpty / OpsError / OpsSkeleton
- [x] P3-7 去大屏化元素（统计字号、去渐变/emoji）
- [x] P3-8 ui-style-guide.md

---

## P4 — 安全、权限、交付（预估 3–5 天）

| ID | 任务 | 验收标准 |
|----|------|----------|
| P4-1 | CORS 生产配置 | 环境变量域名白名单，非 `*` |
| P4-2 | 按钮级 RBAC | `ops:diagnose`、`ops:inspect` 等前后端一致 |
| P4-3 | 文档与 task 同步 | `task.md` 与本文进度一致 |
| P4-4 | E2E 冒烟路径 | 登录 → 选域 → 选集群 → 巡检 → 报告 |
| P4-5 | 部署交付 | `docs/deploy-ops.md` + 镜像说明 |

### P4 勾选清单

- [x] P4-1 CORS（`OPENOCTA_CORS_ORIGINS` 白名单）
- [x] P4-2 按钮级 RBAC（ops:inspect / ops:ack + 前端 disabled）
- [x] P4-3 文档同步
- [x] P4-4 E2E checklist（e2e-ops-smoke.md）
- [x] P4-5 deploy-ops.md

---

## 建议迭代节奏

| Sprint | 范围 | 可演示成果 |
|--------|------|------------|
| **Sprint 1** | P0 + P3-6 基础组件 | 路由正确、无假数字、空状态统一 |
| **Sprint 2** | P1 全部 | 资产 CRUD、大屏真指标、上下文进 Agent |
| **Sprint 3** | P2-A + P2-B | 真巡检报告 + 真告警列表 |
| **Sprint 4** | P2-C + P4 | IM 联动 + 联调 + 上线 |

### 本周可立即开工（最小切片）

1. P0-1 修复 `assetManagement` 路由
2. P0-2 删除 `getMockInspections`，告警 Tab 空状态
3. P0-3 大屏硬编码 → 空状态
4. P3-6 新增 `OpsEmpty` 并在资产页使用
5. P1-1 起草 Cluster API 结构（OpenAPI / 类型定义）

---

## 与历史文档关系

| 文档 | 关系 |
|------|------|
| [1.md](./1.md) | 战略动机与三项 UX 目标 |
| [implementation_plan.md](./implementation_plan.md) | 30 天人天估算；本路线图按优先级重排 |
| [task.md](./task.md) | 阶段勾选表；与本文 P0–P4 映射 |

---

## 修订记录

| 日期 | 说明 |
|------|------|
| 2026-06-01 | 初版：基于代码评审与 Cursor 风格基线建立 P0–P4 |
| 2026-06-01 | P0 完成：路由修复、去 Mock、空状态组件、按域实体配置、告警诚实展示 |
| 2026-06-01 | P1-1/P1-2：集群 JSON 存储 + REST API；资产页与运维大屏汇总接入 |
| 2026-06-01 | P1-3/P1-4：实体选择器接集群 API；Agent 上下文前缀与横幅 |
| 2026-06-01 | P1-5/P1-6：VM 健康分写入 dashboard/summary；大屏全局巡检与告警跳转 |
| 2026-06-01 | P2-A1–A3、P2-B1–B5：告警 JSON 存储与 API、前端告警 Tab、工具 DSN/API、巡检 Markdown |
| 2026-06-01 | P2-A4/A5、P4-1/5、P2-C1/C3：JMX/FI 工具、IM 状态、CORS 白名单、深链与联调文档 |
| 2026-06-01 | P3 主体 + P2-C1：Cursor 风格 CSS、登录页、IM 告警卡片、ui-style-guide |
| 2026-06-01 | P2-C2、P4-2/4：ChatOps 入站指令、ops:ack/inspect 前端 RBAC、e2e-ops-smoke.md |
