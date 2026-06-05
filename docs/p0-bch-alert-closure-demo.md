# P0 BCH 告警闭环演示验收脚本

> 日期：2026-06-05  
> 范围：验证 BCH 告警从工作台进入 AI 分析、确认/驳回建议，并写入执行记录。

## 1. 前置条件

1. 前端已构建通过：`cd ui && npm run build`
2. 后端相关测试通过：`cd src && go test ./pkg/gateway/handlers ./pkg/ops ./pkg/employees`
3. 用户具备查看工作台、确认告警和访问执行记录的权限。
4. `/api/ops/alerts/groups?domain=hadoop` 至少返回一条 BCH/Hadoop 告警组。

## 2. 演示路径

### 2.1 进入工作台

1. 登录 OpenOcta。
2. 点击顶栏“运维工作台”。
3. 确认页面展示“事件中心”和“BCH 告警列表”。

验收点：

- 顶栏存在“AI 运维助手 / 运维驾驶舱 / 运维工作台 / 服务与资产”。
- 工作台能展示 BCH 告警组、原始告警数、降噪后告警组数和根因候选。

### 2.2 AI 分析与建议

1. 在 BCH 告警列表选择一条告警组。
2. 点击“分析根因”。
3. 确认右侧 AI 面板打开，展示“BCH 值班数字员工”模板、结论和证据。
4. 点击“处置建议”，确认建议中区分只读检查、Runbook 和需要审批的动作。

验收点：

- AI 能力嵌入工作台，而不是跳到独立“数字员工中心”。
- 数字员工以专家人设和技能组合包出现。
- 输出内容包含根因候选、影响范围或下一步动作。

### 2.3 发送到全局 AI 运维助手

1. 在告警详情点击“发送到 AI 运维助手”。
2. 系统切换到顶栏“AI 运维助手”。
3. 确认发送的问题携带当前告警标题。
4. 后端 `chat.send` 应注入以下上下文：
   - `domain=hadoop`
   - `capability=observability-alert`
   - `objectType=alert`
   - `objectId=<告警组 ID>`
   - BCH 值班数字员工模板或 manifest 中匹配到的模板

验收点：

- AI 运维助手仍作为全局 Copilot 保留。
- 从工作台进入 Copilot 时，模型具备告警上下文和数字员工人设。

### 2.4 确认 AI 建议

1. 回到“运维工作台”。
2. 打开同一告警的 AI 面板。
3. 点击“确认建议并记录”。
4. 填写处理备注。
5. 确认告警组被标记为已处理。
6. 点击“查看执行记录”。

验收点：

- 必须填写处理备注，否则不能确认。
- 告警 PATCH 为 `resolved`。
- 新增一条 `employee.tasks` 执行记录。
- 记录包含：
  - `employeeId=builtin-bch-oncall`
  - `domainKey=hadoop`
  - `capabilityKey=observability-alert`
  - `objectRef=<告警组 ID>`
  - `evaluation=accepted`
  - `workflowStatus=closed`
  - `rawAlertCount/reducedAlertCount/savedHours`

### 2.5 驳回 AI 建议

1. 选择另一条告警组。
2. 点击“分析根因”。
3. 点击“驳回建议并记录”。
4. 填写驳回原因。
5. 点击“查看执行记录”。

验收点：

- 必须填写驳回原因，否则不能驳回。
- 告警保持或恢复为 `active`。
- 新增一条 `employee.tasks` 执行记录。
- 记录包含：
  - `evaluation=rejected`
  - `workflowStatus=rejected`
  - 驳回原因写入 `output/conclusion`

## 3. 自动验证

已覆盖的自动验证：

```bash
cd src
go test ./pkg/gateway/handlers ./pkg/ops ./pkg/employees

cd ../ui
npm run build
```

新增单测：

- `src/pkg/gateway/handlers/employee_tasks_test.go`
- 覆盖 BCH 告警 AI 建议确认后写入执行记录，并校验领域、能力、评价和降噪指标。

## 4. 通过标准

P0 通过需要同时满足：

1. 工作台能展示 BCH 告警列表。
2. 告警详情能打开 AI 分析/建议面板。
3. 可把当前告警带上下文发送到全局 AI 运维助手。
4. 可确认 AI 建议并关闭告警。
5. 可驳回 AI 建议并保留告警待处理。
6. 确认和驳回都会生成可追溯执行记录。
7. 自动化效果页能基于执行记录统计采纳率、降噪率等指标。
