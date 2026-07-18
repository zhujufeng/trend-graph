# trend-graph

一个面向个人使用的信息雷达：按你关注的主题收集公开信息，保留来源证据，用 AI 判断实际价值，再把少量高价值内容送进收件箱和飞书早晚报。

它不再把“小红书选题”当作产品主线。内容生成仍然保留，但只是你处理完一条信息后的可选动作。

## 产品流程

```text
关注主题 / 明确订阅
        ↓
公开 API 与 RSS/Atom 采集
        ↓
30 天时效 + 主题 + 来源证据筛选
        ↓
AI 提取事实、匹配主题、价值分与行动建议
        ↓
排序、URL/标题去重、每主题限额
        ↓
个人收件箱 + 飞书早晚报 / 重磅提醒
```

收件箱只有四个状态：

- `inbox`：新收件；
- `saved`：稍后处理；
- `done`：已经处理，可选生成内容草稿；
- `dismissed`：忽略，不再出现在当前产品列表。

## 当前信息来源

| 来源 | 公开接口 | 用途 | 密钥 |
| --- | --- | --- | --- |
| DEV Community | DEV API | 按主题发现并读取完整文章 | 不需要 |
| GitHub | REST Search、Contents、Releases | 主题仓库发现、README 证据、明确仓库的最新 Release | `GITHUB_TOKEN` 可选，建议配置以提高限额 |
| Bluesky | 官方 AppView API | 搜索帖子并读取讨论线程 | 不需要 |
| Reddit | 官方 OAuth API | 只读取配置的社区白名单 | 必须配置自己的 OAuth 凭据 |
| RSS / Atom | 发布方 Feed | 订阅博客、媒体、项目公告和任何垂直领域来源 | 不需要 |

GitHub 采用三条简单而稳定的链路：按关注主题串行调用仓库搜索；用 Contents API 读取 README；对设置中明确订阅的 `owner/repo` 获取最新 Release。它没有导入 Stars、用户 Events 或 Discussions，避免把个人信息流做成噪声聚合器。

## 排序与推送规则

AI 为合格信息输出 `matchedTopics` 和 `valueScore`（1–5）。后端统一计算：

```text
rankScore = valueScore × 10 + 时效加分 + 重磅加分
```

- 24 小时内 `+8`，7 天内 `+5`，30 天内 `+1`；
- 合格重磅提醒 `+20`；
- 同 URL 或规范化标题只保留最高分的一条；
- 摘要最多 8 条，每个首要主题最多 2 条；
- 成功推送后写入 `lastDeliveredAt`，重磅提醒和后续摘要都不会重复发送；失败不会标记，仍可重试。

## 技术结构

| 层 | 技术 |
| --- | --- |
| 后端 | Go · Gin · GORM · PostgreSQL |
| 采集器 | Python 3.12+ · uv · 标准库 RSS 解析 |
| AI | DeepSeek OpenAI-compatible API |
| 前端 | React 19 · TypeScript · Vite · Tailwind CSS |
| 调度 | robfig/cron，Go 负责调度和来源健康，Python 负责来源适配 |
| 通知 | 飞书富文本 Webhook |

旧热点、图谱和通用爬虫后端暂时保留用于历史数据兼容，但已经不进入新的收件箱前端和自动采集链路。

## 本地运行

### 1. 环境

- Go 1.26+
- Python 3.12+ 与 `uv`
- Node.js 20+
- PostgreSQL 15+

复制后端环境配置：

```bash
cp backend/.env.example backend/.env
```

至少填写：

```dotenv
DATABASE_URL=host=localhost port=5432 user=postgres password=postgres dbname=trend_graph sslmode=disable timezone=Asia/Shanghai
ADMIN_PASSWORD=replace-with-a-long-random-password
SESSION_COOKIE_SECURE=false
INTERNAL_INGEST_SECRET=replace-with-a-second-random-secret
COLLECTOR_DIR=../services/collector
DEEPSEEK_API_KEY=可选；不配置时采集结果停留在等待分析
GITHUB_TOKEN=可选
REDDIT_CLIENT_ID=仅启用Reddit时必填
REDDIT_CLIENT_SECRET=仅启用Reddit时必填
```

主题、GitHub 仓库、RSS Feed 与 Reddit 社区都在网页设置中维护，不需要写入环境变量。

### 2. 启动

```bash
cd backend
go run ./cmd/server
```

```bash
cd frontend
npm install
npm run dev
```

首次数据库为空时会创建 `AI` 默认主题。以后即使你删除全部主题，重启也不会擅自恢复；没有主题时，系统只采集你明确订阅的 GitHub Release。

### 3. 独立验证采集器

以下命令只输出 JSON，不写数据库（未提供内部写入配置时）：

```bash
cd services/collector
uv run --no-sync python -m signal_collector.cli --source dev --topics "AI,机器人" --query "AI,机器人" --limit 5
uv run --no-sync python -m signal_collector.cli --source github --topics "AI,数据库" --repositories "openai/codex" --limit 5
uv run --no-sync python -m signal_collector.cli --source rss --topics "机器人" --feeds "https://example.com/feed.xml" --limit 5
```

## 验证

```bash
cd backend
GOCACHE=/tmp/trend-graph-go-cache go test ./...

cd ../services/collector
UV_CACHE_DIR=/tmp/trend-graph-uv-cache uv run --no-sync python -m unittest discover -s tests -v

cd ../../frontend
npm test -- --run
npm run build
npm run lint
```

更完整的本地环境说明见 [docs/LOCAL_SETUP.md](docs/LOCAL_SETUP.md)，部署说明见 [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)。
