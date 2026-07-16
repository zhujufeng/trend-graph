# 核心工作流修复设计

## Problem

当前首页按数据处理状态堆内容：待分析原文在最前、已分析结果重复分栏、来源设置常驻。它暴露了存储结构，却没有帮助用户决定今天实践什么。

## Minimal Flow

`qualified/new -> queued -> practiced -> content package`

- `new`：AI 已筛选，尚未加入实践。
- `queued`：用户决定实践。
- `practiced`：用户确认已完成，允许生成三平台内容。
- `dismissed`：从日常工作区隐藏。

复用现有 `signals.lifecycle_state`，不新增表或字段。

## Backend Boundary

- 新增 `PATCH /api/radar/signals/:id/lifecycle`，请求 `{state}`。
- 状态只接受 `new|queued|practiced|dismissed`，且仅更新 qualified radar source。
- 内容包创建额外要求 `lifecycle_state=practiced`。

## Frontend Information Architecture

1. 顶部显示待实践、实践中、可创作、待分析数量。
2. `待实践`：new 和 queued，queued 优先；卡片呈现变化、痛点、行动、安装与原始来源。
3. `已实践，可创作`：practiced，仅此处提供内容生成。
4. `采集队列`：pending 的精简标题列表，默认折叠，不渲染 evidence excerpt。
5. `来源设置`：默认折叠到页面底部。

## Compatibility

已有 qualified 信号默认 `new`；已有内容包不修改。无数据库迁移。失败时前端保留当前列表并显示错误。
