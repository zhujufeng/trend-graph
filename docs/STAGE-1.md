# 阶段 1：HackerNews 单源抓取 + Gin 第一个 API

> 完成时间：阶段 0 之后
> 对应 commit：`feat: stage 1 - HackerNews crawler + Gin API`

## 🎯 本阶段目标

- 用 Go 从 HackerNews 抓 N 条热门帖子
- 写一个 `/api/hots` HTTP 接口返回 JSON
- 第一次接触 Gin 框架

## 📚 学到的概念

### 1. Go 项目分层

```
backend/
├── cmd/server/         # 程序入口（main.go 放这）
├── internal/            # 项目私有包（Go 特有：外部不能 import）
│   ├── api/             # HTTP 路由 + Handler
│   ├── crawler/         # 业务逻辑：爬虫
│   └── types/           # 共享类型
└── go.mod
```

`internal/` 是 Go 编译器强制的私有包约定，别处项目不能 import。

### 2. struct + json tag

```go
type HotItem struct {
    Title   string `json:"title"`
    Source  string `json:"source"`
    Hot     int    `json:"hot"`
}
```

反引号里的 `json:"title"` 是 struct tag，告诉 `encoding/json` 序列化时用哪个字段名。前端拿到的 JSON 字段名就是 tag 里写的。

### 3. interface（接口）— Go 的多态

```go
type Crawler interface {
    Source() string
    Fetch(keyword string, limit int) ([]HotItem, error)
}
```

Go interface 是**隐式实现**：只要某 struct 实现了 Source 和 Fetch 方法，它就自动满足 `Crawler` 接口，不需要 `implements` 关键字。这是 Go 与 Java/C# 的关键区别。

### 4. 构造函数约定

Go 没有构造函数关键字，社区约定用 `New` 前缀：

```go
func NewHackerNewsCrawler() *HackerNewsCrawler {
    return &HackerNewsCrawler{}
}
```

### 5. HTTP 客户端调用

```go
client := &http.Client{Timeout: 10 * time.Second}
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
resp, err := client.Do(req)
defer resp.Body.Close()
```

- `defer` 是 Go 的资源清理机制，函数返回前自动调用（类似 finally）
- `defer resp.Body.Close()` 必写，否则连接泄漏

### 6. JSON 解码

```go
var data SomeStruct
json.NewDecoder(resp.Body).Decode(&data)
```

`&data` 是取地址，Go 所有传参都是值传递，要修改原变量必须传指针。

### 7. Gin 框架基础

```go
r := gin.Default()           // 自带 Logger + Recovery 中间件
r.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
r.Run(":8080")
```

`gin.H` 是 `map[string]interface{}` 的别名，方便写 JSON 响应。

### 8. 可选 JSON 字段用指针

```go
type hnStory struct {
    URL   *string `json:"url"`     // nil = HN 没返回这个字段
    Score *int    `json:"score"`
}
```

指针能区分"零值"和"缺失"，处理 API optional 字段必备。

## 🔍 关键代码位置

| 概念 | 文件 |
|---|---|
| HotItem 业务实体 | `backend/internal/types/types.go` |
| Crawler 接口定义 | `backend/internal/types/types.go` |
| HackerNews 爬虫实现 | `backend/internal/crawler/hackernews.go` |
| Gin 路由 + Handler | `backend/internal/api/handler.go` |
| 入口装配 | `backend/cmd/server/main.go` |

## 🧪 测试

```bash
cd backend
go test -v -run TestSource ./internal/crawler/             # 单元测试
go test -v -run TestHackerNewsFetch ./internal/crawler/    # 集成测试（真打 HN）
```

`_test.go` 结尾文件用 `go test` 运行，能访问同包私有字段。

## 🐛 踩坑记录

1. **`imported and not used`**：Go 严格 unused import，删掉没用到的 import
2. **`go get` 后 package.json 没更新**：sandbox 网络慢时偶发，手动 `go mod tidy`
3. **`go run` 后台进程卡住 shell**：用 `timeout 8 go run` 限时运行

## 📝 一句话总结

第一次跑通"HTTP 调外部 API → 解析 JSON → 用框架暴露接口"全链路，这是后端开发最基础的循环。