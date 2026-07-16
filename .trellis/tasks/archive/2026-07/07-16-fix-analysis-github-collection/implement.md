# Implementation

1. 为 SignalRepo 增加 qualified signal lifecycle 更新，并在 Radar API 暴露 PATCH 路由。
2. 内容包创建要求 practiced；更新 API 单元测试。
3. 前端增加 lifecycle API 和 App 状态更新。
4. 重写 RadarDashboard 的派生分组与卡片，删除重复分栏和 pending evidence 渲染。
5. 更新 React 渲染测试，运行 Go/Python/前端全量检查。
6. 浏览器验证加入实践、标记完成、生成内容入口与刷新持久化。

## Validation

- `cd backend && go test ./... && go vet ./...`
- `cd services/collector && uv run --no-sync python -m unittest discover -s tests`
- `cd frontend && npm test && npm run build && npm run lint`
