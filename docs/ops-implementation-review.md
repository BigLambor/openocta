# OpenOcta 企业级 AIOps 改造实现审查报告

审查日期：2026-06-02  
审查范围：`docs/1.md`、`docs/ops-roadmap.md`、`docs/task.md` 中声明的智能运维改造，以及当前工作区相关后端、前端、测试与文档改动。

## 结论摘要

当前改动不是单纯“换皮”或纯占位，已经落下了一批真实功能：资产 CRUD、JSON 持久化、运维大屏汇总 API、告警组持久化 API、部分 Agent 工具真实调用、前端大屏/资产/告警页面接 API、RBAC 基础权限接线等。

但它也还没有达到 `docs/1.md` 所描述的“专业企业级大数据运维平台”完成态。更准确的状态是：**MVP 骨架 + 部分真实数据接入 + 较多固定规则/本地文件存储/Prompt 驱动逻辑**。部分路线图勾选偏乐观，尤其是 E2E、前端测试、按集群真实巡检、企业级降噪、CMDB 深度集成、按钮级权限闭环和视觉 token 化。

需要特别注意的粉饰风险有三类：

1. 巡检分数在前端从 Agent 文本正则解析，解析不到时默认 90 分，容易把未知结果展示成健康。
2. `query_vm_metrics` 未配置监控地址时默认请求 `http://localhost:8428`，不是明确失败，可能让环境问题被误判为本机监控查询失败。
3. 文档与任务表大量标记 `[x]`，但自动化验证并未通过，且 E2E 文档只是清单，不是执行结果。

## 已经真实实现的部分

### 1. 集群资产管理有真实后端和前端接线

后端新增 `src/pkg/ops`，通过 `{stateDir}/ops/clusters.json` 存储集群资产。`InitStore`、`ListClusters`、`CreateCluster`、`PatchCluster`、`DeleteCluster` 等能力存在，并做了基本校验，例如名称不能为空、domain/status 合法性、节点数非负等。

证据：

- `src/pkg/ops/service.go:20` 初始化本地资产 store。
- `src/pkg/ops/service.go:34` 支持按 domain 过滤集群。
- `src/pkg/ops/service.go:63` 新增集群并校验基础字段。
- `src/pkg/gateway/http/ops_api.go:23` 到 `src/pkg/gateway/http/ops_api.go:125` 暴露资产列表、详情、新增、更新、CMDB 同步等 API。
- `ui/src/ui/controllers/ops-clusters.ts` 调用 `/api/ops/clusters`、`/api/ops/dashboard/summary`、`/api/ops/clusters/sync-cmdb`。
- `ui/src/ui/views/asset-management.ts` 渲染资产列表、空状态、新增表单和 CMDB 同步按钮。

判断：真实实现，但属于本地 JSON store 级别，尚不是企业级 CMDB/数据库级资产中心。

### 2. 运维大屏不再纯硬编码，已经接汇总 API

大屏指标来自 `BuildDashboardSummaryWithContext`，会聚合已登记集群状态，并尝试用 VictoriaMetrics/Prometheus 查询业务域健康分。

证据：

- `src/pkg/ops/service.go:210` 聚合纳管集群、健康/亚健康/异常数量。
- `src/pkg/ops/service.go:248` 固定输出五个业务域顺序。
- `src/pkg/ops/vm_health.go` 实现 instant PromQL 查询与健康分归一化。
- `src/pkg/gateway/http/ops_api.go:149` 暴露 `/api/ops/dashboard/summary`。
- `ui/src/ui/views/overview.ts` 根据 `dashboardSummary` 渲染统计卡、域健康卡和空状态。

判断：不是假大屏，但健康分仍是非常粗的代表性查询，不是按 CMDB 集群标签、组件和 SLA 模型严谨计算。

### 3. 告警组持久化和查询 API 已经实现

告警 hook 会按 source 聚合窗口，触发 Agent 分析前会写入 `alerts.json`；前端告警 Tab 通过 API 获取告警组、统计降噪比并可标记处理。

证据：

- `src/pkg/gateway/http/hooks.go:345` 到 `src/pkg/gateway/http/hooks.go:408` 按 `source` 做 15 秒/20 条滑动窗口聚合。
- `src/pkg/ops/alerts_service.go:59` 持久化合并后的告警组。
- `src/pkg/ops/alerts_service.go:111` 统计 `originalTotal`、`mergedTotal`、`reductionRate`。
- `src/pkg/gateway/http/ops_api.go:157` 到 `src/pkg/gateway/http/ops_api.go:209` 暴露告警组查询、详情和 PATCH。
- `ui/src/ui/controllers/ops-alerts.ts` 接 `/api/ops/alerts/groups` 和 PATCH。

判断：有真实 MVP；但降噪算法只是按 `source` 合并，不是企业告警降噪常见的指纹、标签、实例、拓扑、时间因果和抑制规则。

### 4. 部分 Agent 工具从模拟改成真实调用

`query_gbase_slow_sql`、`query_governance_lineage`、`query_hadoop_jmx`、`query_fi_manager_metrics` 均是实际 DB/HTTP 调用；未配置时多数工具会明确失败，不返回模拟数据。

证据：

- `src/pkg/agent/tools/direct_plugins.go` 中 GBase 使用 DSN 查询慢 SQL，治理平台调用 HTTP API。
- `src/pkg/agent/tools/hadoop_jmx.go` 查询 Hadoop JMX HTTP。
- `src/pkg/agent/tools/fi_manager.go` 查询 FI Manager HTTP。
- `src/pkg/agent/tools/bridge.go` 将这些工具注册进默认工具集。

判断：不是假数据；但多数工具只是通用接口封装，还没有和资产模型、集群 endpoint、凭据、租户、证书、组件实例自动绑定。

### 5. RBAC 基础权限接线存在

后端新增 `ops:inspect`、`ops:diagnose`、`ops:ack` 等权限；资产写操作走 `menu:config`，告警确认走 `ops:ack`。

证据：

- `src/pkg/rbac/db.go:141` 到 `src/pkg/rbac/db.go:145` 定义 ops 权限。
- `src/pkg/gateway/http/server.go:444` 到 `src/pkg/gateway/http/server.go:454` 对 ops API 套用 RBAC/Gateway Token 权限。
- `ui/src/ui/ops/rbac.ts` 提供前端按钮级权限 helper。

判断：权限有基础实现；但 Gateway Token 仍等同全权限，且前端 `user == null` 时默认放行，不能视为严格企业权限闭环。

## 部分实现但尚未完成的部分

### 1. “全局环境/集群选择器”只影响聊天上下文，不影响巡检任务和工具参数

前端按 domain 拉取集群并构建实体选择器，也会在聊天消息前追加 `[运维上下文]` 行。

证据：

- `ui/src/ui/ops/entity-config.ts` 从集群构造 domain/cluster/component 选项。
- `ui/src/ui/app-chat.ts:316` 到 `ui/src/ui/app-chat.ts:318` 在聊天消息前追加上下文。

问题：

- 默认 Cron 巡检任务仍是固定 Prompt 和固定 PromQL，没有读取当前选择的集群 ID、组件或实例标签。
- Agent 工具没有根据 `clusterId` 自动选择 VM label、JMX URL、FI Manager URL、GBase DSN。
- 因此“左侧选场景 -> 右侧圈定集群 -> 下方结果按集群渲染”的业务动线只在 UI 和 Chat 层成立，不在巡检执行链路成立。

### 2. 深度巡检仍主要依赖 Prompt 和文本解析

五个默认巡检任务确实存在，但任务内容是写死的 Prompt 与 PromQL 文本。

证据：

- `src/pkg/cron/service.go:496` 到 `src/pkg/cron/service.go:590` 创建五个默认巡检任务。
- `ui/src/ui/app-render.ts:3093` 默认 `score = 90`，再从 summary 文本中正则匹配健康分。

问题：

- 健康分没有结构化结果 schema。
- Agent 未按格式输出时，前端默认 90 分，偏向“看起来健康”。
- 巡检失败、工具未配置、VM 不通等情况容易被折叠进自然语言 summary，而不是结构化失败态。
- 目前没有按集群/组件的巡检历史维度，只有按 job 的 cron runs。

### 3. “CMDB 同步”只是 HTTP 拉取或请求体导入

`sync-cmdb` 能从 `OPS_CMDB_SYNC_URL` 拉 JSON 或从请求体导入，并按 domain + name upsert。

判断：这是一个可用的导入入口，不是完整 CMDB 集成。

缺口：

- 没有字段映射配置、分页、增量同步、删除/下线同步、冲突审计。
- 没有 CMDB 数据质量校验和错误行详情。
- 没有和监控标签、凭据、endpoint 关联。

### 4. IM / ChatOps 是接线级实现，未证明端到端可用

`/help`、`/ack`、`/diagnose` 命令解析存在，IM 回复通过 `InvokeMethod("send")` 发出。

问题：

- 是否能在飞书/钉钉/企微各通道稳定收发，取决于对应 runtime 和 send handler，不是本次代码能单独证明。
- `docs/e2e-ops-smoke.md` 是人工清单，未包含自动化执行结果。
- IM 卡片实际是普通文本体，离企业 IM 交互卡片、按钮回调、签名校验还有差距。

## 明确未实现或仍是占位的事项

1. 集群实时状态刷新 API 未实现。前端 `刷新状态` 按钮永久 disabled，提示“集群状态 API 尚未接入”，见 `ui/src/ui/views/tech-ops-domain.ts:77` 和 `ui/src/ui/views/tech-ops-domain.ts:192` 到 `ui/src/ui/views/tech-ops-domain.ts:202`。

2. 巡检任务没有按当前选择的集群/组件执行。选择器只影响 Chat 上下文，不影响 `job-inspect-*` 的 Prompt、PromQL label 或工具 endpoint。

3. 企业级告警降噪算法未实现。当前只按 `source` 聚合，不支持 alert fingerprint、labels、instance、service、topology、依赖链、抑制、静默、升级策略。

4. 告警生命周期不完整。只有 `active/analyzing/resolved`，没有认领人、处理备注、关闭原因、SLA、升级、通知重试、审计日志。

5. Root Cause 不是结构化产物。它从 Agent 会话 transcript 读取最后一条 assistant Markdown，缺少分析状态、失败状态、工具证据和引用结构。

6. 资产台账没有编辑/删除前端入口。后端有 PATCH/DELETE，前端资产页只有新增、刷新、同步 CMDB 和列表展示。

7. 监控健康分没有按真实业务模型实现。`domainVMQueries` 是固定代表性查询，无法覆盖企业环境指标命名差异、集群标签、组件权重和 SLA。

8. GBase 默认慢 SQL 表和字段是假定结构。`information_schema.slow_query_log` 不一定存在或字段一致，需要目标版本适配。

9. 治理平台 API 只是通用 GET 封装。没有定义返回 schema、血缘实体、质量规则、项目/租户过滤和错误分级。

10. Hadoop JMX / FI Manager 工具没有从资产中自动取 endpoint，也没有多集群、多组件、多凭据管理。

11. P4 E2E 自动化未完成。`docs/e2e-ops-smoke.md` 是手工 checklist，不是可运行自动化。

12. 前端视觉 token 化未完成。新业务页仍大量内联 style，见 `ui/src/ui/views/tech-ops-domain.ts:93`、`:146`、`:166`、`:177`、`:192`、`:196`。

13. 前端测试当前不通过。`npm run test` 失败 10 个测试，不能声明“前端渲染验证完成”。

14. 全量 Go 测试当前不通过。失败在 `test/gateway_protocol_test.go` 的网关 token 认证用例。

15. `query_vm_metrics` 未配置时默认打本机 `localhost:8428`，见 `src/pkg/agent/tools/vm_query.go:79` 到 `src/pkg/agent/tools/vm_query.go:81`。这与“禁止假数据/诚实未配置”的原则不一致。

16. 前端权限 helper 在 `user == null` 时默认允许操作，可能导致登录态未加载时按钮短暂可用。

## 质量评价

### 优点

1. 改动方向是合理的：从通用 Agent 控制台向业务域、资产、告警、巡检、ChatOps 聚合，符合 `docs/1.md` 的产品目标。
2. 后端新增模块边界较清楚：`ops` 包集中资产、告警、CMDB 同步、IM 状态和深链逻辑。
3. “无数据空状态”比原先硬编码展示更诚实，大屏和资产页没有继续显示“12 集群 / 85 分”一类固定生产假象。
4. GBase/治理/JMX/FI 工具至少改为真实外部调用，不再静默返回模拟结果。
5. 告警和资产都有基本单元测试，新增包 `go test ./pkg/ops ./pkg/agent/tools` 通过。

### 主要问题

1. 文档勾选过于乐观。`docs/task.md`、`docs/ops-roadmap.md` 多数项目标 `[x]`，但测试和 E2E 证据不足。
2. 架构还停留在本地文件 MVP。资产、告警、审计、权限、运行历史都未达到企业多用户/多实例/高可靠要求。
3. 业务闭环不够深。UI 上有业务域和集群，但工具执行、监控查询、巡检报告并没有真正按这些业务实体组织。
4. 健康分和降噪指标可信度不足。健康分依赖固定 PromQL 或文本解析，降噪比依赖 source 批量合并。
5. 前端质量有回归。Vitest 当前失败 10 个，说明导航、渲染和部分组件契约被破坏或测试未同步。
6. UI 实现仍混杂大量内联样式，与“Cursor 风格 token 化”的工程质量目标不一致。

## 验证结果

已执行：

```bash
cd src && go test ./pkg/ops ./pkg/agent/tools ./pkg/gateway/http
```

结果：

- `pkg/ops` 通过。
- `pkg/agent/tools` 通过。
- `pkg/gateway/http` 无测试文件。

已执行：

```bash
cd ui && npm run build
```

结果：构建通过，但有大 chunk 警告。

已执行：

```bash
cd ui && npm run test
```

结果：失败。首次在沙箱内因 Vitest 监听 `::1` 被拒绝；提权后正常运行，但 35 个测试文件中 6 个失败，共 10 个测试失败。失败点包括：

- `src/ui/navigation.test.ts`：`titleForTab("chat")` 期望 `Chat`，实际 `Control`。
- `src/ui/navigation.browser.test.ts`：Shell/content/top-tab 等渲染选择器为空或路由断言失败。
- `src/ui/focus-mode.browser.test.ts`：`.shell` 未渲染。
- `src/ui/chat-markdown.browser.test.ts`：工具输出卡片未找到。
- `src/ui/views/catalog-pages.test.ts`：模型库布局断言失败。
- `src/ui/views/symbol-icon-buttons.test.ts`：Agent Swarm 缺省 props 报错、部分按钮图标断言失败。

已执行：

```bash
cd src && go test ./...
```

结果：失败。新增 `pkg/ops`、`pkg/agent/tools` 通过，但 `github.com/openocta/openocta/test` 中 `TestGatewayProtocol_*` 因 `invalid_gateway_token` 失败。

## 当前完成度判断

| 模块 | 完成度 | 判断 |
|---|---:|---|
| 运维大屏 | 60% | 已接资产/告警/VM 汇总 API，但健康模型粗糙，缺少真实业务 SLA 与集群维度 |
| 资产管理 | 55% | CRUD/API/store 基本可用，前端缺编辑删除，CMDB 只是导入入口 |
| 环境选择器 | 45% | UI 和 Chat 上下文完成，未贯穿巡检和工具执行 |
| 深度巡检 | 40% | 默认任务和工具存在，但结果非结构化、按文本解析、无集群绑定 |
| 告警降噪 | 45% | 有 hook、持久化、列表和 ack；算法只是 source 窗口合并 |
| ChatOps / IM | 35% | 命令解析和发送接线存在，缺端到端联调证据和交互卡片闭环 |
| RBAC | 55% | 基础权限存在，但 Gateway Token 全权限和前端空用户放行削弱边界 |
| 前端体验 | 50% | 业务结构改善明显，但测试失败、内联样式多、部分按钮占位 |
| 自动化验证 | 30% | 后端局部通过，前端和全量 Go 不通过，E2E 只是清单 |

## 优先整改建议

### P0：先停止“完成态”误判

1. 把 `docs/task.md` 和 `docs/ops-roadmap.md` 中未验证通过的 `[x]` 改为 `[/]` 或补充“代码已接线 / 未 E2E”状态。
2. 修复 `npm run test` 的 10 个失败测试。
3. 修复或更新 `go test ./...` 中 `test/gateway_protocol_test.go` 的 token 认证失败。
4. 将巡检分数解析不到时改成 `unknown`，不要默认 90 分。
5. `query_vm_metrics` 未配置 VM/Prometheus 时应明确返回未配置错误，而不是默认 localhost。

### P1：让业务实体真正进入执行链路

1. 为 Cluster 增加监控 label、JMX endpoint、FI endpoint、GBase DSN 引用、凭据引用等字段。
2. 巡检任务运行时注入当前 domain/cluster/component，并将工具调用限制到该上下文。
3. 巡检输出改成结构化 JSON + Markdown 双产物，健康分、指标、异常、证据、工具错误分开存储。
4. 前端巡检列表按 domain + cluster/component 展示历史，而不是只按 job 展示。

### P2：补齐企业级运维闭环

1. 告警降噪改为 fingerprint/labels/source/component 多维聚合，保留 source 窗口合并作为兜底。
2. 告警组增加 assignee、ack note、resolved reason、timeline、notification status、audit log。
3. Root Cause 保存结构化分析结果，包含工具证据、查询语句、失败原因和重试状态。
4. IM 卡片使用飞书/钉钉/企微原生卡片能力，支持按钮回调 ack/diagnose。

### P3：提高 UI 与交付质量

1. 清理业务新页面内联 style，迁入 `ops-*.css` token class。
2. 资产页补编辑、删除、详情、批量同步错误详情。
3. 给 `刷新状态` 做真实 API 或隐藏该按钮，避免“路线图 P1 尚未接入”的露馅文案出现在产品页。
4. 把 `docs/e2e-ops-smoke.md` 转成可运行脚本或 Playwright E2E，并把执行结果写入报告。

## 最终判断

当前逻辑方向合理，改动质量属于“能演示的工程 MVP”，不是纯假数据粉饰。但文档和勾选已经超过了代码真实完成度。若要面向企业试用，至少需要先修复测试失败、去掉健康分默认值、让集群上下文贯穿巡检和工具调用，并把告警降噪从 source 批处理升级为可解释的事件聚类模型。
