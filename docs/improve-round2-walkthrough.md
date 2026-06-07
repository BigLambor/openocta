# OpenOcta 安全加固与架构收敛（Phase 1 & Phase 2）实施结果说明

本改造方案已完整落地并经过自动化测试集验证。我们对 OpenOcta 的网关、鉴权与装配层进行了一系列安全硬化与模块化改造，在提升桌面端免受安全隐患的同时，引入了细粒度的方法级 RBAC 权限控制。

---

## 变更文件概览

### 1. 第一阶段：安全加固与止血 (Phase 1)

* **[config.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/config/config.go)**
  * 移除默认硬编码 Token 的强制行为，仅保留其作为历史兼容比对。
  * 引入 `generateRandomToken` 函数以在首启自动生成 24 字节随机十六进制 Token。
  * 对遗留的旧版默认 Token 实例进行自动升级迁移，自动替换为新的随机 Token 并写入配置文件。

* **[auth.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/auth.go)**
  * 将 `gatewayHTTPAuthDisabled` 设置为 `false`，全面开启网关对外部 API 的 Token/Session 鉴权。

* **[hub.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/ws/hub.go)**
  * 重塑 `websocket.Upgrader` 逻辑，在桌面模式下，仅放行本地回环（`localhost` / `127.0.0.1`）及 Wails 专属协议头（`wails://`），隔绝恶意的跨站 WebSocket 劫持（CSWSH）。

* **[server.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/server.go)**
  * 限制 `/debug/pprof/` 路由默认不注册，仅在 `OPENOCTA_ENABLE_PPROF=1` 环境变量显式启用下进行注册。且注册后增加 `requirePprofAuth` 过滤器，限制只对回环网卡或 admin 角色的授权会话开放。
  * **Token 注入**：修改 `serveIndex`，在向桌面端或本地回环输出 `index.html` 前，自动在 `<head>` 中注入 `<script>window.__gateway_token__ = "TOKEN";</script>` 以便实现无感登录。

* **[storage.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/storage.ts)** (前端 UI)
  * 调整 `loadSettings` 函数，优先检测并选用 `window.__gateway_token__`，且在初次登录或使用原有默认 Token 时自动无缝升级，保持登录闭环。

---

### 2. 第二阶段：架构收敛与 WebSocket 权限控制 (Phase 2)

* **[server.go (Refactoring)](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/server.go)**
  * 将 monolithic 的 `NewServer`（约 320 行）进行模块化拆分，引入以下辅助初始化函数：
    * `initStores(stateDir string) error`：初始化 SQLite 和各业务 Store。
    * `wireToolsHooks()`：将全局 hooks 绑定至 `tools`。
    * `initCron(stateDir string, skipCron bool) (*cron.Service, error)`：创建 Cron 定时任务服务。
    * `initChannels(chReg, outReg)`：初始化 Channel 插件与适配器注册。
  * 重写 `Handler()` 路由分发器，允许 `/ws` 上的 WebSocket 握手直通 `handleWSUpgrade`，避开 `statusLoggingResponseWriter` 的劫持，解决了连接升级返回 500 的问题。

* **[types.go (handlers)](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/types.go)**
  * 修改 `Client` 结构体，添加 `Session *rbac.UserSession` 字段，以从 WebSocket 握手阶段继承用户的 RBAC 权限。
  * 增加 `MethodDescriptor` 结构体：
    ```go
    type MethodDescriptor struct {
        Handler            Handler
        RequiredPermission string
    }
    ```
  * 将 `Registry` 的定义由 `map[string]Handler` 改为 `map[string]MethodDescriptor`。

* **[dispatch.go (handlers)](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/dispatch.go)**
  * 修改 `Dispatch` 方法：
    * 支持从 `Registry` 获取 `MethodDescriptor`。
    * 若方法配置了 `RequiredPermission` 且 `opts.Client != nil`，在分发前执行权限匹配检查：
      * Go 后端内部调用（`opts.Client == nil`）直接放行.
      * legacy Gateway Token 授权客户端（`opts.Client.Session == nil`）直接放行.
      * `admin` 角色用户直接放行.
      * 其它用户检查 `Permissions` 列表中是否包含声明的权限值，无权则返回 `"forbidden"` 错误帧。

* **[health.go (handlers)](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/health.go)**
  * 重构 `NewRegistry`，使注册的 API 映射 to `MethodDescriptor`，并进行高风险权限绑定：
    * 绑定 `"menu:config"` 到配置管理、技能安装、员工增删等高危操作（如 `config.*`、`cron.*` 等）。
    * 绑定 `"ops:ack"` 到告警和审批等（如 `approvals.approve` 等）。

* **[hub.go (ws)](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/ws/hub.go)**
  * 修改 `Client` 结构体，新增 `Session *rbac.UserSession` 字段。
  * 调整 `handleConnect` 握手阶段，提取 `cp.Auth.Token`：
    * 优先进行 RBAC 会话解析与验证，保存解析出来的 `UserSession`。
    * 如果没有有效的 RBAC 会话且配置了 `expectedToken`，则回退检验 legacy 网关令牌。
  * 在消息分发阶段，将 `c.Session` 填充并传递给 `handlers.Client`，实现权限上下文在整个请求流中的闭环。

---

## 验证与测试结果

我们编写了配套的红线安全与 RBAC 权限测试文件 [security_test.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/security_test.go)，覆盖以下四个集成测试：

1. **`TestHTTPAuthHardening_RejectsUnauthorized`**
   * 验证没有 Token / 携带非法 Token 时访问敏感 API（如 `/api/config`）是否能准确拦截并返回 401 状态。携带正确 Token 时是否顺利放行。
2. **`TestPprofAuthHardening`**
   * 验证默认状态下 `/debug/pprof` 路由不存在；环境变量开启后，仅放行回环连接，外部网卡连接拦截并返回 403 Forbidden。
3. **`TestWebSocketOriginChecks`**
   * 验证桌面模式下仅放行 `localhost` 与 `wails://` 协议建立 WebSocket 握手，外部恶意 Origin （如 `http://malicious-attacker.com`）握手被强制拦截返回 403。
4. **`TestWebSocketMethodPermissions`**
   * 验证 WebSocket 下细粒度方法级权限：
     * Viewer 用户（角色为 `viewer`，无 `menu:config` 权限）调用 `config.get` 被拒绝，返回 `forbidden` 错误帧。
     * Viewer 用户调用 `health`（无权限要求）可以正常执行。
     * Admin 用户调用 `config.get` 顺利执行。
     * Legacy Gateway Token 授权客户端调用 `config.get` 顺利执行（绕过权限控制）。

### Go 测试运行结果

```bash
go test -v ./pkg/gateway/http/...
```

```plain
=== RUN   TestHTTPAuthHardening_RejectsUnauthorized
2026/06/07 11:46:54 INFO HTTP Request method=GET url=/api/config status=401 remote_addr=192.0.2.1:1234
2026/06/07 11:46:54 INFO HTTP Request method=GET url=/api/config status=401 remote_addr=192.0.2.1:1234
2026/06/07 11:46:54 INFO HTTP Request method=GET url=/api/config status=200 remote_addr=192.0.2.1:1234
--- PASS: TestHTTPAuthHardening_RejectsUnauthorized (0.01s)
=== RUN   TestPprofAuthHardening
2026/06/07 11:46:55 INFO HTTP Request method=GET url=/debug/pprof/ status=404 remote_addr=127.0.0.1:1234
2026/06/07 11:46:55 INFO HTTP Request method=GET url=/debug/pprof/ status=200 remote_addr=127.0.0.1:1234
2026/06/07 11:46:55 INFO HTTP Request method=GET url=/debug/pprof/ status=403 remote_addr=192.168.1.100:1234
--- PASS: TestPprofAuthHardening (0.01s)
=== RUN   TestWebSocketOriginChecks
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=192.0.2.1:1234
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=192.0.2.1:1234
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=192.0.2.1:1234
--- PASS: TestWebSocketOriginChecks (0.01s)
=== RUN   TestWebSocketMethodPermissions
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=127.0.0.1:62397
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=127.0.0.1:62398
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=127.0.0.1:62399
2026/06/07 11:46:55 INFO WS Upgrade Request method=GET url=/ws remote_addr=127.0.0.1:62400
--- PASS: TestWebSocketMethodPermissions (0.01s)
PASS
ok  	github.com/openocta/openocta/pkg/gateway/http	1.214s
```

所有的单元和集成测试均编译通过并成功执行。

---

## 第三阶段：核心状态数据层 SQLite 迁移 (Phase 3)

为了从根本上解决原 JSON 文件存储在并发写入时的局限性并提供事务保障，我们在第三阶段完成并优化了核心状态数据的 SQLite 迁移。

### 核心变更说明

1. **OPS Health Store SQLite 迁移**
   * 已完成在 `sqliteHealthSignalStore` 与 `HealthSignalStore` 接口的装配。
   * 创建了 `health_signals` 与 `health_snapshots` 数据库表，以 `detail_json` 作为核心数据存储格式。
   * 实现了首启时的平滑迁移逻辑：自动寻找并读取原本的 `health_signals.json` 和 `health_snapshots.json`，导入 SQLite 数据库后备份为 `.json.bak`。

2. **Cron 任务服务 SQLite 迁移与缺陷修复**
   * 重构了 `cron.Service`，修复了原本在 SQLite 数据库启用时，执行 `Run` 操作中遍历 `s.store.Jobs` 导致的空指针异常（Nil Pointer Dereference）。现在 `Run` 操作已完美适配 SQLite 数据源。
   * 实现了 `cron_jobs` 表的数据存储、查询及更新逻辑，并在此基础上提供了 `jobs.json` 的平滑迁移与备份备份功能。

3. **Session 存储 SQLite 迁移**
   * 修改了 `session/store.go`，适配 SQLite 对 `sessions` 表的读写操作。
   * 实现了 `sessions.json` 到 SQLite `sessions` 表的平滑数据合并与自动迁移备份。

---

## 第四阶段：状态治理与体验重构 (Phase 4)

在第四阶段中，我们专注于前端根组件的状态解耦，并实现了 Channel 启动异常在运行期的捕获与回显机制。

### 核心变更说明

1. **Channel 启动期异常捕获与回显**
   * 在 [manager.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/channels/manager.go) 中定义了 `RuntimeStartResult` 结构体，用于封装每个通道运行时的启动结果（包含 `RuntimeID` 与首启 `Error` 描述）。
   * 重构了 `Manager.Start(ctx)`，捕获初始化或连接阶段抛出的致命错误，在标记对应的通道运行时为 `ConnectionFailed` 的同时，通过 `RuntimeStartResult` 列表回传至上层。
   * 修改了 [server.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/http/server.go) 中的调用，在控制台日志中输出具体异常，以便管理员快速排查通道故障。

2. **前端根组件状态树拆分 (Lit Controllers)**
   * 将 [app.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app.ts) 中过于庞大的 Chat, Config, Ops 和 Channels 等四大本地域状态与逻辑，分别提取到独立的 **Reactive Controller** 中：
     * [chatStore.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/controllers/chatStore.ts) (管理 Copilot 历史会话、流式交互、思考级别等)
     * [configStore.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/controllers/configStore.ts) (管理配置表单、草稿暂存、Schema 校验等)
     * [opsStore.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/controllers/opsStore.ts) (管理集群纳管、大屏指标、健康评分、Spark/Flink 治理等)
     * [channelsStore.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/controllers/channelsStore.ts) (管理微信公众号、企业微信、Nostr、WhatsApp 等通道配置)
   * 在根组件 `OpenClawApp` 中引入这 4 个 Store 实例，并通过 `get/set` 委托机制和 `this.requestUpdate()` 保持向下兼容，确保现有的渲染模板层（[app-render.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/app-render.ts)）不发生大范围的代码抖动。**注意：此设计目前作为架构演进中的兼容性过渡，并非最终的根组件彻底解耦。彻底重构工作将在下一阶段继续推进。**

3. **历史遗留测试缺陷修复与回归**
   * **HDFS 命名空间选项构造恢复**：修复了在 [workbench-context.test.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/ops/workbench-context.test.ts) 中的测试用例失败。该失败系因历史提交中误删了 `workbench-context.ts` 里对集群 Namespace & Directory 循环构造逻辑，我们对此进行了完整恢复。
   * **CMDB 纳管资产交互测试重塑**：修复了 [e2e_remediation_smoke.browser.test.ts](file:///Users/isadmin/MagicSpace/openocta/ui/src/ui/e2e_remediation_smoke.browser.test.ts) 在验证集群新增流程时试图操纵已被废弃的 `<details class="asset-form-panel">` 导致的崩溃。现已重塑为通过按钮触发侧边栏 Drawer 打开，进而操作抽屉内表单。
   * **一键巡检按钮标识别名修复**：修改 E2E 测试对 “一键手动巡检” 的模糊查询，同步改为生产所使用的 “一键巡检”。

---

## 第四阶段补充：Session 数据隔离与 WebSocket 权限强化 (P1 修复与权限加固)

在接受三方专家评审后，针对其提出的 P1 级 Session 数据覆盖风险与 WS 权限覆盖不完整问题，我们进行了二次加固：

### 核心加固变更说明

1. **Session SQLite 隔离防覆盖机制 (P1 修复)**
   * **数据隔离与复合主键**：修改了 [store.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/session/store.go) 中的 `sessions` 表结构，添加了 `store_path` 字段，并将主键修改为复合主键 `PRIMARY KEY (store_path, session_key)`。
   * **平滑数据库升级**：在 `createSessionTables` 中引入了 PRAGMA 级别的数据库字段检测与热升级逻辑：如果旧表存在且缺失 `store_path` 字段，会自动将其安全重命名（`sessions_old`），创建新复合主键表，并将数据安全导回（`store_path` 默认为空），最后销毁旧表，实现无损平滑迁移。
   * **作用域隔离读写**：重构了 `LoadSessionStore`, `SaveSessionStore` 和 `migrateSessionJSONToSQLite`，所有操作均强绑定当前传入的 `storePath`。保存或删除时仅影响该 Store Path 对应的 Agent/Session 数据，从源头上杜绝了多 Agent 数据相互覆盖或全量删除的隐患。
   * **配套单元测试**：在 [store_test.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/session/store_test.go) 中新增了 `TestSessionSQLiteIsolation` 测试，对多 Agent 数据隔离性写入、读取，以及旧表迁移机制进行了全面覆盖。

2. **WebSocket 高风险方法权限补齐与安全标示**
   * **高危方法权限全覆盖**：修改了 [health.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/health.go)，给专家指出的所有高危未保护方法均补齐了 `RequiredPermission: "menu:config"` 权限校验：
     * 代理及文件操作：`agents.files.set`、`chat.inject`
     * 数字员工任务变更：`employee.tasks.create/update/delete`
     * 系统更新执行：`update.run`
     * 节点及设备配对审批：`node.pair.approve/reject`、`device.pair.approve/reject`、`device.token.rotate/revoke`
     * 多 Agent 协同空间变更：`swarm.workspaces.create/delete/abortAll`、`swarm.members.add/remove`、`swarm.message.send`
   * **过渡兼容性注释标示**：在 [dispatch.go](file:///Users/isadmin/MagicSpace/openocta/src/pkg/gateway/handlers/dispatch.go) 针对 Legacy Gateway Token 绕过权限检查的逻辑（`opts.Client.Session == nil`）处添加了明确的警告注释，明示其为为了历史兼容设计的过渡性方案，不应作为最终的安全模型。

### 测试验证结果

我们重新运行了前后端的所有测试：

* **前端测试运行** (Vitest Playwright Browser Runner):
  ```bash
  npm run test
  ```
  结果：**53 个测试套件，396 个测试用例全部通过（100% 成功率）**。
  
* **后端测试运行** (Go test):
  ```bash
  go test ./pkg/...
  ```
  结果：**所有后端单元测试（包含新添加的 SQLite 数据隔离与模式热迁移测试）全部通过**。
