# 阶段 5：扩展其余 8 个信息源 + 多源并发调度

> 对应 commit：`feat: stage 5 - 9 源聚合 + 多源并发调度`

## 🎯 目标

- 把 9 个信息源都实现 `types.Crawler` 接口
- 写 `MultiCrawler` 并发调度器，9 源同时抓
- 单源失败不影响其他源

## 📚 学到的概念

### 1. interface 抽象 N 个不同实现

```go
type Crawler interface {
    Source() string
    Fetch(keyword string, limit int) ([]HotItem, error)
}
```

9 个源各自实现这个接口，调用方一视同仁。这就是 Go interface 的精髓。

### 2. goroutine + sync.WaitGroup

```go
var wg sync.WaitGroup
for _, c := range crawlers {
    wg.Add(1)
    crawler := c  // 关键：复制循环变量到局部
    go func() {
        defer wg.Done()
        // 用 crawler
    }()
}
wg.Wait()  // 阻塞到所有 goroutine 完成
```

- `Add(n)` 加计数器
- 每个 goroutine `Done()` 减 1
- `Wait()` 主线程等计数器归 0

### 3. sync.Mutex 保护共享 map

Go map 不是并发安全的，多 goroutine 写要加锁：

```go
var mu sync.Mutex
mu.Lock()
results[source] = items
mu.Unlock()
```

或用 `sync.RWMutex` 读写分离锁（读多写少时性能更好）。

### 4. 闭包陷阱（Go 经典坑）

```go
for _, c := range crawlers {
    go func() {
        c.Source()  // 错！所有 goroutine 共享同一个 c
    }()
}
```

Go 1.22 之前所有 goroutine 共享循环变量，最后一次循环的 c。修法：

```go
for _, c := range crawlers {
    c := c  // 复制到局部变量
    go func() { c.Source() }()
}
```

Go 1.22+ 修了这个坑，但显式复制仍是好习惯。

### 5. context.WithTimeout 跨调用控制超时

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()  // 必调，释放 ctx 资源
```

ctx 贯穿调用链，超时自动 cancel 所有下游请求。

### 6. 容错降级

```go
// Bing 失败时降级到 DuckDuckGo
items, err := c.fetchBing(keyword, limit)
if err == nil && len(items) > 0 {
    return items, nil
}
items2, err2 := c.fetchDuckDuckGo(keyword, limit)
```

爬虫的反爬应对策略：模拟浏览器头、降级备选源。

### 7. 错误收集模式

```go
results map[string][]HotItem
errors  map[string]error
```

把每个源的成功/失败分别收集，调用方可以查"哪个源失败"而不是"整体失败"。

### 8. HTML 解析 + 正则分块

```go
articleRe := regexp.MustCompile(`<article[^>]*class="[^"]*Box-row[^"]*"[^>]*>([\s\S]*?)</article>`)
blocks := articleRe.FindAllStringSubmatch(body, -1)
```

`[\s\S]*?` 非贪婪匹配跨行。`FindAllStringSubmatch` 返回 `[][]string`，每条 `[全文, 捕获组1, ...]`。

### 9. JSON API 调用 + struct tag

每个源的响应都定义 struct：

```go
type bilibiliResp struct {
    Code int `json:"code"`  // 0 = 成功
    Data struct {
        List []struct {
            Title string `json:"title"`
            // ...
        } `json:"list"`
    } `json:"data"`
}
```

嵌套 struct + tag 精确对齐后端 JSON。

### 10. builder 风格装配

```go
var crawlers []types.Crawler = []types.Crawler{
    crawler.NewHackerNewsCrawler(),
    crawler.NewGitHubCrawler(),
    // ...
}
multi := crawler.NewMultiCrawler(crawlers...)
```

切片 + 可变参数，注册新源只加一行。

## 🔍 9 个源对照

| 源 | 实现方式 | 难点 |
|---|---|---|
| HackerNews | 官方 JSON API（两步：先 ID 后详情） | 两次请求 |
| GitHub Trending | HTML 爬虫（正则解析 article） | 正则维护 |
| Reddit | r/all JSON API | 反爬，要浏览器 UA |
| Bing | HTML 爬虫 → DuckDuckGo 降级 | Captcha 应对 |
| B 站 | web-interface/popular JSON API | UA + Referer |
| 知乎 | hot-lists/total API | 需 cookie，401 常见 |
| 微博 | ajax/side/hotSearch API | 需 cookie |
| Linux.do | Discourse latest.json | Cloudflare 反爬 |
| Twitter | v2 search/recent API | 需 Bearer Token |

## 🧪 测试

```bash
go test -v -run TestMultiE2E ./internal/crawler/
# 看 9 源并发结果汇总
```

## 🐛 踩坑

1. **Reddit 403**：要带浏览器 UA + Accept-Language
2. **Bing 0 条 + Captcha**：sandbox 网络被识别，降级 DuckDuckGo
3. **`undefined: urlEncode`**：把公共函数放 `util.go`，不要散在各文件
4. **闭包陷阱**：循环变量 `c` 一定要复制到局部 `crawler := c`

## 📝 一句话总结

interface + goroutine + WaitGroup + Mutex 是 Go 并发的"黄金组合"，9 源并发抓取是这套组合的典型应用。