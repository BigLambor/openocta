# 外部告警接入与 ChatOps 联调清单（P2-C）

## Webhook 接入

1. 在网关配置中启用 Hooks，设置 `hooks.token`。
2. 将 Prometheus Alertmanager / Grafana / 自定义监控的 Webhook 指向：
   - `POST {GATEWAY}/hooks/alert`
   - Header：`Authorization: Bearer <hooks.token>` 或 `X-OpenOcta-Token`
3. 请求体示例：

```json
{
  "title": "YARN 队列资源不足",
  "message": "default 队列 memory 使用率 95%",
  "severity": "critical",
  "source": "hadoop-prod",
  "data": { "cluster": "bj-bch-01" }
}
```

4. 同一 `source` 在 15s 滑动窗口内合并；满 20 条或超时后触发 Agent 分析并写入 `ops/alerts.json`。

## UI 深链（IM 卡片）

设置环境变量 `OPENOCTA_UI_BASE_URL`（如 `https://ops.example.com`），IM 消息可携带：

- 巡检异常：`{base}/hadoop?opsSubTab=inspections`
- 告警组：`{base}/hadoop?opsSubTab=alerts&alertGroup=alert-group-xxx`

## ChatOps 指令（P2-C2）

在已启用的飞书/钉钉通道中，向机器人发送：

| 指令 | 说明 |
|------|------|
| `/help` | 显示指令列表 |
| `/ack <告警组ID>` | 将告警组标记为 `resolved`（Web 告警列表可复制 ID） |
| `/diagnose [问题]` | 转交 Agent 进行运维诊断（需通道可达 Agent） |

Web 端确认告警需 RBAC 权限 `ops:ack`（或 admin）。

## 联调检查

- [ ] `POST /hooks/alert` 返回 `202` 且 `status: queued`
- [ ] 业务域「告警降噪」Tab 出现新告警组
- [ ] Agent 会话结束后详情区展示根因 Markdown
- [ ] 配置 `OPENOCTA_UI_BASE_URL` 后 IM 卡片含可点击链接
- [ ] 巡检得分 &lt; 85 且启用飞书/钉钉时收到推送（见 [ops-api.md](./ops-api.md) IM 状态接口）
