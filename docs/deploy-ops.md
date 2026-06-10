# 运维模块部署说明（P4-5）

## 环境变量

| 变量 | 用途 |
|------|------|
| `OPENOCTA_STATE_DIR` | 状态目录（含 `openocta.db`、旧 JSON 导入备份） |
| `VICTORIAMETRICS_URL` | 大屏健康分 PromQL |
| `GBASE_DSN` | GBase 慢 SQL 工具 |
| `GOVERNANCE_API_URL` | 治理血缘 API |
| `HADOOP_JMX_URL` | Hadoop JMX HTTP |
| `FI_MANAGER_URL` | FusionInsight Manager API |
| `OPENOCTA_UI_BASE_URL` | IM / 告警深链前缀 |
| `OPENOCTA_CORS_ORIGINS` | 生产 CORS 白名单（逗号分隔 Origin）；未设置则 `*`（仅建议开发环境） |
| `OPS_CMDB_SYNC_URL` | CMDB 导出 HTTP GET 地址（资产页「同步 CMDB」） |
| `OPS_CMDB_SYNC_TOKEN` | 可选，访问 CMDB 时的 Bearer Token |

## 备份与恢复（C4-11 / C4-12）

```bash
cd src
go build -o openocta ./cmd/openocta

# 在线备份（SQLite VACUUM INTO + 附件目录 tar.gz）
export OPENOCTA_STATE_DIR=/var/lib/openocta
./openocta backup -o /var/backups/openocta-$(date +%Y%m%d).tar.gz

# 校验备份包
./openocta backup-verify -i /var/backups/openocta-20260610.tar.gz

# 恢复到空目录（含 schema 版本兼容检查）
./openocta restore -i /var/backups/openocta-20260610.tar.gz --state-dir /var/lib/openocta-new --force
```

备份包包含：`openocta.db`、`openocta.json`、`sessions/`、`ops/`、`credentials/`、`workspace/`、`employees/` 及 `manifest.json`（SHA-256 校验）。

## 启动

```bash
cd src
export OPENOCTA_STATE_DIR=/var/lib/openocta
export VICTORIAMETRICS_URL=http://vm:8428
export OPENOCTA_CORS_ORIGINS=https://ops.example.com
go run ./cmd/openocta gateway
```

## 验证

1. `GET /api/ops/dashboard/summary` — 集群汇总与健康分  
2. `GET /api/ops/inspection/im-status` — IM 是否可用于巡检推送  
3. `POST /hooks/alert` — 告警合并入库  

前端静态资源由网关 `dist` 或嵌入资源提供，与网关同域部署时可不配置 CORS。
