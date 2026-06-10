# 主业务存储准入规则

> 状态：商用化基线 v1  
> 目标：停止新增主业务 JSON store，避免生产数据继续分散

## 强制规则

1. 资产、告警、任务、审批、权限、健康结果、审计、会话索引不得新增 JSON 主写路径。
2. 新增主业务表必须进入 `openocta.db`，并通过 migration 管理。
3. 新 repository 必须定义接口、错误语义、幂等策略和测试 fixture。
4. JSON 文件只能用于兼容导入、备份导出、transcript 归档、测试 fixture 或明确的用户导入导出。
5. demo seed 数据默认关闭，只有 `OPENOCTA_SEED_DEMO_DATA=1` 时允许写入。

## 评审门槛

新增持久化 PR 必须回答：

| 问题 | 必须满足 |
|------|----------|
| 归属对象是什么？ | 已映射到商用核心领域对象。 |
| 主存储在哪里？ | `openocta.db` 表或已批准的外部系统。 |
| 是否新增 JSON 主写？ | 不允许。 |
| 如何迁移旧数据？ | 有导入、备份、幂等和重复数据策略。 |
| 如何审计？ | 至少有 actor、action、object、time、request/run/session 关联。 |
| 如何测试？ | 覆盖空库、旧数据、重复启动、并发写或权限边界。 |

## 允许例外

| 类型 | 条件 |
|------|------|
| Transcript JSONL | 仅作为聊天原文归档；列表、查询、权限过滤必须走 DB。详见 [session-transcript-strategy.md](./session-transcript-strategy.md)。 |
| 用户导入导出 | 必须由显式操作触发，不得成为服务运行主路径。 |
| 测试 fixture | 只能位于测试目录或测试临时目录。 |
| 兼容迁移备份 | migration/import 时生成，只读保留，不能继续作为主写。 |

任何例外必须在对应模块 README 或任务文档中标明生命周期和退役条件。
