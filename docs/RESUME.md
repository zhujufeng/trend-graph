# 简历亮点话术 + 面试题

## 一、简历项目段落（可直接用）

### 简洁版（3 行）

> **trend-graph · AI 热点监控 + 关联图谱工具** | Go · TypeScript · PostgreSQL · DeepSeek
> 基于 yupi-hot-monitor 二次开发，纯 Go + TypeScript 全栈实现：9 源并发抓取（HN/GitHub/B站/微博/知乎/Reddit/Bing/Twitter/Linux.do）、DeepSeek AI 综合分析、关键词-实体-热点三重关联图谱（差异化亮点）、WebSocket 实时推送、cron 定时任务 + 多渠道通知（邮件/飞书/钉钉），VPS 部署上线。
> 仓库：https://github.com/zhujufeng/trend-graph

### 详尽版（5~6 行，适合校招/实习简历）

> **trend-graph · AI 热点监控 + 关联图谱工具** | Go · TypeScript · PostgreSQL · React · DeepSeek
>
> - 基于 yupi-hot-monitor 二次开发，重写后端为 Go，新增**关联图谱**差异化能力
> - **后端**：Go + Gin + GORM + PostgreSQL，定义统一 Crawler 接口实现 9 个信息源（HackerNews / GitHub Trending / B站 / 微博 / 知乎 / Reddit / Bing / Twitter / Linux.do），用 goroutine + WaitGroup + Mutex 实现 9 源并发抓取，单源失败容错降级
> - **AI 分析**：接入 DeepSeek 大模型（OpenAI 兼容协议），实现查询扩展、综合分析（摘要+相关性+真假识别+实体抽取），用 `response_format=json_object` + prompt 工程稳定输出结构化 JSON
> - **🎯 关联图谱（差异化亮点）**：设计关键词-实体-热点三重网络图，用 PG ON CONFLICT UPSERT + 复合唯一索引管理节点和关系，前端用 React Flow 实现可交互可视化（节点点击高亮相邻、按关系类型着色）
> - **实时推送**：gorilla/websocket Hub 模式 + Ping/Pong 心跳 + 自动重连，抓取/分析完成实时广播给在线客户端
> - **运维**：robfig/cron 定时任务 + 邮件/飞书/钉钉三渠道通知 + 节流策略，systemd + nginx + Caddy HTTPS 部署上线 VPS
> - 仓库：https://github.com/zhujufeng/trend-graph

## 二、可量化的"亮点动词"清单

简历里多用这些动词让描述更具体：

| 维度 | 动词 + 量化 |
|---|---|
| 并发 | "9 源并发抓取，goroutine + WaitGroup + Mutex" |
| 容错 | "单源失败不影响全局，Bing→DuckDuckGo 降级兜底" |
| AI | "一次调用拿到摘要+相关性+真假+实体 4 项结果" |
| 图谱 | "三重网络图，最多 100 节点 + 200 边" |
| 实时 | "WebSocket Hub 模式 + 心跳保活 + 自动重连" |
| 调度 | "每分钟重载关键词表，动态增删 cron 任务" |
| 通知 | "邮件 + 飞书 + 钉钉三渠道扇出，单渠道失败不影响其他" |
| 节流 | "relevance ≥ 0.5 才推送，避免刷屏" |
| 部署 | "systemd + nginx + Caddy HTTPS 上线" |

## 三、面试可能问到的题 + 参考答案

### Q1：为什么选 Go 不用 Node.js（原项目）？

> Go 在并发（goroutine 比 JS event loop 更轻量）、类型安全（编译期检查 vs TS 运行时）、二进制部署（无运行时依赖）上有优势；本项目 9 源并发抓取是 IO 密集场景，Go 的 goroutine + channel 模型比 Promise.all 更直观；学习目标也是 Go。

### Q2：goroutine 和线程有什么区别？为什么能开几千个不爆？

> goroutine 是用户态轻量级线程，初始栈 2KB（线程 1~8MB），由 Go runtime 调度不依赖操作系统；创建/切换成本低，可以轻松开 10 万个；但 IO 密集型场景才合适，CPU 密集会浪费调度。

### Q3：你怎么处理 9 个源里有些失败的情况？

> MultiCrawler 用 `map[string]error` 收集每个源的错误，成功的源照常返回；失败的源打日志但不阻塞整体；某些源（Bing）还有降级方案（DuckDuckGo 兜底）；这样保证 9 源至少有几个能成功就有数据。

### Q4：闭包陷阱是什么？你怎么解决的？

> Go 1.22 之前 for 循环变量在所有 goroutine 间共享，最后一次循环的值会覆盖前面的。解决：在循环体内复制 `c := c`，让每个 goroutine 持有自己的副本。React useEffect 也有类似问题，用 useRef 持有最新 callback 解决。

### Q5：WebSocket 怎么处理客户端失联？

> 三道防线：
> 1. 服务端定时（54s）发 Ping，浏览器自动回 Pong
> 2. 读超时 60s 没收到任何消息（含 Pong），认为失联，踢掉
> 3. 前端 onclose 自动重连（指数退避 + 最大次数限制）
>
> 这样 NAT 超时、网络抖动、客户端崩溃都能恢复。

### Q6：让 AI 返回 JSON 你怎么保证稳定？

> 三个手段：
> 1. 请求加 `response_format: {type: "json_object"}`，OpenAI/DeepSeek 都支持
> 2. system prompt 里给一个示例 JSON 让 AI 模仿
> 3. 解析失败兜底返回默认值（不报错让主流程继续）
>
> Temperature 调低到 0.2~0.3 也增加稳定性。

### Q7：关联图谱为什么不用 Neo4j 这种图数据库？

> 三点考虑：
> 1. 节点/边规模不大（百~千级），PostgreSQL 完全够用
> 2. 减少技术栈复杂度，本项目已经用 PG 存热点
> 3. 关系查询用 SQL JOIN 已够表达，引入 Neo4j 增加运维成本
>
> 如果未来节点数到百万级，再考虑迁移 Neo4j。

### Q8：ON CONFLICT 和"先 SELECT 再 INSERT/UPDATE"有什么区别？

> ON CONFLICT 是一次 SQL 完成 UPSERT，原子性保证不会并发冲突；先查再改是两次 SQL，并发时容易撞（虽然有事务可加锁但更复杂）。性能上 ON CONFLICT 一次往返，明显更快。

### Q9：节流策略 relevance ≥ 0.5 是怎么定的？

> 经验值：DeepSeek 对"AI"关键词的相关性评分，0.5 大致是"标题或正文明显出现关键词"的阈值；阈值太低会刷屏，太高会漏推送。生产中可以让用户在前端配置阈值。

### Q10：你这个项目最难的部分是什么？

> 阶段 8 关联图谱：要把"AI 抽实体 → 入库去重 → 建关系 → 查询成图"串成一条链路，涉及 PG ON CONFLICT UPSERT、复合唯一索引、GORM clause.OnConflict、多段 SQL 拼装、React Flow 自定义节点 + 高亮逻辑。改了好几遍 schema 才让 ON CONFLICT 不报错。

## 四、面试可现场演示的" Wow 点"

打开 https://你的域名.com 演示：

1. **实时性**：点"立即抓取 + AI"，看着卡片**逐条冒出**（WS 推送 analyze_done）
2. **多源**：选 source=all，看到 9 源汇总数据
3. **图谱**：点"图谱"按钮，看到关键词→实体→热点的网络图，点节点高亮相邻
4. **关键词管理**：加一个关键词设 1 分钟间隔，等机器人推送（开了飞书/钉钉的话）

这 4 个演示点足以让面试官记住你。