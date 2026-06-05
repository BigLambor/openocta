# OpenOcta 智能运维平台架构重构 — 定稿实施方案

> 版本：v2.2 (Final)
> 日期：2026-06-05
> 状态：**待确认后执行**
> 基于：原架构方案 → AI 范式分析 → 三方评审 → 专家定稿，四轮共识

---

## 1. 核心共识（不可逆）

| # | 共识 | 状态 |
|---|------|------|
| 1 | 数字员工不做一级产品主轴，不作为一级导航入口 | ✅ 必须执行 |
| 2 | 三个平行入口（技术域/能力中心/数字员工中心）收敛 | ✅ 必须执行 |
| 3 | AI 嵌入工作流场景，不独立成"中心" | ✅ 必须执行 |
| 4 | "AI 与自动化"不做一级导航，自动化配置归入系统管理 | ✅ 必须执行 |
| 5 | AI 运维助手保留为全局 Copilot，但不替代运维工作台 | ✅ 必须执行 |
| 6 | 先验证一个 BCH 闭环场景再扩展骨架 | ✅ 必须执行 |
| 7 | "数字员工"是品牌词不是架构词：品牌保留，架构瘦身（详见 §1.1） | ✅ 必须执行 |
| 8 | 技术域作为全局上下文过滤器，不复制完整能力菜单 | ✅ 必须执行 |
| 9 | 角色化首页（工程师→工作台，管理者→驾驶舱） | ✅ 必须执行 |

### 1.1 "数字员工" 命名原则

"数字员工"已在团队和领导层获得认可，具备商业辨识度。**保留品牌，不做架构。**

| 场景 | 是否用"数字员工" | 示例 |
|------|:---:|------|
| 方案书、PPT、商务沟通 | ✅ 用 | "OpenOcta 数字员工能力" |
| 模板/卡片的名称与描述 | ✅ 用 | "BCH 巡检数字员工"、"值班数字员工" |
| 驾驶舱/工作台中的状态明细 | ✅ 用 | "由 BCH 巡检数字员工模板执行" |
| AI 交互上下文 | ✅ 用 | "使用 BCH 诊断数字员工的专家人设、偏好和技能组合" |
| 产品引导 Tour | ✅ 用 | "挑选数字员工模板" |
| 一级导航入口名 | ❌ 不用 | 不叫"数字员工中心"，也不把"自动化配置"做成一级 Tab |
| 侧边栏/菜单的模块名 | ❌ 不用 | 叫"助手模板库""我的助手"，不叫"员工市场""我的员工" |
| 管理体系（档案/绩效/编排） | ❌ 不用 | 不搞"员工档案12个字段""员工绩效" |

> 简单说：用户在系统里看到的**内容明细**可以叫"数字员工"，但**导航菜单、管理概念和核心指标**不叫"数字员工"。

### 1.2 数字员工能力保留说明

本方案取消的是“数字员工中心”作为一级产品主轴，不取消数字员工本身。

OpenOcta 中的数字员工继续作为 AI 交互的角色化上下文包存在，用于封装：

- 领域人设：如 BCH 专家、GBase 专家、值班运维、巡检专家、作业诊断专家。
- 能力偏好：擅长诊断、巡检、治理、容量分析、变更护航等。
- 背景知识：技术域、对象类型、常见故障、分析口径和业务约束。
- Prompt 策略：系统提示词、输出风格、任务边界和安全约束。
- Skill / MCP / Runbook：可调用工具、数据源和自动化能力。
- 输出模板：诊断报告、巡检报告、治理建议、处置步骤、复盘草稿。

因此，数字员工不是一级导航和管理体系，而是 AI 助手背后的**专家角色 + 背景上下文 + 技能组合包**。

在工作台点击场景 AI 操作时，系统应根据技术域、对象类型和任务类型自动推荐或注入合适的数字员工模板：

| 场景 | 推荐数字员工模板 |
|------|------------------|
| BCH 告警 | BCH 值班数字员工 |
| HDFS 巡检 | BCH 巡检数字员工 |
| Spark 作业失败 | Spark 诊断数字员工 |
| GBase 慢 SQL | GBase 诊断数字员工 |
| 变更风险评估 | 变更护航数字员工 |

这样既保留原 OpenOcta “人设 + 技能组合包”的核心价值，又避免把产品主线变成“管理员工”。

---

## 2. 目标一级导航

### 2.1 导航结构

```text
┌────────────────────────────────────────────────────────────────────────┐
│  ApexOps  v0.x  │  AI 运维助手  运维驾驶舱  运维工作台  服务与资产  ⚙️ 👤 │
└────────────────────────────────────────────────────────────────────────┘
               ↑                                  ↑
    全局 Copilot 入口 + 工作台内嵌 AI      系统管理/自动化配置在齿轮下拉
```

| 导航项 | Tab 标识 | 路由 | 定位 | 目标用户 |
|--------|---------|------|------|----------|
| AI 运维助手 | `message` | `/message` | 全局 Copilot、跨域问答、探索分析、快速检索和任务跳转 | 所有角色 |
| 运维驾驶舱 | `overview` | `/overview` | 全局态势感知 | 运维管理者 |
| 运维工作台 | `workbench` | `/workbench` | 日常运维工作 | 运维工程师 |
| 服务与资产 | `assets` | `/assets` | 技术域资产、拓扑、依赖 | 所有角色 |
| 系统管理 | （⚙️ 齿轮下拉） | 各自路由 | 模型、权限、安全、通道 | 平台管理员 |

> `自动化配置` 不是一级导航 Tab，只作为系统管理下拉中的管理员入口。普通运维工程师默认不可见，不作为日常工作入口。

### 2.2 原 "AI 运维助手" 入口处理

原顶栏第一个 Tab `message`（"AI 运维助手"）保留，但重新定位为全局 Copilot 入口，不作为运维闭环主入口。

| 方式 | 位置 | 说明 |
|------|------|------|
| 顶栏 Tab | `/message` | 全局 AI 助手入口，用于跨域问答、探索分析、快速检索和任务跳转 |
| 工作台内嵌对话 | `/workbench` 内的侧边面板 | 日常使用主入口 |
| 快捷入口 | 系统管理下拉菜单中保留"AI 对话"入口 | 兼容直接对话场景 |

> **注意**：`AI 运维助手` 保留为全局 Copilot，不替代运维工作台。告警、巡检、诊断、治理、容量、变更等结构化工作流仍以 `/workbench` 为主入口。

### 2.3 与现有导航的映射

**当前代码位置**: [`app-render.ts` L802-876](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-render.ts#L802-L876)

当前顶栏 5 个 Tab：

```typescript
// 当前 (app-render.ts L803-808)
{ tab: "message", label: "AI 运维助手" },
{ tab: "overview", label: "运维驾驶舱" },
{ tab: "opsCapabilities", label: "运维能力中心" },
{ tab: "techDomains", label: "技术域运维" },
{ tab: "employeeCenter", label: "数字员工中心" },
```

改为：

```typescript
// 目标
{ tab: "message", label: "AI 运维助手" },       // 保留，重新定位为全局 Copilot
{ tab: "overview", label: "运维驾驶舱" },
{ tab: "workbench", label: "运维工作台" },     // 新增
{ tab: "assets", label: "服务与资产" },          // 新增
```

### 2.4 现有路由完整映射表

| 现有路由/Tab | 现有标题 | 处理方式 | 新归属 |
|-------------|---------|---------|--------|
| `message` `/message` | AI 运维助手 | **保留**，重新定位为全局 Copilot，不承载结构化运维闭环主流程 | 一级导航 |
| `overview` `/overview` | 运维驾驶舱 | **保留** | 一级导航 |
| `opsCapabilities` `/ops-capabilities` | 运维能力中心 | **取消一级入口**，能力拆入工作台 | 运维工作台子模块 |
| `techDomains` `/tech-domains` | 技术域运维 | **取消一级入口**，改为全局上下文过滤器 | 服务与资产 |
| `employeeCenter` `/employee-center` | 数字员工中心 | **取消** | — |
| `employeeMarket` `/employee-market` | 员工市场 | 菜单**改名**为"助手模板库"，内容保留"数字员工"称谓 | 系统管理 → 自动化配置 |
| `digitalEmployee` `/digital-employee` | 我的员工 | 菜单**改名**为"我的助手"，卡片/模板保留"数字员工"称谓 | 系统管理 → 自动化配置 |
| `agentSwarm` `/agent-swarm` | 员工编排 | **改名**为"工作流编排" | 系统管理 → 自动化配置 |
| `employeeTasks` `/employee-tasks` | 任务记录 | **改名**为"执行记录" | 系统管理 → 自动化配置 |
| `employeeEffectiveness` `/employee-effectiveness` | 效能评估 | **改名**为"自动化效果" | 系统管理 → 自动化配置 |
| `hadoop` `/hadoop` | BCH 生态 | **取消独立 Tab**，改为资产上下文过滤 | 服务与资产 (domain=hadoop) |
| `fi` `/fi` | FI 商业生态 | 同上 | 服务与资产 (domain=fi) |
| `gbase` `/gbase` | GBase 数据库 | 同上 | 服务与资产 (domain=gbase) |
| `governance` `/governance` | 开发治理平台 | 同上 | 服务与资产 (domain=governance) |
| `dataapps` `/dataapps` | 数据 App 运维 | 同上 | 服务与资产 (domain=dataapps) |
| `assetManagement` `/asset-management` | 集群资产管理 | **迁移** | 服务与资产子模块 |
| `skillLibrary` `/skill-library` | 技能库 | 保留 | 系统管理下拉 |
| `toolLibrary` `/tool-library` | 工具库 | 保留 | 系统管理下拉 |
| `modelLibrary` `/model-library` | 模型 | 保留 | 系统管理下拉 |
| `channels` `/channels` | 通道配置 | 保留 | 系统管理下拉 |
| `scheduledTasks` `/scheduled-tasks` | 定时任务 | 保留 | 系统管理下拉 |
| `config` `/config` | 系统配置 | 保留 | 系统管理下拉 |
| `sandbox` `/sandbox` | 安全策略 | 保留 | 系统管理下拉 |
| `tutorials` `/tutorials` | 教程 | 保留 | 系统管理下拉 |

### 2.5 角色化默认首页

| 角色 | 默认首页 | 实现方式 |
|------|----------|----------|
| 运维工程师 | `/workbench` | 用户首次登录时选择角色，存 localStorage |
| 运维管理者 | `/overview` | 同上 |
| 平台管理员 | `/overview` 或 `/assets` | 同上，自动化配置通过系统管理进入 |

> 第一阶段简化：`/` 默认跳转 `/overview`（保持现有行为），系统管理下拉中增加"首页偏好"设置项。

---

## 3. 技术域上下文过滤器

### 3.1 设计

技术域从独立的一级/二级 Tab 改为**全局上下文选择器**，嵌入运维工作台和服务与资产的页面头部。

```text
┌─ 运维工作台 ──────────────────────────────────────┐
│  [技术域: 全部 ▼]  事件中心  巡检中心  诊断中心... │
│  ─────────────────────────────────────────────────│
│         ↓ 选择 BCH 后                             │
│  [技术域: BCH 生态 ▼]  事件中心  巡检中心  ...     │
│  所有数据自动按 BCH 过滤                           │
└───────────────────────────────────────────────────┘
```

### 3.2 行为规则

| 行为 | 说明 |
|------|------|
| 选择技术域后 | 工作台、资产页面的数据自动按该技术域过滤 |
| 选择"全部" | 显示跨域汇总视图 |
| 状态持久化 | 选择存入 `localStorage`，刷新后保持 |
| URL 参数 | `?domain=hadoop`，方便分享和书签 |
| 权限关联 | 复用现有 RBAC `menu:hadoop` / `menu:fi` 等权限控制可见域列表 |

### 3.3 技术域定义复用

后端已有完整的技术域常量定义：

**文件**: [`taxonomy.go` L5-11](file:///Users/isadmin/MagicSpace/openocta/src/pkg/employees/taxonomy.go#L5-L11)

```go
DomainHadoop     = "hadoop"
DomainFI         = "fi"
DomainGBase      = "gbase"
DomainGovernance = "governance"
DomainDataApps   = "dataapps"
```

前端选择器的域列表复用这些值，无需新增后端定义。

### 3.4 现有技术域视图内容迁移

当前每个技术域视图（`tech-ops-domain.ts`）内部有 10 个能力域 Tab：

**文件**: [`ops/navigation.ts` L3-13](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/ops/navigation.ts#L3-L13)

```typescript
"overview" | "assetTopology" | "observability" | "inspection" |
"jobGovernance" | "diagnosis" | "governance" | "capacity" | "change" | "employees"
```

迁移方案：

| 技术域内能力域 Tab | 迁移目标 |
|-------------------|---------|
| `overview` | 驾驶舱（按技术域过滤时的概览） |
| `assetTopology` | 服务与资产 |
| `observability` | 运维工作台 → 事件中心 |
| `inspection` | 运维工作台 → 巡检中心 |
| `jobGovernance` | 运维工作台 → 治理中心（BCH 特有，作为治理子类） |
| `diagnosis` | 运维工作台 → 诊断中心 |
| `governance` | 运维工作台 → 治理中心 |
| `capacity` | 运维工作台 → 容量性能 |
| `change` | 运维工作台 → 变更护航 |
| `employees` | 系统管理 → 自动化配置 → 我的助手（按技术域过滤） |

---

## 4. 运维工作台

### 4.1 子模块结构

```text
运维工作台 /workbench
├── 工作看板    /workbench           (默认首页，汇总待办)
├── 事件中心    /workbench?cap=incidents
├── 巡检中心    /workbench?cap=inspections
├── 诊断中心    /workbench?cap=diagnosis
├── 治理中心    /workbench?cap=governance
├── 容量性能    /workbench?cap=capacity
└── 变更护航    /workbench?cap=changes
```

> 工作台内部使用 Tab 切换（类似当前技术域的能力域 Tab），不是独立路由。

### 4.2 工作台首页 — 工作看板

```text
┌─────────────────────────────────────────────────────┐
│  [技术域: 全部 ▼]                                    │
├─────────────────────────────────────────────────────┤
│  ⚠️ 待处理告警 (12)   │  🔴 待处理事件 (3)           │
│  📋 今日巡检 (2)       │  🔍 进行中诊断 (1)           │
│  🔧 治理任务 (5)       │  🚀 变更护航 (0)             │
├─────────────────────────────────────────────────────┤
│  最近 AI 分析                                        │
│  · BCH 集群告警根因分析 — 3 分钟前 — [查看]           │
│  · HDFS 深度巡检报告 — 1 小时前 — [查看]              │
└─────────────────────────────────────────────────────┘
```

### 4.3 工作台内 AI 嵌入点

每个工作台子模块内嵌 AI 操作按钮：

| 模块 | 页面场景 | AI 按钮 |
|------|---------|---------|
| 事件中心 | 告警详情 | [🤖 分析根因] [🤖 聚合相似] [🤖 处置建议] |
| 事件中心 | 事件详情 | [🤖 影响面分析] [🤖 推荐 Runbook] [🤖 生成复盘] |
| 巡检中心 | 巡检报告 | [🤖 解释风险] [🤖 生成治理任务] [🤖 趋势分析] |
| 诊断中心 | 诊断详情 | [🤖 深度根因] [🤖 关联日志指标] [🤖 推荐修复] |
| 治理中心 | 治理任务 | [🤖 生成治理建议] [🤖 评估效果] |
| 容量性能 | 容量页面 | [🤖 预测水位] [🤖 识别热点] [🤖 扩容建议] |
| 变更护航 | 变更单 | [🤖 评估风险] [🤖 生成验证项] |

**实现方式**：AI 按钮打开侧边对话面板，自动注入当前上下文（技术域 + 对象 ID + 数据范围），并匹配对应数字员工模板（专家人设 + Prompt + Skill/MCP + 输出模板）调用现有 Chat API。

---

## 5. 系统管理中的自动化配置

### 5.1 结构

```text
系统管理 / 自动化配置
├── 助手模板库     /automation?sub=templates       # 原 employeeMarket (模板内容仍称"数字员工")
├── 我的助手       /automation?sub=assistants       # 原 digitalEmployee (卡片仍称"数字员工")
├── 工作流编排     /automation?sub=workflows        # 原 agentSwarm
├── 执行记录       /automation?sub=executions       # 原 employeeTasks
└── 自动化效果     /automation?sub=effectiveness    # 原 employeeEffectiveness
```

> `/automation` 路由保留，但入口位于系统管理下拉中；它不是顶栏一级 Tab，也不是日常运维入口。

### 5.2 UI 文案改造清单

涉及文件及改动：

#### [`navigation.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/navigation.ts) — 标题与副标题

| 函数 | 行号 | 原文 | 改为 | 说明 |
|------|------|------|------|------|
| `titleForTab` | L304 | `"数字员工中心"` | `"自动化配置"` | 系统管理入口名，不用"数字员工" |
| `titleForTab` | L306 | `"员工市场"` | `"助手模板库"` | 导航菜单名 |
| `titleForTab` | L368 | `"我的员工"` | `"我的助手"` | 导航菜单名 |
| `titleForTab` | L370 | `"员工编排"` | `"工作流编排"` | 导航菜单名 |
| `titleForTab` | L308 | `"任务记录"` | `"执行记录"` | 导航菜单名 |
| `titleForTab` | L310 | `"效能评估"` | `"自动化效果"` | 导航菜单名 |
| `subtitleForTab` | L458 | 数字员工中心副标题 | `"管理数字员工模板、工作流编排、触发规则和执行记录"` | 内容描述保留"数字员工" |
| `subtitleForTab` | L460 | 员工市场副标题 | `"发现、安装和上架面向运维场景的数字员工模板"` | 内容描述保留"数字员工" |
| `subtitleForTab` | L462 | 任务记录副标题 | `"数字员工处理告警、巡检、诊断、治理等场景的执行记录"` | 内容描述保留"数字员工" |
| `subtitleForTab` | L464 | 效能评估副标题 | `"量化自动化执行效果：任务量、成功率、闭环率和成本"` | 效能指标不拟人 |
| `subtitleForTab` | L510 | 我的员工副标题 | `"管理已安装和自建数字员工，配置 Prompt、技能和工具"` | 内容描述保留"数字员工" |
| `subtitleForTab` | L512 | 员工编排副标题 | `"编排自动化步骤、触发条件、审批节点和工具调用"` | 编排对象不拟人 |

#### [`app-render.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-render.ts) — 侧边栏标题

| 位置 | 行号范围 | 原文 | 改为 |
|------|---------|------|------|
| 自动化配置侧边栏 | L1291-1334 | "数字员工中心"/"员工市场"/"我的员工"/"员工编排"/"任务记录"/"效能评估" | "自动化配置"/"助手模板库"/"我的助手"/"工作流编排"/"执行记录"/"自动化效果" |

> **注意**：侧边栏的**菜单项名称**用新名称，但各页面**内部内容**中的模板卡片、描述文字仍可使用"数字员工"称谓。

#### 产品引导 (Product Tour)

[`navigation.ts` L410-415](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/navigation.ts#L410-L415)

```typescript
// 原
{ tab: "employeeMarket", title: "数字员工", body: "在「员工市场」挑选或启用数字员工模板..." }
// 改为（保留"数字员工"品牌词，仅更新入口名称）
{ tab: "employeeMarket", title: "数字员工", body: "在「助手模板库」挑选数字员工模板，快速落地巡检、诊断、值班等运维助手。" }
```

### 5.3 后端 API 兼容策略

| 层级 | 现有 | 本次改造 |
|------|------|---------|
| WebSocket 方法 | `employees.list`/`employees.get`/`employees.create`/`employees.delete` | **不改** |
| WebSocket 方法 | `employee.tasks.list`/`employee.tasks.get`/... | **不改** |
| HTTP REST | `/api/ops/*` | **不改** |
| Manifest 模型 | `pkg/employees/model.go` 完整字段 | **不改** |
| EmployeeTask 模型 | `pkg/employees/task_model.go` | **不改** |
| Taxonomy 常量 | `pkg/employees/taxonomy.go` | **不改** |

> 后端 API 和模型本阶段全部保持不变。仅前端展示层改名。后续如需 API 改名，通过路由别名保持向后兼容。

---

## 6. 全局 AI 助手交互

### 6.1 触达方式

| 方式 | 位置 | 说明 |
|------|------|------|
| 顶栏 Tab | `/message` | 全局 Copilot 入口，用于跨域问答、探索分析、快速检索和任务跳转 |
| 工作台内嵌面板 | 工作台右侧可展开/收起的侧边面板 | 日常使用主入口 |
| 场景内 AI 按钮 | 各页面内（告警详情/巡检报告等） | 自动注入上下文后打开面板 |
| 快捷键 | `Cmd+K` / `Ctrl+K` | 任何页面唤起 |
| 系统管理入口 | 齿轮下拉菜单中 "AI 对话" | 独立全屏对话（兼容） |

### 6.2 上下文自动注入

当用户从工作台场景触发 AI 时，自动携带：

```json
{
  "context": {
    "domain": "hadoop",
    "workflowType": "incident",
    "objectType": "alert",
    "objectId": "alert-12345",
    "clusterId": "bch-prod-01",
    "assistantTemplate": "bch-oncall-digital-employee"
  }
}
```

`assistantTemplate` 用于注入数字员工模板，包含专家人设、偏好、Prompt 策略、Skill/MCP 和输出模板。未显式指定时，由系统根据 `domain`、`workflowType`、`objectType` 自动推荐。

### 6.3 对现有对话系统的改造

| 现有 | 改造 |
|------|------|
| `message` Tab 全屏对话页 | 保留为顶栏全局 Copilot |
| 对话需手动选择员工 | 从场景进入时自动推荐匹配的助手模板 |
| 独立全屏 | 新增侧边面板模式（工作台内嵌） |
| 普通聊天结果 | 增加结构化落点：跳转告警/事件/巡检报告、创建治理任务、触发 Runbook |

---

## 7. 服务与资产模块

### 7.1 结构

```text
服务与资产 /assets
├── 资产总览    /assets              (默认首页)
├── 集群资产    /assets?view=clusters
├── 组件资产    /assets?view=components
├── 作业资产    /assets?view=jobs
├── 服务依赖    /assets?view=dependencies
└── 拓扑视图    /assets?view=topology
```

### 7.2 复用现有能力

- 集群 CRUD 复用现有 [`asset-management.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/asset-management.ts) 和后端 `/api/ops/clusters` API
- 技术域概览数据复用 `/api/ops/dashboard/summary`
- 告警数据复用 `/api/ops/alerts/groups`

---

## 8. 驾驶舱增强

### 8.1 现有 [`overview.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/overview.ts) 改造

| 区域 | 现状 | 增强 |
|------|------|------|
| 全局健康度 | 已有骨架 | 按技术域过滤器联动 |
| 告警概览 | 已有 | 增加 Top 告警、未处理数、趋势 |
| 巡检风险 | 已有 | 显示最近巡检结果、风险项 |
| 容量水位 | 已有 | 按技术域显示关键资源水位 |
| "数字员工状态" | 现有文案 | **改为"自动化执行状态"**；明细中可展示"由 BCH 巡检数字员工模板执行" |

---

## 9. 前端文件级改造清单

### 9.1 核心改动文件

| 文件 | 改动类型 | 改动内容 |
|------|---------|---------|
| [`navigation.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/navigation.ts) | **修改** | ① 新增 Tab 类型 `workbench`/`assets` ② 保留 `message` 顶栏 Tab ③ 保留 `automation` 路由但不作为顶栏 Tab ④ 文案改名（§5.2） ⑤ 路由映射 ⑥ 产品引导更新 |
| [`app-render.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-render.ts) | **修改** | ① 顶栏 Tab 列表(L802-808)：保留 `message`，移除能力/技术域/员工中心 ② 系统管理下拉增加自动化配置入口 ③ 侧边栏逻辑(L1291-1334) ④ `isDigitalEmployeeArea` 判断(L668-674) ⑤ shell class 逻辑 |
| [`app-view-state.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-view-state.ts) | **修改** | ① 新增 `selectedDomain` 状态 ② 默认 Tab 逻辑 |
| [`ops/navigation.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/ops/navigation.ts) | **修改** | 技术域导航逻辑调整 |

### 9.2 新增文件

| 文件 | 说明 |
|------|------|
| `ui/src/ui/views/workbench.ts` | 运维工作台（工作看板 + 子模块 Tab） |
| `ui/src/ui/views/assets-view.ts` | 服务与资产页面 |
| `ui/src/ui/views/automation-hub.ts` | 系统管理下的自动化配置 Hub 页面 |
| `ui/src/ui/components/domain-filter.ts` | 技术域上下文过滤器组件 |
| `ui/src/ui/components/ai-action-button.ts` | 通用 AI 操作按钮组件 |
| `ui/src/ui/components/ai-side-panel.ts` | AI 侧边对话面板组件 |

### 9.3 内容迁移（不删除原文件，P3 阶段确认后移除）

| 原文件 | 内容迁移目标 |
|--------|-------------|
| [`employee-center.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/employee-center.ts) | `automation-hub.ts`（系统管理入口） |
| [`employee-market.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/employee-market.ts) | 改文案，归入自动化配置侧边栏 |
| [`digital-employee.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/digital-employee.ts) | 改文案，归入自动化配置侧边栏 |
| [`employee-operations.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/employee-operations.ts) | 改文案（任务/效能） |
| [`agent-swarm.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/agent-swarm.ts) | 改文案 |
| [`ops-capability-center.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/ops-capability-center.ts) | 能力卡片迁入工作台导航 |
| [`tech-ops-hub.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/tech-ops-hub.ts) | 技术域健康矩阵迁入资产/驾驶舱 |
| [`tech-ops-domain.ts`](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/views/tech-ops-domain.ts) | 各能力域内容迁入对应工作台子模块 |

### 9.4 P3 阶段待删除文件

```text
ui/src/ui/views/employee-center.ts
ui/src/ui/views/ops-capability-center.ts
ui/src/ui/views/tech-ops-hub.ts
```

---

## 10. 后端改造

### 10.1 本次不改的后端文件

| 文件 | 原因 |
|------|------|
| [`model.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/employees/model.go) (Manifest) | 模型结构完整，无需修改 |
| [`task_model.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/employees/task_model.go) (EmployeeTask) | 任务模型完整，无需修改 |
| [`taxonomy.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/employees/taxonomy.go) | 域/能力/状态常量完整 |
| [`handlers/employees.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/employees.go) | WebSocket API 不改 |
| [`handlers/employee_tasks.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/employee_tasks.go) | WebSocket API 不改 |
| [`http/server.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/server.go) | HTTP 路由不改 |
| [`http/ops_api.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/ops_api.go) | Ops REST API 不改 |
| [`http/bch_api.go`](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/bch_api.go) | BCH API 不改 |

### 10.2 P0 阶段后端改动

| 文件 | 改动 |
|------|------|
| Chat handler | 支持 `context` 参数（domain, workflowType, objectType, objectId），传入 Prompt 上下文 |
| Assistant template resolver | 根据上下文自动匹配数字员工模板，并注入专家人设、偏好、Skill/MCP 和输出模板 |

### 10.3 P3 阶段后端新增

| 文件 | 说明 |
|------|------|
| `pkg/automation/triggers.go` | 触发规则引擎（告警触发、定时触发） |
| `pkg/automation/scheduler.go` | 定时任务调度（复用现有 `pkg/cron/`） |

---

## 11. RBAC 权限适配

### 11.1 现有权限

当前 RBAC 已有：`menu:overview`、`menu:techDomains`、`menu:employeeCenter`、`menu:hadoop`/`fi`/`gbase`/`governance`/`dataapps`

### 11.2 权限映射

| 现有权限 | 新 Tab | 处理 |
|----------|--------|------|
| `menu:overview` | `overview` | 保持 |
| `menu:techDomains` | — | 改为控制技术域过滤器可见性 |
| `menu:employeeCenter` | 系统管理 → 自动化配置 | 映射为管理员配置入口 |
| `menu:hadoop` 等 | — | 改为控制域过滤器中该域的可见性 |
| 新增 `menu:workbench` | `workbench` | 默认所有用户可见 |
| 新增 `menu:assets` | `assets` | 默认所有用户可见 |
| 新增 `menu:automation` | 系统管理 → 自动化配置 | 仅管理员可见，不出现在普通运维角色导航中 |

---

## 12. 分阶段执行计划

### P0：BCH 告警闭环验证（1-2 周）

> **目标**：用一个端到端闭环场景证明 AI 价值。不改正式导航结构。

| ID | 任务 | 涉及文件 | 验收标准 |
|----|------|----------|----------|
| P0-1 | 工作台骨架 + 事件中心页面 | 新增 `workbench.ts` | 可展示 BCH 告警列表（复用现有 `/api/ops/alerts/groups`） |
| P0-2 | AI 操作按钮 + 侧边对话面板 | 新增 `ai-action-button.ts`、`ai-side-panel.ts` | 点击告警上的"分析根因"按钮，侧边面板展示 AI 分析结果 |
| P0-3 | 上下文注入 + 数字员工模板匹配 | 修改 Chat handler / template resolver | AI 能自动感知当前告警信息，并使用 BCH 值班数字员工模板进行分析 |
| P0-4 | 处置建议 + 确认/驳回 | `workbench.ts` 增强 | 用户可确认或驳回 AI 建议，写入执行记录 |
| P0-5 | 临时访问入口 | `navigation.ts`/`app-render.ts` | 通过 `/workbench-beta`、系统管理临时入口或 Feature Flag 访问；不加入正式顶栏 |

> P0 阶段验证成功后，再进入 P1 全面改造。

---

### P1：导航重构（1-2 周）

> **目标**：完成一级导航切换，技术域改为上下文过滤器。

| ID | 任务 | 涉及文件 | 验收标准 |
|----|------|----------|----------|
| P1-1 | 重构顶栏 Tab 列表 | `app-render.ts` L802-876 | 5→4 个 Tab（AI 运维助手/驾驶舱/工作台/资产）+ 齿轮下拉 |
| P1-2 | 新增 Tab 类型和路由 | `navigation.ts` | `workbench`/`assets` 路由生效；`automation` 仅系统管理入口可达 |
| P1-3 | 技术域上下文过滤器 | 新增 `domain-filter.ts` | 可切换技术域，状态持久化 |
| P1-4 | 侧边栏逻辑调整 | `app-render.ts` L1291-1334 | 自动化配置区域用新文案 |
| P1-5 | 旧路由兼容重定向 | `navigation.ts` | `/employee-center` → `/automation` 等 |
| P1-6 | RBAC 权限映射 | `app-render.ts` 权限检查逻辑 | 新 Tab 权限正常 |

---

### P2：工作台 + 资产模块落地（2-3 周）

> **目标**：事件中心完整上线，巡检中心半完整上线，其他工作台模块以骨架态承载后续扩展；资产页面上线。

| ID | 任务 | 涉及文件 | 验收标准 |
|----|------|----------|----------|
| P2-1 | 工作台首页（工作看板） | `workbench.ts` 完善 | 6 个模块待办计数 + 快捷入口 |
| P2-2 | 事件中心完善 | `workbench.ts` | 告警列表、事件详情、AI 分析、状态流转 |
| P2-3 | 巡检中心 | `workbench.ts` | 巡检报告、风险项、AI 解释 |
| P2-4 | 诊断/治理/容量/变更中心 | `workbench.ts` | 骨架页面 + 即将上线状态；仅保留明确可用的 AI 按钮 |
| P2-5 | 资产页面 | `assets-view.ts` | 复用现有集群 CRUD，按技术域过滤 |
| P2-6 | 角色化首页 | `app-view-state.ts` | 首次登录选角色，默认首页分流 |

---

### P3：文案改造 + 旧页面清理（1-2 周）

> **目标**：原数字员工相关页面完成改名和迁移。

| ID | 任务 | 涉及文件 | 验收标准 |
|----|------|----------|----------|
| P3-1 | navigation.ts 导航文案改名 | `navigation.ts` | 菜单名改新名称，副标题/描述保留"数字员工" |
| P3-2 | 员工市场页面菜单文案改造 | `employee-market.ts` | 页面标题改为"助手模板库"，模板卡片内容保留"数字员工" |
| P3-3 | 我的员工页面菜单文案改造 | `digital-employee.ts` | 页面标题改为"我的助手"，卡片内容保留"数字员工" |
| P3-4 | 员工编排页面文案改造 | `agent-swarm.ts` | 改为"工作流编排" |
| P3-5 | 任务/效能页面文案改造 | `employee-operations.ts` | 改为"执行记录"/"自动化效果" |
| P3-6 | 驾驶舱文案检查 | `overview.ts` | 主标题改为"自动化执行状态"，明细可保留"数字员工模板"称谓 |
| P3-7 | 删除员工中心首页 | 删除 `employee-center.ts` | 路由重定向正常 |
| P3-8 | 删除运维能力中心 | 删除 `ops-capability-center.ts` | 路由重定向正常 |

---

### P4：场景深化 + 自动化触发（2-4 周）

| ID | 任务 | 验收标准 |
|----|------|----------|
| P4-1 | BCH 告警闭环接入真实数据 | 真实告警可处理 |
| P4-2 | BCH 巡检闭环 | 定时巡检 + 报告 + 风险 |
| P4-3 | 自动化触发规则 | 告警触发 AI 分析、定时触发巡检 |
| P4-4 | 自动化效果看板接入真实数据 | 执行次数、成功率可查 |
| P4-5 | 工作台各场景 AI 按钮全部接通 | 所有 6 个子模块 AI 可用 |

---

### P5：复制到其他技术域 + 持续迭代（持续）

| ID | 任务 | 验收标准 |
|----|------|----------|
| P5-1 | GBase 数据库场景接入 | 切换域到 GBase，工作台可用 |
| P5-2 | FI 商业生态场景接入 | 同上 |
| P5-3 | 开发治理平台接入 | 同上 |
| P5-4 | 数据 App 运维接入 | 同上 |
| P5-5 | 后端 API 命名改造（可选） | WebSocket 方法别名 |
| P5-6 | 权限体系按技术域隔离 | 不同团队只看到自己负责的域 |

---

## 13. 验收标准

| # | 标准 | 验证方式 |
|---|------|----------|
| 1 | 一级导航为 4 个 Tab + 齿轮下拉（AI 运维助手/驾驶舱/工作台/资产），不出现"数字员工中心"或一级"自动化配置" | UI 截图 |
| 2 | 技术域通过页面内选择器切换，不作为独立菜单树 | UI 操作录屏 |
| 3 | 工作台可处理告警、巡检、诊断、治理、容量、变更 | 功能演示 |
| 4 | AI 操作按钮在场景内可用（至少告警和巡检） | UI 截图 |
| 5 | AI 侧边对话面板可在工作台内展开，自动注入上下文 | 功能演示 |
| 6 | 新增场景时不需要新增一级菜单 | 加一个 Kafka 诊断场景验证 |
| 7 | BCH 告警闭环可跑通（告警→分析→建议→确认→记录） | 端到端演示 |
| 8 | 前端可正常构建 `npm run build` 通过 | CI 验证 |
| 9 | 所有旧路由有重定向，不出现 404 | 手动测试 |
| 10 | 普通运维角色看不到自动化配置入口，管理员可从系统管理进入 | 权限测试 |
| 11 | 场景 AI 能自动匹配数字员工模板，保留专家人设、偏好、Skill/MCP 和输出模板能力 | 功能演示 |

---

## 14. 风险与缓解

| 风险 | 影响 | 缓解措施 |
|------|------|----------|
| 导航改动影响面大，可能引入回归 | 高 | P0 阶段新旧共存，P1 保留旧路由重定向 |
| `app-render.ts` 4657 行改动风险 | 高 | 精确定位改动区域，改前跑通构建 |
| 技术域过滤器需要所有页面适配 | 中 | 通过全局 state 传递 selectedDomain，子页面按需过滤 |
| 后端无真实告警/巡检数据 | 中 | P0 可用 Mock 数据验证流程，P4 接真实数据 |
| "数字员工"改名影响远程市场兼容性 | 低 | API 不改，只改前端展示文案 |
| 工作台子模块初期内容空 | 中 | 事件中心完整、巡检半完整，其他子模块明确标记骨架态 |
| 自动化配置变成新的一级工作中心 | 中 | 仅管理员可见，放入系统管理下拉，不作为默认首页和日常入口 |
| AI 运维助手顶栏入口被误用为唯一主入口 | 中 | 明确其为全局 Copilot；告警、巡检、诊断、治理、容量、变更仍以工作台为主入口 |

---

## 15. 总结

```text
核心变化:
  数字员工中心 → 降级为"系统管理/自动化配置"中的助手模板
  数字员工本体 → 保留为专家人设 + 背景上下文 + 技能组合包
  运维能力中心 → 拆解为"运维工作台"的子模块
  技术域独立菜单 → 全局上下文过滤器
  AI 运维助手 → 保留为全局 Copilot，不替代工作台
  五个入口 → 四个 Tab + 齿轮下拉

不变的:
  后端 API 路径和模型结构（全部保持兼容）
  Manifest / EmployeeTask / Taxonomy
  Skill / MCP / Runbook 能力
  现有工具链（HDFS/YARN/Spark/Flink/Alert/Inspection）
  ChatOps 通道集成

执行原则:
  先验证再铺骨架（P0 先跑通 BCH 告警闭环）
  新旧共存过渡（旧路由重定向，不一刀切）
  场景闭环优先（P0 即支持 Chat context，不能等到导航重构后再做）
  逐步移除旧页面（确认新页面稳定后再删）
  AI 日常运维能力嵌入工作流，全局 Copilot 只做跨域问答、探索分析和任务跳转
```
