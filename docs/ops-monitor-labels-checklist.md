# 集群资产 monitorLabels 对齐检查清单

本文说明 **集群资产 id** 与 **VictoriaMetrics 时序标签** 如何关联，以及 BCH / FI / GBase 等业务域的接入验收步骤。

## 1. 逻辑链条（不要掉链子）

```
登记集群 → 生成资产 id (cluster-<uuid>)
         ↓
    配置 monitorLabels（与 VM 时序标签一致）
         ↓
    vm_health / Agent vm_query 将 monitorLabels 注入 PromQL
         ↓
    查询 VictoriaMetrics，得到单集群 / 域级健康分
```

| 环节 | 标识 | 是否必须相等 |
|------|------|----------------|
| 资产主键 | `id` = `cluster-<uuid>` | — |
| VM 时序 | `job` / `cluster` / `instance` 等 label | — |
| 桥梁字段 | `monitorLabels` | **值**须与 VM label 一致，**不等于**资产 id |

**错误做法**：把资产 `id` 填进 `monitorLabels` 的 `cluster` 字段，但 VM 里根本没有这个 label 值。  
**正确做法**：先在 VM 查 `label_values(up, job)` / `label_values(up, cluster)`，再把查到的值写入 `monitorLabels`。

### 格式要求

- 使用 **PromQL 标签选择器**语法，例如：`job="hadoop-prod",cluster="bj-bch-prod"`
- **不要**使用 JSON：`{"job":"hadoop-prod"}`（前后端会拒绝）
- 非 `inactive` 集群 **必须**填写 `monitorLabels`
- 各业务域至少包含下表「推荐标签键」之一

### 代码落点

| 步骤 | 文件 |
|------|------|
| 登记校验 | `src/pkg/ops/monitor_labels.go` → `CreateCluster` / `PatchCluster` |
| PromQL 注入 | `src/pkg/agent/tools/context.go` → `InjectLabelsIntoPromQL` |
| 域健康分 | `src/pkg/ops/vm_health.go` → `domainHealthScore` |
| Agent 查询 | `src/pkg/agent/tools/vm_query.go` |
| 登记页提示 | `ui/src/ui/components/ops-monitor-labels-field.ts` |

---

## 2. BCH 生态 (hadoop)

### 推荐标签键

`job` · `cluster` · `instance`（至少其一）

### 示例

```text
job="hadoop-prod",cluster="bj-bch-prod"
```

### 域级探测基线（注入前）

```promql
avg(up{job=~".*(hadoop|yarn|hdfs).*"} or up{instance=~".*hadoop.*"})
```

### 验收清单

- [ ] **VM-1** 执行 `label_values(up, job)`，记录目标集群的 `job` 值
- [ ] **VM-2** 若有 `cluster` label，执行 `label_values(up, cluster)` 确认与 CMDB/运维命名一致
- [ ] **登记** `monitorLabels` 使用上一步查到的值（不是 `cluster-uuid`）
- [ ] **VM-3** 验证：`count(up{job="<你的job>",cluster="<你的cluster>"})` > 0
- [ ] **驾驶舱** 该域健康分来源为「监控」而非仅「资产」估算（需配置 `VICTORIAMETRICS_URL`）
- [ ] **Agent** 在集群上下文中执行 `vm_query`，确认结果只包含该集群序列
- [ ] **告警**（可选）告警 fingerprint 的 `clusterId` 填资产 `id`，与 `monitorLabels` 分工不同

### 常见问题

| 现象 | 原因 | 处理 |
|------|------|------|
| 健康分「待评分」 | 未配置 VM URL 或 monitorLabels 为空 | 配置环境变量 + 补全标签 |
| 有分但不对 | 只配了域级 `job`，多集群串数据 | 增加 `cluster` / `env` 区分 |
| JSON 保存失败 | 误用 JSON 格式 | 改为 `key="value"` |

---

## 3. FI 商业生态 (fi)

### 推荐标签键

`job` · `cluster` · `fusion_cluster` · `instance`（至少其一）

### 示例

```text
job="fi-prod",cluster="huhe-fi-prod"
```

### 域级探测基线

```promql
avg(up{job=~".*(fusion|fi).*"} or up{instance=~".*fi.*"})
```

### 验收清单

- [ ] **VM-1** `label_values(up, job)` 与 `label_values(up, cluster)`（或 `fusion_cluster`）
- [ ] **FI Manager** 若有独立指标端点，登记 `fiManagerUrl`（SQL/巡检用，与 monitorLabels 互补）
- [ ] **登记** monitorLabels 与 FI 节点 exporter 标签一致
- [ ] **VM-2** `count(up{job="fi-prod",cluster="huhe-fi-prod"})` > 0
- [ ] **多集群** 同域每台集群必须用不同 `cluster`/`env` 标签，避免平均到错误机器
- [ ] **驾驶舱** 域详情风险 Top 集群能对应到登记名称

---

## 4. GBase 数据库 (gbase)

### 推荐标签键

`job` · `cluster` · `instance`（至少其一）

### 示例

```text
job="gbase-prod",instance="gbase-primary"
```

### 域级探测基线

```promql
avg(up{job=~".*gbase.*"} or up{instance=~".*gbase.*"})
```

### 验收清单

- [ ] **VM-1** 确认 GBase exporter 的 `job` / `instance` 标签
- [ ] **主备** 多套实例用 `instance` 或 `cluster` 区分
- [ ] **登记** `monitorLabels` 仅负责指标；`gbaseDsnRef` 负责 SQL 巡检
- [ ] **VM-2** `count(up{job="gbase-prod"})` 或带 `instance` 过滤 > 0
- [ ] **驾驶舱** GBase 域聚合分包含该集群样本

---

## 5. 其他域（简表）

| 域 | 推荐标签键 | 示例 |
|----|------------|------|
| governance | job, cluster, service | `job="gov-platform",service="metadata-registry"` |
| dataapps | job, cluster, app | `job="dataapp-scheduler",app="core-scheduler"` |

---

## 6. 一键 VM 验证命令

将 `VM_URL`、`LABELS` 替换为实际值（`LABELS` 为登记 monitorLabels 片段，如 `job="hadoop-prod",cluster="bj-bch-prod"`）：

```bash
# 注意：PromQL 中逗号分隔的 label 需整体 URL 编码
curl -G "${VM_URL}/api/v1/query" \
  --data-urlencode 'query=count(up{job="hadoop-prod",cluster="bj-bch-prod"})'
```

返回 `result` 非空且 value > 0，说明标签对齐成功。

---

## 7. CMDB 同步注意

CMDB 行须携带可解析的 `monitorLabels`（同上 PromQL 格式）。同步走 `validateClusterCreate`，**非 inactive 且无标签的行会失败**，避免「有资产无监控」的断链数据进入系统。

---

## 8. 告警与资产 id

| 字段 | 用途 |
|------|------|
| 资产 `id` | API、导航、告警 `clusterId` 关联 |
| `monitorLabels` | PromQL 过滤 VM 时序 |

告警接入时 `clusterId` 建议填资产 `id`；指标查询仍只认 `monitorLabels`，二者各司其职。
