# Bug Analysis: 内容生成 500 与后台任务边界

## 1. Root Cause Category

- **Category**: D / E — Test Coverage Gap and Implicit Assumption
- **Specific Cause**: Go 将 nil `*analyzer.Analyzer` 装入接口后接口本身非 nil，绕过 handler 防护并 panic。验收时又把模型和后台任务一起关闭，错误地假设“安全启动”仍能测试手动生成。

## 2. Why Fixes Failed

1. 安全验收启动：清空 DeepSeek 避免批量调用，同时也关闭了用户主动生成，交付状态不完整。
2. 首次后台开关实现：搜索同名 `INTERNAL_INGEST_SECRET` 条件时改错边界，关闭了写入路由却仍启动采集，产生 404；真实 health 和日志立刻暴露问题。

## 3. Prevention Mechanisms

| Priority | Mechanism | Specific Action | Status |
| --- | --- | --- | --- |
| P0 | Runtime boundary | `BACKGROUND_JOBS_ENABLED=false` 只关闭自动任务，保留模型和手动 API | DONE |
| P0 | Integration feedback | 缺模型请求断言 503；有模型请求断言 201；health 断言 schedulers=0 | DONE |
| P1 | Documentation | 在 signal radar contract 记录 typed-nil 和后台任务边界 | DONE |

## 4. Systematic Expansion

- **Similar Issues**: 任何把可空具体指针传给 Go 接口的装配代码都可能产生 typed-nil。
- **Design Improvement**: 可选依赖在 composition root 显式转成 literal nil，不用 reflection 掩盖调用错误。
- **Process Improvement**: 启动模式必须通过 health 和一次真实目标请求验证，不能只看进程成功监听。

## 5. Knowledge Capture

- [x] 更新 `.trellis/spec/backend/signal-radar-contract.md`
- [x] 更新 `backend/.env.example`
- [x] 增加配置解析测试和真实端到端反馈环
