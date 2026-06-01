# OpenOcta 运维 UI 风格指南（P3）

> 对齐 [ops-roadmap.md](./ops-roadmap.md) Cursor Dashboard 气质：克制、工具型、低噪声。

## 设计 Token

定义于 `ui/src/styles/ops-tokens.css`：

| Token | 用途 |
|-------|------|
| `--ops-radius-sm/md/lg` | 卡片、按钮圆角 8–12px |
| `--ops-space-xs` ~ `--ops-lg` | 间距 8 / 12 / 16 / 24px |
| `--ops-health-ok/warn/critical` | 健康态小标签色 |

## 布局

- 页面：`ops-page`，最大宽度约 1200–1400px，左右留白 32–40px
- 业务域：`ops-sidebar`（260px）+ `ops-main-content`
- 主从详情：`ops-main-columns`（约 38% / 62%）

## 组件

| 类名 | 说明 |
|------|------|
| `ops-panel` | 1px 边框 + 12px 圆角容器 |
| `ops-btn` / `ops-btn--primary` | 次要 / 主按钮 |
| `ops-banner` | 信息条（IM 未配置等） |
| `ops-status` | 空状态 / 错误 / 骨架（见 `ops-status.ts`） |
| `ops-markdown` | Agent 报告与告警根因 Markdown |

## 侧栏 Active（P3-3）

- **不用**整块 accent 底
- 使用 `box-shadow: inset 3px 0 0 var(--accent)` + 浅背景

## 控制条（P3-4）

- `ops-main-header` + `ops-entity-selector`
- 下拉 `box-shadow: 0 8px 24px`，最大高度 350px

## 数据展示

- 统计数字：20–22px 字重 600（避免 28px+ 大屏字）
- 健康分：16–18px 带 ok/warn/critical 色类
- 禁止无 API 时的假指标表

## 空状态文案

| 场景 | 标题 | 说明 |
|------|------|------|
| 无集群 | 尚无纳管集群 | 引导至集群资产管理 |
| 无告警 API | 告警 API 未就绪 | 升级网关 |
| 无告警组 | 暂无活动告警 | 不展示演示数据 |
| 无 IM | 巡检 Tab 横幅 | 前往通道配置 |

## 样式文件

| 文件 | 范围 |
|------|------|
| `ops-tokens.css` | 通用状态、按钮、页面头 |
| `ops.css` | 业务域、告警/巡检双栏 |
| `ops-dashboard.css` | 运维大屏 |
| `ops-login.css` | RBAC 登录 |
