# 运维模块 E2E 冒烟清单（P4-4）

按顺序执行；失败项记录环境与日志片段。

## 前置

- [ ] 网关已启动，`OPENOCTA_STATE_DIR` 可写
- [ ] 已登录 RBAC（或配置 Gateway Token）
- [ ] 可选：`VICTORIAMETRICS_URL`、`GBASE_DSN`、`OPENOCTA_UI_BASE_URL`

## 1. 登录与导航

- [ ] 使用非 admin 账号（如 Viewer）仅能看到有 `menu:*` 权限的 Tab
- [ ] 运维大屏 `/overview` 可打开，汇总数字与集群资产一致

## 2. 集群资产（P1）

- [ ] 「集群资产管理」可新增集群并刷新列表
- [ ] 配置 `OPS_CMDB_SYNC_URL` 后「同步 CMDB」返回 created/updated 统计并刷新表格
- [ ] 业务域页实体选择器出现刚登记的集群

## 3. 巡检（P2-A）

- [ ] 有 `ops:inspect` 时「一键手动巡检」可点击；Viewer 账号按钮 disabled
- [ ] 巡检完成后「巡检报告」列表出现新记录，详情为 Markdown（无假指标表）
- [ ] 无 IM 通道时巡检 Tab 显示配置提示

## 4. 告警（P2-B）

- [ ] `POST /hooks/alert`（带 hooks token）返回 `202`
- [ ] 业务域「告警降噪」出现告警组，降噪统计非零
- [ ] 有 `ops:ack` 时可「标记为已处理」或 `PATCH /api/ops/alerts/groups/{id}`

## 5. ChatOps（P2-C2）

- [ ] 飞书/钉钉向已启用通道发送 `/help`，收到指令说明
- [ ] `/ack <告警组ID>` 返回确认，Web 列表状态变为 resolved
- [ ] `/diagnose 磁盘偏高` 触发 Agent 回复（非即时 ack）

## 6. 大屏快捷（P1-6）

- [ ] 「启动全局深度巡检」触发 5 个 cron 任务并跳转定时任务页
- [ ] 「查看未处理告警」进入告警子 Tab

## 7. API 抽样

```bash
curl -s -H "Authorization: Bearer $TOKEN" "$GW/api/ops/dashboard/summary" | jq .pendingAlerts
curl -s -H "Authorization: Bearer $TOKEN" "$GW/api/ops/alerts/groups?domain=hadoop" | jq .total
curl -s -H "Authorization: Bearer $TOKEN" "$GW/api/ops/inspection/im-status" | jq .imConfigured
```

## 8. CORS（P4-1）

- [ ] 生产设置 `OPENOCTA_CORS_ORIGINS` 后，仅白名单 Origin 可跨域调用 API
