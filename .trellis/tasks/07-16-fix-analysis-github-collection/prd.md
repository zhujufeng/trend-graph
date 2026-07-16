# 修复 AI 分析与 GitHub 采集

## Goal

让首次采集后的 AI 信号能够稳定完成分析，并把仪表盘恢复成可执行的个人工作流：发现真实案例、加入实践、确认实践、再生成二创内容。

## Background

- PostgreSQL、DeepSeek 凭证和 `deepseek-v4-pro` 已实测可用。
- 当前待分析证据长度从数百字符到 410,848 字符；`AnalyzeSignal` 将全文放进提示词，并只允许 1,200 个输出 token，实测返回截断 JSON。
- GitHub 每个候选当前重复读取仓库元数据，再读取 README 和 release；20 个候选连同搜索共 61 次请求，超过匿名每小时 60 次额度。
- GitHub 单个候选缺 README 或请求失败会中断整轮，与现有信号雷达合同冲突。

## Requirements

- 分析提示词必须对超长证据设置确定上限，同时保留开头的项目说明和结尾的最新 release 信息；数据库中的原始证据不得修改。
- 模型输出预算必须足够容纳现有结构化字段，并在模型因长度停止时返回明确错误，不得把残缺 JSON 持久化。
- GitHub 详情抓取不得重复请求搜索结果已经提供的仓库元数据。
- 单个候选的详情抓取或入库失败不得阻止后续候选；如果整批候选全部失败，采集命令仍需失败以便来源健康状态可见。
- 不新增依赖，不改变来源、调度、推送或数据库结构。
- 已分析信号必须优先于待分析队列；同一信号在首页只出现一次。
- 待分析信号不得把原始 README/文章正文送进页面 DOM，只显示来源、标题和时间。
- 雷达列表查询和响应不得加载或传输原始证据正文；完整证据只供单条分析和内容生成使用。
- 复用现有 `signals.lifecycle_state` 实现 `new -> queued -> practiced` 以及 `dismissed`；只有 qualified 信号可改变实践状态。
- 内容包创建必须要求 `lifecycle_state=practiced`，避免未经本人实践就直接产出“可发布”内容。
- 来源设置属于低频管理操作，默认折叠，不得抢占日常工作区。

## Test Seams

- Go 包公开入口 `(*analyzer.Analyzer).AnalyzeSignal`：验证发送给 AI 的证据有界、首尾保留、输出预算和截断响应处理。
- Python 包入口 `GitHubCollector.search/fetch_detail` 与采集 CLI：验证请求次数、单候选失败继续、整批失败报错。
- HTTP `PATCH /api/radar/signals/:id/lifecycle`：验证状态白名单、qualified 限制和持久化。
- React `RadarDashboard`：验证每条信号只进入一个工作区、原始 evidence 不进入 HTML、内容生成只对已实践信号开放。

## Acceptance Criteria

- [x] 超过上限的 UTF-8 证据不会产生非法文本，AI 请求同时包含正文开头和结尾，原始证据不被修改。
- [x] `finish_reason=length` 被明确拒绝；有效结构化响应仍保留 token 用量并通过原有校验。
- [x] 20 个 GitHub 候选最多使用 41 次 GitHub API 请求，匿名额度足够时可完成整轮。
- [x] 一个 GitHub 候选缺少 README 时，后续有效候选仍能抓取和入库；全部失败时命令非零退出。
- [x] 后端 Go 测试和 collector Python 测试全部通过。
- [x] 使用当前本地配置完成一次真实 AI 分析 smoke test，页面出现至少一条已分析结果；若外部 API 额度未重置，明确记录该外部限制。
- [x] 首页先展示待实践和已实践内容，待分析队列折叠在后；同一信号不重复出现在多个栏目。
- [x] 待分析信号的完整 evidence 不出现在 HTML 中。
- [x] 雷达列表 API 不返回 evidence excerpt，列表数据库查询也不读取正文列。
- [x] 用户可以把 qualified 信号加入实践、标记已实践或忽略，刷新后状态仍保留。
- [x] 未实践信号不能创建内容包；已实践信号可以进入小红书、公众号、X 中英内容工作台。
- [x] 前端测试、构建、lint，后端完整测试与浏览器核心路径全部通过。

## Out of Scope

- Reddit 凭证配置。
- GitHub 付费/API token 方案。
- 自动发布或图片生成。
