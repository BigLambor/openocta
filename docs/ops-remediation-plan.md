# OpenOcta AIOps 整改计划与验收标准

创建日期：2026-06-02  
依据：[ops-implementation-review.md](./ops-implementation-review.md)  
目标：把当前“能演示的 AIOps MVP”推进到“可信、可验证、可继续扩展”的企业运维基础版本。

## 总原则

1. **先可信，再扩展**：先修复假健康分、测试失败、文档过度勾选，再继续做新能力。
2. **先结构化，再智能化**：巡检、告警、根因分析必须有结构化结果，不能只依赖 Agent 自然语言。
3. **上下文必须贯穿执行链路**：UI 选择的业务域、集群、组件必须影响 PromQL、JMX、GBase、FI Manager 等实际查询目标。
4. **未接入就明确未接入**：不使用默认分数、默认本地监控地址、演示数据、隐式成功文案包装未知状态。
5. **验收以命令和行为为准**：每个阶段必须有可执行验证命令或可复现 UI/API 行为。

## P0：可信度止血

目标：消除最明显的“粉饰太平”风险，恢复文档与代码真实状态一致。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P0-1 | 修正任务文档状态 | `docs/task.md`、`docs/ops-roadmap.md` | 未经测试/E2E 验证的项目不再标 `[x]`；每个进行中项注明“已接线/未验证/未实现”的真实状态 |
| P0-2 | 巡检分数解析不到时展示未知 | `ui/src/ui/app-render.ts`、`ui/src/ui/views/tech-ops-domain.ts` | Agent summary 无 `健康得分` 时，UI 显示“未生成健康分/未知”，不显示 90/100 或健康态 |
| P0-3 | VM 未配置时明确失败 | `src/pkg/agent/tools/vm_query.go` | 未配置 `VICTORIAMETRICS_URL` 和 `PROMETHEUS_URL` 时，`query_vm_metrics` 返回明确未配置错误，不请求 `localhost:8428` |
| P0-4 | 权限未加载时不默认放行 | `ui/src/ui/ops/rbac.ts`、相关按钮调用处 | `rbacUser == null` 时，涉及 `ops:inspect`、`ops:ack`、`ops:diagnose` 的按钮默认 disabled 或 loading，不短暂显示可操作 |
| P0-5 | 清理产品页占位按钮 | `ui/src/ui/views/tech-ops-domain.ts` | “刷新状态”若无 API，则隐藏或改为清晰空状态操作，不出现“路线图 P1 尚未接入”这类内部文案 |

### P0 验收命令

```bash
cd src && go test ./pkg/agent/tools ./pkg/ops
cd ui && npm run build
```

### P0 手工验收

- 未配置 VM 时发起巡检，前端不出现默认健康分。
- 退出登录或 RBAC 未加载时，巡检/确认告警按钮不可操作。
- 业务域页面不显示内部路线图文案。

## P1：自动化测试基线恢复

目标：让当前工作区有可信回归基线，避免继续在失败测试上叠功能。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P1-1 | 修复前端导航标题与路由测试 | `ui/src/ui/navigation.ts`、`ui/src/ui/app-render.ts`、相关测试 | `src/ui/navigation.test.ts`、`src/ui/navigation.browser.test.ts` 全部通过 |
| P1-2 | 修复 Shell/Chat 渲染测试 | `ui/src/ui/app-render.ts`、`ui/src/ui/views/chat.ts` | `focus-mode.browser.test.ts`、`chat-markdown.browser.test.ts` 通过 |
| P1-3 | 修复 catalog/model/swarm 测试 | `ui/src/ui/views/model-library.ts`、`ui/src/ui/views/agent-swarm.ts`、相关测试 | `catalog-pages.test.ts`、`symbol-icon-buttons.test.ts` 通过 |
| P1-4 | 修复 Go 全量测试的 gateway token 失败 | `src/test/gateway_protocol_test.go` 或认证逻辑相关文件 | `cd src && go test ./...` 通过；若是测试预期过期，更新测试并说明原因 |
| P1-5 | 建立整改验证清单 | `docs/ops-remediation-plan.md`、`docs/e2e-ops-smoke.md` | 文档列出每次整改必须执行的最小命令集 |

### P1 验收命令

```bash
cd ui && npm run test
cd ui && npm run build
cd src && go test ./...
```

通过标准：三条命令全部成功。

## P2：集群上下文贯穿执行链路

目标：让“选业务域/集群/组件”不只改变 UI 文案，而是真正决定巡检和工具查询范围。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P2-1 | 扩展 Cluster 模型 | `src/pkg/ops/cluster.go`、`src/pkg/ops/service.go`、`docs/ops-api.md` | Cluster 支持 `monitorLabels`、`vmUrlRef` 或 `metricsBaseUrl`、`jmxUrl`、`fiManagerUrl`、`gbaseDsnRef`、`credentialsRef` 等字段；旧数据可兼容加载 |
| P2-2 | 前端资产页支持关键执行配置 | `ui/src/ui/views/asset-management.ts`、`ui/src/ui/controllers/ops-clusters.ts` | 可新增/编辑集群监控标签、JMX URL、FI Manager URL、GBase DSN 引用；敏感值不明文展示 |
| P2-3 | 巡检运行参数携带上下文 | `ui/src/ui/controllers/ops-inspection-run.ts`、`src/pkg/cron/service.go`、cron handler | 点击某集群的一键巡检时，后端收到 domain、clusterId、component，并写入运行记录 |
| P2-4 | 工具按集群解析目标 | `src/pkg/agent/tools/*.go`、`src/pkg/ops` | `query_vm_metrics` 自动拼接选中集群 labels；`query_hadoop_jmx`、`query_fi_manager_metrics`、`query_gbase_slow_sql` 可从集群配置取目标 |
| P2-5 | 大屏健康分按资产标签计算 | `src/pkg/ops/vm_health.go` | 不再只用固定 domain 查询；至少能按 cluster 的 monitorLabels 计算单集群健康分，再聚合到 domain |

### P2 验收标准

- 新增两个同 domain 不同集群，配置不同监控 label。
- 分别选择两个集群运行巡检，实际 PromQL 查询包含不同 label。
- Hadoop 集群配置不同 JMX URL 后，工具请求目标不同。
- GBase 集群无 DSN 引用时，巡检显示该工具未配置，不生成成功态。

### P2 验收命令

```bash
cd src && go test ./pkg/ops ./pkg/agent/tools ./pkg/cron
cd ui && npm run test
```

## P3：巡检结果结构化

目标：巡检结果从“Agent 文本 + 正则猜分”升级为“结构化结果 + Markdown 报告”。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P3-1 | 定义巡检结果 schema | `src/pkg/ops`、`docs/ops-api.md` | schema 包含 `score`、`scoreStatus`、`toolRuns`、`metricsEvidence`、`errors`、`reportMarkdown`、`startedAt/finishedAt` |
| P3-2 | 后端持久化巡检结果 | `src/pkg/ops`、cron run 记录相关代码 | 巡检完成后写结构化结果；失败也记录失败原因和工具错误 |
| P3-3 | Agent 输出解析与兜底 | cron delivery / chat send 相关代码 | Agent 未按 schema 输出时，标记 `scoreStatus=unknown`，不猜分 |
| P3-4 | 前端按结构化字段渲染 | `ui/src/ui/views/tech-ops-domain.ts`、controllers | 健康分、工具证据、错误、Markdown 报告分区展示；工具失败有明确错误态 |
| P3-5 | 低分 IM 推送基于结构化 score | `src/pkg/gateway/handlers/cron_delivery.go` | IM 低分判断不再用文本正则；无 score 时不发送“低分”告警，只发送失败/未知状态 |

### P3 验收标准

- 正常巡检：UI 显示结构化健康分、指标证据和 Markdown 报告。
- 工具失败：UI 显示失败工具、错误信息，健康分为 unknown 或 degraded，不显示健康。
- Agent 输出缺少健康分：系统不默认 90 分。

### P3 验收命令

```bash
cd src && go test ./pkg/ops ./pkg/gateway/handlers ./pkg/cron
cd ui && npm run test
```

## P4：告警降噪与处理闭环升级

目标：把当前按 source 合并的告警 MVP 升级为可解释、可处理、可审计的企业告警组。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P4-1 | 定义告警指纹和归并键 | `src/pkg/ops/alerts.go`、`src/pkg/gateway/http/hooks.go` | 支持从 labels 中提取 `alertname`、`service`、`instance`、`clusterId`、`component`；默认 fingerprint 可配置 |
| P4-2 | 聚合算法从 source 升级为 fingerprint + 时间窗 | `src/pkg/gateway/http/hooks.go`、`src/pkg/ops/alerts_service.go` | 同 source 不同 service 不会被错误合并；同 service 同 fingerprint 在窗口内合并 |
| P4-3 | 告警生命周期补齐 | `src/pkg/ops/alerts.go`、`ops_api.go`、前端告警页 | 支持 assignee、ack note、resolved reason、timeline、updatedBy、audit events |
| P4-4 | Root Cause 结构化保存 | `src/pkg/ops/alerts_service.go` | 保存分析状态、工具证据、根因摘要、影响范围、建议动作，不只读 transcript 最后一条 |
| P4-5 | IM 卡片动作闭环 | 通道发送/回调相关代码、`inbound_sink.go` | 飞书/钉钉卡片支持“确认/诊断/打开详情”；按钮动作能回写告警组 |

### P4 验收标准

- 连续发送同 source 不同 service 的告警，生成不同告警组。
- 连续发送同 fingerprint 的告警，合并为一个告警组，原始事件数递增。
- Web 端确认告警时必须填写处理备注或关闭原因。
- ChatOps `/ack <id>` 写入 timeline，而不是只改状态。

### P4 验收命令

```bash
cd src && go test ./pkg/ops ./pkg/gateway/http
cd ui && npm run test
```

## P5：CMDB 与部署交付增强

目标：让资产同步和交付具备企业环境可维护性。

| ID | 任务 | 涉及文件 | 验收标准 |
|---|---|---|---|
| P5-1 | CMDB 字段映射配置 | `src/pkg/ops/cmdb_sync.go`、配置 schema、docs | 支持外部字段名映射到 OpenOcta Cluster 字段 |
| P5-2 | CMDB 同步错误详情 | `src/pkg/ops/cmdb_sync.go`、资产页 | 同步响应包含每条失败原因；前端可展开查看 |
| P5-3 | 下线/删除同步策略 | `src/pkg/ops/cmdb_sync.go` | 支持 dry-run、mark-inactive、delete 三种策略，默认不删除 |
| P5-4 | 生产部署安全检查 | `docs/deploy-ops.md`、启动检查代码可选 | 未配置 CORS 白名单、默认 admin 密码、明文敏感字段时给出明确告警 |
| P5-5 | 自动化 E2E | `docs/e2e-ops-smoke.md`、新增脚本或 Playwright 测试 | 登录 -> 建集群 -> 选集群 -> 巡检 -> 告警 -> ack 路径可自动执行 |

### P5 验收命令

```bash
cd src && go test ./...
cd ui && npm run test
cd ui && npm run build
```

## 统一验收门槛

每个 P 阶段完成前，必须满足：

1. 本阶段表格中的验收标准全部可复现。
2. 新增或修改行为有对应单元测试，跨 UI/API 的关键路径有浏览器或 E2E 测试。
3. `docs/task.md` 只在验证通过后标 `[x]`。
4. 未完成能力在 UI 中显示为未配置、不可用或空状态，不显示默认成功态。
5. 不引入新的业务假数据；示例数据只能出现在文档示例或测试 fixture。

## 最小执行顺序

建议按以下顺序推进，避免在不可信基线上继续扩展：

1. P0-2：巡检分数 unknown 化。
2. P0-3：VM 未配置明确失败。
3. P1-1 到 P1-4：恢复测试基线。
4. P2-1 到 P2-4：上下文贯穿执行链路。
5. P3-1 到 P3-5：巡检结果结构化。
6. P4-1 到 P4-5：告警降噪与生命周期。
7. P5-1 到 P5-5：CMDB 和交付。

## 当前状态快照

基于 2026-06-02 审查：

| 验证项 | 当前状态 | 处理归属 |
|---|---|---|
| `cd src && go test ./pkg/ops ./pkg/agent/tools ./pkg/gateway/http` | 通过 | 维持 |
| `cd ui && npm run build` | 通过，有 chunk 警告 | P1 后复测 |
| `cd ui && npm run test` | 失败，10 个测试失败 | P1 |
| `cd src && go test ./...` | 失败，gateway token 测试失败 | P1 |
| 巡检无分数默认 90 | 存在 | P0 |
| VM 未配置默认 localhost | 存在 | P0 |
| 集群选择器影响巡检目标 | 未完成 | P2 |
| 告警按 fingerprint 降噪 | 未完成 | P4 |
| E2E 自动化 | 未完成 | P5 |

## 文档关系

| 文档 | 用途 |
|---|---|
| [1.md](./1.md) | 产品动机与改造背景 |
| [ops-implementation-review.md](./ops-implementation-review.md) | 当前实现审查与问题清单 |
| [ops-remediation-plan.md](./ops-remediation-plan.md) | 下一步整改任务与验收标准 |
| [task.md](./task.md) | 简短进度跟踪，应随本计划验证结果更新 |
| [ops-roadmap.md](./ops-roadmap.md) | 历史路线图，需按本计划校正完成状态 |
