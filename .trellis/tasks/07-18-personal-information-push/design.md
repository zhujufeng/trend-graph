# 个人信息推送产品技术设计

## 1. 设计结论

沿用现有 Go + Python + React 架构，只修正产品主线和几个共享契约：

1. 现有 `keywords` 成为新雷达的关注主题来源。
2. Python 采集器新增 RSS/Atom，并让现有搜索源消费动态主题。
3. GitHub 只做主题仓库发现和明确仓库 Release 关注。
4. Go 统一负责资格判断、AI 价值分析、确定性排序、摘要选择和推送去重。
5. React 改成个人收件箱，内容生成留在已处理内容的次级动作。

不引入推荐系统、消息队列、向量数据库、GraphQL 客户端、RSS 第三方库或新服务。

## 2. 当前问题与代码证据

- `backend/internal/radar/collector_runner.go` 的 `collectorArgs` 为 DEV、GitHub、Bluesky 写死查询，旧关键词管理没有进入新采集链路。
- `services/collector/signal_collector/qualification.py` 与 `backend/internal/radar/qualifier.go` 各自维护固定 AI 词表，形成重复且不可配置的产品边界。
- `backend/internal/store/signal_repo.go` 在限制结果数量前按不同来源不可比的原始 `score` 排序。
- `backend/internal/radar/digest.go` 直接取前八条，没有主题多样性或跨来源去重。
- `backend/internal/radar/delivery.go` 的运行级幂等只能阻止同一时段任务重跑，不能阻止同一信号进入早报、晚报和重磅提醒。
- `frontend/src/pages/RadarDashboard.tsx` 的主信息架构仍是“待实践 / 已实践可创作”，且内容生成固定占据核心路径。
- `frontend/src/pages/HotListPage.tsx`、`GraphPage.tsx` 及其组件不被当前 `App.tsx` 引用，是可删除的旧前端闭环。

## 3. 目标数据流

```text
启用主题 + 来源设置
        ↓
Go CollectionRunner 生成有限采集参数
        ↓
Python: DEV / GitHub / Bluesky / Reddit / RSS
        ↓
候选主题预筛 → 证据抓取 → 内部鉴权入库
        ↓
Go: 主题资格判断 → DeepSeek 结构化价值分析
        ↓
共享排序与去重
   ↙              ↘
个人收件箱       摘要 / 重磅提醒
   ↓
按需内容生成
```

Python 只做低成本候选过滤；Go 是资格、排序和推送规则的最终事实来源。两层共享相同输入主题，但不共享复制粘贴的固定产品词表。

## 4. 数据模型与兼容性

### 4.1 关注主题

复用 `store.Keyword`、`KeywordRepo` 和 `/api/keywords`：

- `Word`：主题搜索词，例如 `AI`、`机器人`、`量化投资`。
- `Note`：用户对主题的说明，保留。
- `Active`：是否进入采集和资格判断。
- `IntervalMin`、`LastFetchedAt`：旧调度兼容字段，当前新雷达 UI 不再展示或修改。

增加 `KeywordRepo.EnsureDefault()`：仅当 `keywords` 表从未有过记录（含软删除记录）时创建 `AI`。这样首次安装有默认值，用户主动删空后重启不会被强行恢复。

### 4.2 来源设置

继续使用 `SourceConfig.SettingsJSON`，设置形状由来源拥有：

```json
{
  "reddit": { "communities": ["r/localllama"] },
  "github": { "repositories": ["openai/codex"] },
  "rss": { "feeds": ["https://example.com/feed.xml"] }
}
```

API 入口负责 trim、去重、数量上限和格式校验：主题最多 10 个启用项、GitHub 仓库最多 20 个 `owner/repo`、Feed 最多 20 个 HTTP(S) URL。设置仍以各来源一行保存，不新增配置表。

### 4.3 Signal

新增：

```go
LastDeliveredAt *time.Time
```

生命周期值改为 `inbox | saved | done | dismissed`。启动迁移只更新旧字符串值，不删除或重建数据：

```text
new → inbox
queued → saved
practiced → done
```

内容生成的前置状态同步从 `practiced` 改为 `done`。

### 4.4 分析 JSON

在现有分析结构追加：

```json
{
  "matchedTopics": ["AI"],
  "valueScore": 5
}
```

- `matchedTopics` 只能取自当前启用主题，最多 3 个。
- `valueScore` 为 1–5，综合用户相关性、信息新意和可行动性。
- 旧分析缺少 `valueScore` 时按 3 处理，避免历史数据突然消失。

不新增独立评分表或信号-主题关联表；个人规模下，分析 JSON 足够支撑展示和摘要多样性。若未来需要按主题统计历史趋势，再增加规范化关联表。

## 5. 采集设计

### 5.1 动态主题

`CollectionRunner.Run` 同时读取来源配置和启用主题：

- 无启用主题时，DEV、Bluesky、Reddit、RSS 和 GitHub 主题搜索都跳过网络请求；只有配置了明确关注仓库时，GitHub 仍检查其 Release。
- 主题通过独立 `--topics` 参数传给 Python，用于统一预筛。
- 搜索型来源通过 `--query` 接收逗号分隔主题；GitHub 改为逐主题串行搜索并按仓库去重，与 DEV/Bluesky 当前行为一致。
- 默认 AI 主题在预筛时使用现有中英文 AI 别名；自定义主题第一版做大小写不敏感的直接匹配。

### 5.2 RSS/Atom

新增 `rss.py`，只使用 `urllib` 已有 HTTP 客户端和 `xml.etree.ElementTree`：

- 支持常见 RSS 2.0 `channel/item` 与 Atom `feed/entry`。
- 候选 ID 优先 `guid/id`，其次链接；链接必须为 HTTP(S)。
- 标题、摘要/内容、作者、发布时间来自 Feed。
- 证据类别为 `publisher_feed`，证据正文使用 Feed 提供内容；不跟进抓取任意网页 HTML。
- 单 Feed 解析失败记录为该来源本次失败，但不能破坏其他来源的既有故障隔离。

### 5.3 GitHub 官方公开 API

采用能力：

| 产品用途 | 官方端点 | 决策 |
|---|---|---|
| 按主题发现仓库 | `GET /search/repositories` | 使用；支持 topic、stars、pushed、archived 等限定，逐主题串行查询 |
| 读取 README | `GET /repos/{owner}/{repo}/readme` | 继续使用，作为原始文档证据 |
| 关注版本更新 | `GET /repos/{owner}/{repo}/releases` | 对明确关注仓库使用；Release 的 `html_url` 作为独立信号 URL |
| 用户公开 Stars | `GET /users/{username}/starred` | 第一版不使用；Star 是书签，不等于通知意愿 |
| 公共 Events | `GET /events` 等 | 不使用；高噪声且官方说明延迟可达数小时 |
| Discussions | GraphQL | 不使用；需要额外认证与 GraphQL 契约，第一版收益不足 |

官方约束：公开数据可匿名读取，但匿名主额度为每 IP 每小时 60 次，认证请求通常为每小时 5,000 次；搜索有独立更严格额度。实现限制主题/仓库数量、串行调用并清晰报告 `403/429`。Token 继续通过服务端环境变量传入，不进入浏览器或数据库。

参考：

- https://docs.github.com/en/rest/search/search
- https://docs.github.com/en/search-github/searching-on-github/searching-for-repositories
- https://docs.github.com/en/rest/releases/releases
- https://docs.github.com/en/rest/activity/starring
- https://docs.github.com/en/rest/activity/events
- https://docs.github.com/en/rest/using-the-rest-api/rate-limits-for-the-rest-api
- https://docs.github.com/en/rest/using-the-rest-api/best-practices-for-using-the-rest-api

## 6. 资格与排序

### 6.1 资格判断

`Qualify` 接收启用主题：

1. 来源必须受支持，时间在 30 天内，证据存在。
2. 普通候选的标题 + 证据必须匹配至少一个主题。
3. 明确关注仓库产生的 GitHub Release 可绕过文本主题命中，但仍要求可读取的 README 或 Release 原始证据。
4. Reddit、Bluesky、RSS 继续验证各自证据类别。

### 6.2 共享排序

新增纯函数 `RankSignals`，由看板和摘要共同调用：

```text
rankScore = valueScore × 10
          + recencyBonus       // 24h 内 +8；7d 内 +5；30d 内 +1
          + alertBonus         // alertEligible=true 时 +20
```

同分时按来源更新时间、创建时间、ID 逆序。原始 `Signal.Score` 仅保留作来源元数据，不参与跨来源排序。

选择结果时：

1. 跳过 `dismissed`。
2. 按规范化 URL 去重。
3. 按去标点、小写、压缩空白后的标题去重。
4. 摘要只选择 `inbox` 或 `saved`、`LastDeliveredAt == nil` 的信号。
5. 摘要每个首要匹配主题最多 2 条，总数最多 8 条。

## 7. 推送幂等

`DeliveryService` 在通知成功后，通过一个数据库事务同时更新 `Signal.LastDeliveredAt` 和 DeliveryRun 状态：

- 通知失败：DeliveryRun 标记 failed，不更新信号。
- 通知成功：事务内标记该批信号已推送并把 DeliveryRun 标记 sent。
- 若完成事务失败，返回错误并保留 running 记录，不能假装完整成功。

重磅提醒与摘要共享该字段，因此即时提醒过的内容不会再次进入摘要。

外部 Webhook 与数据库无法形成真正的分布式事务；第一版选择“发送失败可重试”而不是提前标记导致消息永久丢失。极少数“Webhook 已收、随后数据库不可用”的情况可能重复一次；代码用 `ponytail:` 注释标出该上限，只有实际发生时才引入 outbox。

## 8. API 与前端契约

### API

- 继续使用 `/api/keywords`，前端展示名改为“关注主题”。创建时忽略旧的自定义间隔，使用兼容默认值。
- `PUT /api/source-configs/:source` 增加 `githubRepositories`、`rssFeeds`，并保留 `redditCommunities`。
- `GET /api/radar/signals` 由 Handler 获取最多 100 条、统一排序去重后再应用请求 limit。
- Signal 响应增加 `rankScore`；`analysis.matchedTopics/valueScore` 继续来自结构化分析 JSON。
- 生命周期 PATCH 只接受 `inbox | saved | done | dismissed`。

### 前端

- 标题改为“个人信息雷达”，副标题回答“今天值得知道什么”。
- 主区：收件箱；次区：稍后处理；已处理与采集队列使用折叠区。
- 卡片显示来源、时间、匹配主题、价值说明、行动建议和原始证据入口。
- 操作为：稍后处理、标记完成、不感兴趣；已处理卡片提供次级“生成内容”入口。
- 设置区整合关注主题、来源开关、Reddit 社区、GitHub 关注仓库和 RSS Feed。
- 删除当前入口不可达的旧热点/图谱页面、专用组件、API、WebSocket hook及 `reactflow` 依赖；保留关键词 API 客户端供主题设置复用。

## 9. 错误处理与回滚

- 新来源继续使用现有 `CollectionRun` 记录失败原因；某来源失败不回滚其他来源成功结果。
- RSS XML、URL、GitHub 仓库名和来源设置都在入口校验；无效配置返回 400，不把坏 JSON 留进数据库。
- 生命周期迁移是字符串更新，可用反向映射回滚；新增列可留空，不影响旧数据。
- 若新排序出现问题，可回退 Handler/Digest 到原顺序，不需回滚采集或数据。
- 若 RSS 不稳定，可在 SourceConfig 中关闭，不影响其他来源。

## 10. 有意延后

- Star 自动导入、GitHub Discussions、Events、任意网页抓取和行为学习都没有足够的第一版收益。
- 后端旧热点/图谱仍有历史数据和文档依赖，本任务只清理不可达前端；等新产品稳定后再单独删除后端遗留。
