# trend-graph 🔥🕸️

> 基于 [yupi-hot-monitor](https://github.com/liyupi/yupi-hot-monitor) 二次开发，纯 **Go + TypeScript** 技术栈的 AI 热点监控与关联图谱工具。
>
> 个人学习 + 可上线产品 + 简历亮点，三位一体。

## 项目简介

输入要监控的关键词，系统会自动从 **9 个信息源**（HackerNews / B 站 / 微博热搜 / GitHub Trending / Reddit / Bing / Twitter / Linux.do / 知乎）聚合抓取热点内容，用 **DeepSeek 大模型**做查询扩展、真假识别、相关性分析和智能摘要，并构建 **关键词 ↔ 实体 ↔ 热点 三重关联图谱**（差异化亮点），通过 WebSocket / 邮件 / 飞书 / 钉钉 实时推送。

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.22+ · Gin · GORM · PostgreSQL/SQLite |
| 爬虫 | colly（Go 爬虫框架） |
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
│   │   ├── crawler/        # 9 个信息源各一个文件
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