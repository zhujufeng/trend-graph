# 个人信息推送产品实施计划

## 0. 基线与边界

- [x] 记录基线：Go、Python、前端测试/构建/lint 当前通过。
- [x] 开发前加载 backend、frontend 规范及跨层/复用指南。
- [x] 保持一个 Trellis 任务：主题、采集、排序、推送和 UI 共享同一契约，拆成子任务会增加中间不兼容状态。

## 1. 固化主题与来源配置

- [x] 为 `KeywordRepo` 增加仅首次安装生效的 AI 默认主题。
- [x] 让 `CollectionRunner` 读取启用主题，移除硬编码查询；无主题时安全跳过搜索型采集。
- [x] 在 `types.RadarSources`、`SourceConfigRepo.EnsureDefaults` 和来源设置 API 中加入 RSS。
- [x] 扩展来源设置入口：Reddit communities、GitHub repositories、RSS feeds，完成 trim、去重、数量与格式校验。
- [x] 更新 Go API/runner 测试，确认配置从 DB 到 Python 参数完整传递。

回滚点：来源配置仍是 JSON，删除 RSS 默认行并恢复旧 `collectorArgs` 即可，不改变信号表。

## 2. 扩展 Python 采集器

- [x] 把 `shortlist` 改为接收动态主题；保留 AI 默认别名，加入非 AI 主题、无匹配和关注 Release 测试。
- [x] GitHub 仓库搜索改为逐主题串行、合并去重；保留 README 与 Release 证据。
- [x] 为明确关注仓库创建最新 Release 候选，以 Release HTML URL 保证每个版本独立幂等。
- [x] 用标准库实现 RSS 2.0 / Atom 解析器及常见日期、链接、摘要处理。
- [x] CLI 增加 `--topics`、`--repositories`、`--feeds` 和 `rss` 来源，保持现有 JSON 输出/入库合同。
- [x] 补充 Python 单元测试：RSS/Atom、坏 Feed、主题筛选、GitHub 多主题与 Release 去重。

验证：

```bash
cd services/collector
UV_CACHE_DIR=/tmp/trend-graph-uv-cache uv run --no-sync python -m unittest discover -s tests -v
```

## 3. 统一资格、分析与排序

- [x] `AnalysisRunner` 读取当前启用主题并传给 `Qualify` 与 Analyzer；修复其忽略 `shanghaiDay` 错误的现有小问题。
- [x] `Qualify` 改为主题驱动，增加 RSS 证据和关注 Release 例外，补充表驱动测试。
- [x] Analyzer 输入/输出增加 `matchedTopics` 与 `valueScore`，限制取值并验证模型返回。
- [x] 实现纯函数 `RankSignals` 与摘要选择：价值分、时效、重磅、URL/标题去重、单主题上限。
- [x] 看板 Handler 和 Digest 共用排序选择逻辑，响应增加 `rankScore`。
- [x] 补充旧分析无新字段的兼容测试。

回滚点：新增分析字段位于 JSON 中，旧消费者会忽略；排序函数可独立回退。

## 4. 生命周期与推送去重

- [x] 新增生命周期常量，迁移 `new/queued/practiced` 到 `inbox/saved/done`，保留 `dismissed`。
- [x] 更新 Signal Repo、Radar API、内容生成前置条件及相关测试。
- [x] Signal 新增 `LastDeliveredAt`；列表和摘要排除已忽略/已推送内容。
- [x] DeliveryService 在发送成功后标记信号，失败不标记；摘要和重磅提醒共享去重规则。
- [x] 增加重磅→摘要、已推送排除和发送失败重试测试。

回滚点：生命周期可反向更新；`LastDeliveredAt` 可保持为空/忽略，不需要删除列。

## 5. 重做个人收件箱前端

- [x] `App.tsx` 同时加载信号、来源和关注主题，并提供主题/来源更新动作。
- [x] 把 Dashboard 主信息架构改成收件箱、稍后处理、已处理、采集队列。
- [x] 卡片增加匹配主题和价值信息；动作改为保存、完成、忽略，内容生成降为已处理后的次级入口。
- [x] 在设置区复用关键词 API，提供最小主题表单；隐藏无效的旧每关键词调度间隔。
- [x] 增加 GitHub 关注仓库与 RSS Feed 文本设置，保留 Reddit 白名单。
- [x] 更新前端类型和 RadarDashboard 测试，覆盖分区、可选内容入口和来源设置。

## 6. 删除确认不可达的前端遗留

- [x] 用 `rg` 再次确认旧 HotList/Graph 闭环没有被当前入口引用。
- [x] 删除旧页面、专用 HotCard/RelevanceBar/SourceBadge、旧 graph API 和旧 WebSocket hook；保留被主题设置复用的关键词 API。
- [x] 移除 `reactflow` 并更新 lockfile。
- [x] 清理只服务旧前端的 TypeScript 类型，不动后端历史表/接口。

回滚点：删除前先形成独立 diff；若发现当前入口或测试依赖，跳过对应文件，不为清理扩大范围。

## 7. 文档、规范与完整验证

- [x] 更新 README、本地配置示例和产品文案，列出真实来源、主题、GitHub watchlist、RSS 和推送规则。
- [x] 更新 backend signal-radar contract 与 frontend component guidelines，记录新生命周期、排序和设置契约。
- [ ] 执行格式化和全量检查：

```bash
cd backend && gofmt -w <changed-go-files> && go test ./...
cd services/collector && UV_CACHE_DIR=/tmp/trend-graph-uv-cache uv run --no-sync python -m unittest discover -s tests -v
cd frontend && npm test -- --run && npm run build && npm run lint
```

- [x] 启动安全本地示例数据做桌面/手机浏览器验收：主题、来源设置、收件箱状态移动、空态与控制台错误。
- [x] 审查 `git diff --check`、变更文件和数据迁移，不混入构建产物或无关文件。
- [x] 运行 Trellis 完整质量检查并更新必要规范。

## 完成门槛

- PRD 的 AC1–AC11 全部可验证。
- 新采集来源失败不影响其他来源。
- 看板与摘要不存在两套排序规则。
- 没有引入新运行时依赖。
- 用户确认规划后才执行 `task.py start` 并进入实现。
