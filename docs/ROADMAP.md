# trend-graph 学习路线（Roadmap）

> 这是本项目的完整学习路线，10 个阶段循序渐进。每个阶段我会：
> 1. **先讲概念**：这一步要学什么、为什么这么做
> 2. **写带注释的代码**：每个文件解释清楚是什么、为什么这么写
> 3. **本地验证**：给出可以跑通的命令，确认无误再进下一阶段

---

## 阶段 0：项目骨架 + 环境准备 ✅

**目标**：搭好目录骨架，第一次 `git push` 到自己的 GitHub 仓库，确认本机 Go / Node / Postgres / Docker 环境齐全。

**学到的概念**：
- Monorepo 项目结构
- Go module 是什么、`go mod init` 干了什么
- Git 基本工作流（init → remote → commit → push）

**产出**：能 push 到自己 GitHub 的最小骨架。

---

## 阶段 1：HackerNews 单源抓取 + Gin 第一个 API

**目标**：用 Go 从 HackerNews 抓 N 条最新内容，写一个 `/api/hots` 接口返回 JSON。

**学到的概念**：
- Go 基础语法：`package`、`import`、`func`、`struct`、`interface`、`error`
- HTTP 客户端调用（`net/http` + `encoding/json`）
- Gin 框架：路由、Handler、JSON 响应
- go 项目分层：`cmd/` vs `internal/`

**产出**：访问 `http://localhost:8080/api/hots` 能拿到 HackerNews 数据。

---

## 阶段 2：数据库设计（GORM + PostgreSQL）

**目标**：把抓到的热点存进数据库，支持查询、按时间筛选。

**学到的概念**：
- PostgreSQL 基础（建库、建表、连接）
- GORM 用法：模型定义、自动迁移、增删改查
- 配置管理（`.env` + viper 或标准库）

**产出**：热点入库 + `/api/hots` 从数据库读。

---

## 阶段 3：接入 DeepSeek（查询扩展 + 摘要）

**目标**：调用 DeepSeek API，对抓到的内容做查询扩展（提高召回）和智能摘要。

**学到的概念**：
- 大模型 API 调用（OpenAI 兼容协议）
- Prompt 工程基础
- Go 中的 HTTP POST + JSON streaming
- Provider 抽象（为后续多模型留口子）

**产出**：一条热点进来，AI 能给出"这是不是真的 + 相关性 + 摘要"。

---

## 阶段 4：React 前端骨架 + 热点列表页

**目标**：用 Vite 起一个 React + TS 前端，调通后端接口，展示热点列表。

**学到的概念**：
- TypeScript 基础：类型、接口、泛型
- React 函数组件 + Hook（useState/useEffect）
- Vite 工程化、跨域处理
- TailwindCSS + 一个 UI 库的引入

**产出**：浏览器看到能滚动加载的热点列表。

---

## 阶段 5：扩展其余 8 个信息源

**目标**：把另外 8 个源（B 站 / 微博 / GitHub Trending / Reddit / Bing / Twitter / Linux.do / 知乎）挨个加进去，每个源一个文件，统一接口。

**学到的概念**：
- Go interface 设计：定义统一的 `Crawler` 接口
- colly 爬虫框架用法（HTML 解析、并发、限速）
- 各平台的 API/页面结构差异
- 并发抓取（goroutine + WaitGroup + channel）

**产出**：9 个源都能返回数据，并发抓取合并。

---

## 阶段 6：WebSocket 实时推送

**目标**：后端抓到新热点时，实时推到前端，不用刷新。

**学到的概念**：
- WebSocket 协议与 HTTP 的区别
- gorilla/websocket 用法
- 前端 useWebSocket Hook
- 连接管理（心跳、重连、广播）

**产出**：网页挂着不动，新热点自动冒出来。

---

## 阶段 7：定时任务 + 多渠道通知

**目标**：用 cron 定时跑监控，命中关键词就推 WebSocket + 邮件 + 飞书 + 钉钉。

**学到的概念**：
- robfig/cron 用法、cron 表达式
- Go SMTP 发邮件
- Webhook 推送飞书/钉钉
- 通知节流（避免刷屏）

**产出**：设个关键词"AI"，5 分钟查一次，命中即通知。

---

## 阶段 8：🎯 关联图谱差异化亮点

**目标**：用 DeepSeek 抽取热点中的「实体」和「主题」，构建 **关键词 ↔ 实体 ↔ 热点** 三重网络图，前端 React Flow 可交互、可下钻。

**学到的概念**：
- 大模型信息抽取（NER/主题抽取）
- 图数据建模（节点/边/权重）
- React Flow 可交互网络图
- 图谱查询与下钻

**产出**：页面上一张可拖动、可点击节点展开相关热点的网络图——**这就是你简历那行写得出彩的亮点**。

---

## 阶段 9：VPS 直接部署（systemd + nginx + Caddy HTTPS）

**目标**：用 systemd + nginx 把项目部署到 VPS，绑域名加 HTTPS。

**学到的概念**：
- systemd 服务单元文件
- nginx 反向代理 + 静态托管
- WebSocket 反代特殊头
- Caddy 自动签 Let's Encrypt 证书

**产出**：公网可访问的 https://你的域名.com

详见 [DEPLOYMENT.md](DEPLOYMENT.md)。

---

## 阶段 10：README + 简历亮点话术

**目标**：完善 README 和文档，整理简历能写的亮点和面试话术。

**产出**：能拿出去给面试官看的项目。

---

每完成一个阶段记得：
1. 在 `README.md` 里勾选 ✅
2. 写一篇学习笔记到 `backend/docs/` 或 `docs/`
3. 提交并 push 到 GitHub