# Ops 集群资产 API

存储路径：`{OPENOCTA_STATE_DIR}/ops/clusters.json`（随网关启动由 `ops.InitStore` 加载）。

认证：与网关其他 API 相同，支持 **RBAC Bearer Token** 或 **Gateway Token**。  
写操作（POST/PATCH/DELETE）需要 `menu:config` 权限（`admin` 角色豁免）。

## 集群 CRUD

### `GET /api/ops/clusters`

查询参数：

| 参数 | 说明 |
|------|------|
| `domain` | 可选，过滤业务域：`hadoop` \| `fi` \| `gbase` \| `governance` \| `dataapps` |

响应：

```json
{
  "clusters": [ { "id": "cluster-...", "name": "...", "domain": "hadoop", ... } ],
  "total": 1
}
```

### `GET /api/ops/clusters/{id}`

返回单个集群；不存在时 `404`。

### `POST /api/ops/clusters`

请求体示例：

```json
{
  "name": "北京 BCH 生产",
  "domain": "hadoop",
  "region": "北京",
  "nodeCount": 120,
  "components": ["HDFS", "YARN", "HIVE"],
  "owner": "张三",
  "status": "healthy",
  "description": "",
  "monitorLabels": "cluster=\"bj-bch-prod\",env=\"prod\"",
  "vmUrlRef": "VICTORIAMETRICS_URL_BJ_BCH",
  "metricsBaseUrl": "https://vm.example.com",
  "jmxUrl": "https://rm.example.com/jmx",
  "fiManagerUrl": "",
  "gbaseDsnRef": "",
  "credentialsRef": "secret://ops/bj-bch-prod"
}
```

`status`：`healthy` \| `warning` \| `critical` \| `unknown` \| `inactive`（默认可省略为 `unknown`）。

执行配置字段说明：

| 字段 | 说明 |
|------|------|
| `monitorLabels` | 注入 PromQL 的标签片段，例如 `cluster="prod-a",env="prod"` |
| `vmUrlRef` | VictoriaMetrics/Prometheus URL 的环境变量引用名 |
| `metricsBaseUrl` | 直接配置的指标查询地址；优先级高于 `vmUrlRef` |
| `jmxUrl` | Hadoop JMX 查询地址 |
| `fiManagerUrl` | FI Manager 指标 API 地址 |
| `gbaseDsnRef` | GBase DSN 的环境变量或密钥引用名 |
| `credentialsRef` | 企业密钥系统中的凭据引用，不建议写明文密码 |

### `PATCH /api/ops/clusters/{id}`

部分更新，字段均为可选，例如：`{"status": "warning"}`。

### `DELETE /api/ops/clusters/{id}`

响应：`{"ok": true}`。

### `POST /api/ops/clusters/sync-cmdb`

从 CMDB 合并纳管集群（P1-2）。需要 `menu:config`。

**方式一（推荐）**：配置环境变量 `OPS_CMDB_SYNC_URL`，POST 空 body `{}`，网关对该 URL 发起 `GET` 并解析 JSON。

- 可选：`OPS_CMDB_SYNC_TOKEN` 作为 `Authorization: Bearer` 发往 CMDB
- 响应格式：`{"clusters":[...]}` 或 `[...]` 数组
- `components` 可为字符串数组或逗号分隔字符串

**方式二**：请求体直接携带行：

```json
{
  "strategy": "upsert",
  "clusters": [
    {
      "name": "北京 BCH 生产",
      "domain": "hadoop",
      "region": "北京",
      "nodeCount": 120,
      "components": ["HDFS", "YARN"],
      "owner": "张三",
      "status": "healthy"
    }
  ]
}
```

`strategy` 可选：

| 值 | 语义 |
|----|------|
| `upsert` | 默认策略；按 **业务域 + 集群名**（不区分大小写）创建或更新，不处理本地缺失项 |
| `dry-run` | 只校验输入并预估 `created/updated/skipped`，不创建、不更新、不删除 |
| `mark-inactive` | 先 upsert，再把本地存在但本次 CMDB feed 缺失的集群置为 `inactive` |
| `delete` | 先 upsert，再删除本地存在但本次 CMDB feed 缺失的集群 |

请求体也可传 `mapping` 覆盖外部字段名：

```json
{
  "strategy": "dry-run",
  "mapping": {
    "name": "ext_cluster_name",
    "domain": "ext_domain",
    "region": "ext_region",
    "nodeCount": "ext_node_count",
    "components": "ext_components",
    "owner": "ext_owner",
    "status": "ext_status",
    "description": "ext_desc"
  },
  "clusters": []
}
```

响应：

```json
{
  "created": 1,
  "updated": 2,
  "skipped": 0,
  "total": 3,
  "source": "webhook",
  "strategy": "upsert",
  "dryRun": false
}
```

`source`：`webhook`（拉取 URL）或 `body`（请求体内联）。
`errors` 存在时包含 `{rowIndex,name,error}`，用于展示逐条失败原因。

## 运维大屏汇总

### `GET /api/ops/dashboard/summary`

从已登记集群聚合，供运维大屏使用（告警待处理数在 P2-B 接入前恒为 `0`）。

```json
{
  "totalClusters": 2,
  "healthyClusters": 1,
  "warningClusters": 1,
  "criticalClusters": 0,
  "pendingAlerts": 0,
  "vmConfigured": true,
  "domains": [
    {
      "domain": "hadoop",
      "clusterCount": 2,
      "healthyCount": 1,
      "warningCount": 1,
      "criticalCount": 0,
      "healthScore": 88,
      "healthScoreSource": "victoriametrics",
      "note": "1 个集群亚健康"
    }
  ]
}
```

**健康分（P1-5）**：当环境变量 `VICTORIAMETRICS_URL`（或 `PROMETHEUS_URL`）已配置且该域有纳管集群时，服务端对各业务域执行 instant PromQL（以 `avg(up{…})` 为主、回退 `avg(up)`），将采样值归一化到 0–100 写入 `healthScore`。未配置或查询无数据时省略 `healthScore`，`healthScoreNote` 说明原因（前端不得再用集群状态估算 85/98 分）。

## 前端调用

见 `ui/src/ui/controllers/ops-clusters.ts`。

### 实体上下文 ID 约定（P1-3）

| `entityId` | 含义 |
|------------|------|
| `all` | 业务域下全部已登记集群 |
| `{clusterId}` | 某一集群全域（UUID 来自 API） |
| `{clusterId}#{urlEncodedComponent}` | 集群内某一核心组件 |

进入业务域 Tab 时会请求 `GET /api/ops/clusters?domain=…` 构建选择器；Agent 发消息时自动附带 `[运维上下文] …` 行（P1-4）。

## 告警组（P2-B）

存储路径：`{OPENOCTA_STATE_DIR}/ops/alerts.json`。`POST /hooks/alert` 滑动窗口合并触发分析时自动写入。

### `GET /api/ops/alerts/groups`

| 参数 | 说明 |
|------|------|
| `domain` | 可选，按业务域过滤 |
| `status` | 可选：`active` \| `analyzing` \| `resolved` |

响应含 `groups`、`originalTotal`、`mergedTotal`、`reductionRate`、`pendingActive`。

### `GET /api/ops/alerts/groups/{id}`

返回单个告警组；若 Agent 会话已有回复，会填充 `rootCauseMarkdown`（从 transcript 读取）。

### `PATCH /api/ops/alerts/groups/{id}`

请求体：`{"status":"resolved"}` 等。需要 RBAC 权限 **`ops:ack`**（admin 豁免；Gateway Token 等同全权限）。

**产品路径（P2-B5）**：告警降噪保持在各业务域左侧子 Tab，不设独立顶栏 Alert Studio。

### `GET /api/ops/inspection/im-status`

返回巡检低分 IM 推送是否可用：

```json
{
  "imConfigured": true,
  "channels": ["feishu"],
  "lowScoreThreshold": 85,
  "hint": ""
}
```

未启用飞书/钉钉时 `imConfigured` 为 `false`，`hint` 提示前往通道配置。

## Agent 工具环境变量（P2-A）

| 工具 | 环境变量 |
|------|----------|
| `query_gbase_slow_sql` | `GBASE_DSN`, 可选 `GBASE_SLOW_SQL_QUERY` |
| `query_governance_lineage` | `GOVERNANCE_API_URL`, 可选 `GOVERNANCE_API_TOKEN` |
| `query_hadoop_jmx` | `HADOOP_JMX_URL` |
| `query_fi_manager_metrics` | `FI_MANAGER_URL`, 可选 `FI_MANAGER_TOKEN`, `FI_MANAGER_HEALTH_PATH` |

## 深链与 CORS

- `OPENOCTA_UI_BASE_URL`：IM 卡片与 Agent 告警分析中的 UI 链接前缀  
- `OPENOCTA_CORS_ORIGINS`：生产 API CORS 白名单（见 [deploy-ops.md](./deploy-ops.md)）  
- 告警深链格式：`/{domain}?opsSubTab=alerts&alertGroup={id}`（见 [alert-integration.md](./alert-integration.md)）
