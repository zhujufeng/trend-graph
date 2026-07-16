# trend-graph

> 面向个人创作者的 AI 信号雷达：收集可实践的一手资料，保留原始证据，再生成中文分析和多平台内容素材。

## 项目简介

当前产品聚焦 AI、Agent、Skill、MCP 和可落地的效率实践。Go 后端每三小时调度 Python 采集器，先做免费官方 API 搜索，再为候选项抓取正文、README、发布信息或讨论线程；只有保存证据并通过确定性筛选后，才会调用 `deepseek-v4-pro` 分析。

当前采集源：

- DEV Community：无需密钥，按标签搜索并读取完整文章。
- GitHub：无需密钥也可使用；配置 `GITHUB_TOKEN` 可提高速率限额，详情保留 README 和最新 Release。
- Bluesky：无需密钥，按关键词搜索并读取讨论线程。
- Reddit：API 本身可用于符合条件的免费应用，但必须配置自己的 `REDDIT_CLIENT_ID` 和 `REDDIT_CLIENT_SECRET`；缺少凭证时会明确标记为采集失败，不会退回网页爬虫。

WaytoAGI、SkillsMP、Linux.do、B 站和通用热点源不再进入新信号雷达。X 仍是重要来源，但免费的官方 API 不满足关键词搜索需求，因此留待后续只读爬虫任务。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go · Gin · GORM · PostgreSQL |
| 采集 | Python 3.12+ · uv · 官方只读 API |
| AI | DeepSeek 官方 API（OpenAI 兼容） |
| 实时 | gorilla/websocket |
| 定时 | robfig/cron |
| 前端 | React 19 · TypeScript · Vite · Aceternity UI · TailwindCSS |
| 图谱 | React Flow（关联图谱） · ECharts（趋势图） |
| 通知 | WebSocket · SMTP 邮件 · 飞书/钉钉 Webhook |
| 部署 | VPS · Docker Compose |

## 项目结构

```
trend-graph/
├── backend/                 # Go 后端
│   ├── cmd/server/         # 程序入口
│   ├── internal/
│   │   ├── api/            # HTTP 路由 + Handler
│   │   ├── radar/          # 新信号分析、调度与推送
│   │   ├── crawler/        # 保留的旧热点采集代码（新调度不启用）
│   │   ├── ai/             # DeepSeek 接入
│   │   ├── analyzer/       # 查询扩展 / 真假识别 / 实体抽取
│   │   ├── graph/          # 关联图谱构建与查询
│   │   ├── notify/         # WebSocket / 邮件 / 飞书 / 钉钉
│   │   ├── scheduler/      # cron 定时任务
│   │   ├── store/          # 数据库（GORM）
│   │   ├── config/         # 配置加载
│   │   └── types/          # 公共类型定义
│   └── docs/               # 后端学习笔记
├── frontend/                # React + TS 前端
├── services/collector/      # uv 管理的免费 API 采集器
├── docs/                    # 项目文档与学习路线
├── docker-compose.yml       # 一键部署
└── README.md
```

## 学习路线（10 个阶段）

详见 [docs/ROADMAP.md](docs/ROADMAP.md)。

- [x] 阶段 0：项目骨架 + 环境准备
- [x] 阶段 1：HackerNews 单源抓取 + Gin 第一个 API
- [x] 阶段 2：数据库设计（GORM + PostgreSQL）
- [x] 阶段 3：接入 DeepSeek（查询扩展 + 摘要）
- [x] 阶段 4：React 前端骨架 + 热点列表页
- [x] 阶段 5：扩展其余 8 个信息源
- [x] 阶段 6：WebSocket 实时推送
- [x] 阶段 7：定时任务 + 多渠道通知
- [x] 阶段 8：🎯 关联图谱差异化亮点
- [x] 阶段 9：VPS 直接部署（systemd + nginx + Caddy HTTPS）
- [x] 阶段 10：README + 教学文档 + 简历亮点话术

## 本地运行

详见 [docs/LOCAL_SETUP.md](docs/LOCAL_SETUP.md)。

启用新采集链路至少需要在后端环境中配置：

```dotenv
INTERNAL_INGEST_SECRET=请生成一个足够长的随机值
COLLECTOR_DIR=/absolute/path/to/trend-graph/services/collector
GITHUB_TOKEN=可选
REDDIT_CLIENT_ID=仅启用Reddit时必填
REDDIT_CLIENT_SECRET=仅启用Reddit时必填
```

独立验证免费 API（不写数据库）：

```bash
cd services/collector
uv run --no-sync python -m signal_collector.cli --source dev --query mcp,ai --limit 5
uv run --no-sync python -m signal_collector.cli --source github --query "agent skill mcp" --limit 5
uv run --no-sync python -m signal_collector.cli --source bluesky --query "MCP,Claude Code,Codex" --limit 5
```

## 部署上线

详见 [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)（VPS + systemd + nginx + Caddy HTTPS）。

## 学习文档

每个阶段都有详细教学笔记，配合源码阅读：

- [docs/ROADMAP.md](docs/ROADMAP.md) — 10 阶段学习路线总览
- [docs/LOCAL_SETUP.md](docs/LOCAL_SETUP.md) — 本地环境准备
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) — VPS 部署指南
- [docs/STAGE-1.md](docs/STAGE-1.md) — HackerNews 单源抓取 + Gin
- [docs/STAGE-2.md](docs/STAGE-2.md) — GORM + PostgreSQL 持久化
- [docs/STAGE-3.md](docs/STAGE-3.md) — DeepSeek AI 接入
- [docs/STAGE-4.md](docs/STAGE-4.md) — React + TS 前端骨架
- [docs/STAGE-5.md](docs/STAGE-5.md) — 9 源并发抓取
- [docs/STAGE-6.md](docs/STAGE-6.md) — WebSocket 实时推送
- [docs/STAGE-7.md](docs/STAGE-7.md) — 定时任务 + 多渠道通知
- [docs/STAGE-8.md](docs/STAGE-8.md) — 关联图谱差异化亮点
- [docs/LEARNING_NOTES.md](docs/LEARNING_NOTES.md) — Go + TS 知识图谱
- [docs/RESUME.md](docs/RESUME.md) — 简历亮点话术 + 面试题

## 致谢

本项目灵感来源于 [程序员鱼皮](https://github.com/liyupi) 的 [yupi-hot-monitor](https://github.com/liyupi/yupi-hot-monitor) 教学项目，特此致谢。本项目在其基础上重写了技术栈（Node.js → Go），并新增了关联图谱差异化能力。
