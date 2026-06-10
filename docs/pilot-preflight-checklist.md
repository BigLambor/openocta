# 试点前检查清单

> 适用范围：内网 MVP 试点上线前（非正式商用版全量验收）  
> 关联：[商用发布门禁](./commercialization-release-gates.md) · [任务分解 §9 MVP 范围](./commercialization-task-breakdown.md#9-最小商用版本任务范围) · [运维冒烟](./e2e-ops-smoke.md)

**用法：** 逐项勾选；**必过** 未全绿则不启动试点；**建议** 可记录风险后试点。

---

## 1. 环境与部署（必过）

- [ ] 已设置 `OPENOCTA_STATE_DIR`（独立、可写、已备份权限 750）
- [ ] 网关可启动，前端可访问（桌面或 `go run ./cmd/openocta gateway` / 等价部署）
- [ ] `GET /health` 或 `GET /_ready` 返回 200
- [ ] **未**设置 `OPENOCTA_SEED_DEMO_DATA=1`（生产试点默认空库）
- [ ] 生产跨域已配置 `OPENOCTA_CORS_ORIGINS`（若前后端不同域）

---

## 2. 安全与账号（必过）

- [ ] 首次启动已完成管理员 setup（无默认 `admin/admin888` 可直接登录）
- [ ] 使用 HttpOnly Cookie / RBAC 登录成功，刷新页面会话仍有效
- [ ] 连续错误登录触发 lockout（或审计中有失败记录）
- [ ] 至少创建 1 个 **Viewer** 与 1 个 **运维角色** 账号用于越权验证
- [ ] Viewer 无法访问「系统设置 / config.get / chat.send」（403 或按钮不可用）
- [ ] 运维账号可执行有权限的巡检/告警确认操作

---

## 3. 数据与迁移（必过）

- [ ] 空库启动：`openocta.db` 自动 migration，无报错退出
- [ ] 若有旧 JSON：集群/告警/任务/Cron/Session 导入后 **原文件已备份或删除**，重启数据仍在
- [ ] 重复启动 migration 幂等（二次启动无 checksum 错误）
- [ ] 在 UI 或 API 新增 1 条集群、1 条告警组，重启后仍在

---

## 4. 核心业务流程（必过）

按 [e2e-ops-smoke.md](./e2e-ops-smoke.md) 精简路径：

- [ ] **资产**：新增集群 → 列表可见 → 大屏汇总数字一致
- [ ] **告警**：Webhook/`POST /hooks/alert` 入库 → 告警列表可见 → 有 `ops:ack` 时可确认/resolve
- [ ] **巡检/Agent**：有 `ops:inspect` + `tool:execute` 时手动巡检或 chat 可触发 → JobRun/报告可查看
- [ ] **审批**：高风险命令（如 bash/write）弹出或阻塞审批 → 批准后执行 / 拒绝后不执行
- [ ] **Cron**（若启用）：添加任务 → 重启后任务仍在 → 手动 run 有 run 记录

---

## 5. 备份恢复（必过）

```bash
cd src && go build -o openocta ./cmd/openocta
export OPENOCTA_STATE_DIR=/path/to/state

# 在线备份
./openocta backup -o /tmp/openocta-pilot-backup.tar.gz

# 校验
./openocta backup-verify -i /tmp/openocta-pilot-backup.tar.gz

# 恢复到空目录
mkdir -p /tmp/openocta-restore-test
./openocta restore -i /tmp/openocta-pilot-backup.tar.gz \
  --state-dir /tmp/openocta-restore-test --force

# 用恢复目录启动并验证数据仍在
OPENOCTA_STATE_DIR=/tmp/openocta-restore-test <启动网关>
```

- [ ] backup 成功，manifest 含 schema version 与 checksum
- [ ] restore 成功，集群/告警等试点数据可读取
- [ ] 恢复后 migration 可正常启动（schema 版本兼容）

---

## 6. 自动化测试（建议，发布前至少跑一轮）

```bash
cd src
go test ./pkg/backup/... ./pkg/rbac/... ./pkg/gateway/http/... \
  ./pkg/ops/... ./pkg/security/... -count=1
```

- [ ] 上述包测试全部 PASS
- [ ] 已知缺口已记录（域过滤 C3-10、readyz 深度检查、metrics 未做）

---

## 7. 试点前已知限制（知情即可）

| 项 | 说明 |
|----|------|
| `/metrics` | 未实现 Prometheus 指标（C4-7） |
| readyz | 当前 `/_ready` 仅进程级，未检查 DB/connector 深度状态 |
| 域隔离 | C3-10 部分完成，非 admin 可能仍看到未授权域数据 |
| Transcript | 消息全文仍在 JSONL，非 DB 查询 |
| OIDC/LDAP | 未接入，仅本地 RBAC |
| Phase 5 | License / 多租户 / 插件市场均未做 |

---

## 8. 试点启动签字（可选）

| 字段 | 填写 |
|------|------|
| 版本 | OpenOcta __ · schema __ · 前端构建 __ |
| 环境 | 部署形态 __ · stateDir __ · 域名 __ |
| 账号 | 管理员初始化方式 __ · 试点角色账号 __ |
| 备份 | 最近 backup 路径 __ · restore 演练时间 __ |
| 测试 | `go test` 结果 __ · 手工冒烟执行人 __ |
| 风险接受 | 未过「建议」项及 §7 限制是否接受：是 / 否 |

---

**判定：**

- **可启动试点**：§1～§5 必过项全部勾选，§6 建议项至少执行一次或书面豁免。
- **暂缓试点**：备份/恢复失败、默认 demo 数据写入、Viewer 可越权写配置/执行 Agent、migration 失败仍启动。
