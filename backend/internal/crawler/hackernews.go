// Package crawler 实现 9 个信息源各自的爬虫。
//
// 每个信息源各一个文件，都实现 types.Crawler 接口，
// 这样上层调度时一视同仁、方便并发。
//
// 这是本阶段（阶段 1）的阶段性文件，一次只写一个 HackerNews，
// 阶段 5 会按同样的套路把另外 8 个源一个一个加进来。
package crawler

// 阶段 1 的导入讲解：
// - context: Go 的超时/取消传递机制，HTTP 请求必须用
// - encoding/json: 把 JSON 字符串解析成 Go struct
// - fmt: 格式化错误信息
// - net/http: 发 HTTP 请求
// - time: 设置超时
// - trend-graph/internal/types: 引用本项目定义的 HotItem / Crawler
import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"trend-graph/internal/types"
)

// HackerNewsCrawler 是 HackerNews 爬虫 struct。
//
// 它没有任何字段，因为它只用 HN 公开 API，不需要 API Key 或状态。
// 后面接 B 站、Reddit 时，struct 可能会有 clientID / apiKey 等字段。
//
// 它通过"实现 Fetch 和 Source 方法"自动满足 types.Crawler 接口，
// Go 不要求显式声明 implements，这点和 Java/C# 不同。
type HackerNewsCrawler struct{}

// NewHackerNewsCrawler 是构造函数。
// Go 没"构造函数"关键字，约定用 New 前缀。
// 返回指针 *HackerNewsCrawler 让调用方共享同一个实例。
func NewHackerNewsCrawler() *HackerNewsCrawler {
	return &HackerNewsCrawler{}
}

// Source 返回信息源标识字符串 "hn"。
// 这是实现 types.Crawler 接口的第一方法。
func (c *HackerNewsCrawler) Source() string {
	return "hn"
}

// HackerNews 的 API 设计有点"反人类"，分两步：
// 1) GET /v0/topstories.json 返回一个 [N]int 的数组，只是 ID 列表
// 2) GET /v0/item/<id>.json 才返回具体一条帖子的内容
// 所以抓 N 条要 N+1 次请求。我们这里一次先抓够 limit 条。

// hnStory 是 HN item API 返回的原始结构。
// 字段用指针类型（*string *int）是为了区分"字段缺失"和"零值"：
// - nil 表示 HN 没返回这个字段
// - "空字符串" 也是一种值，对应 *string = &""
// 这是 Go 处理 JSON optional 字段的标准做法。
type hnStory struct {
	Title string  `json:"title"`     // 标题
	URL   *string `json:"url"`       // 链接（Ask HN 类帖子没有 url，是 self 文本）
	Score *int    `json:"score"`     // 热度分数
	By    string  `json:"by"`        // 作者
	Time  int64   `json:"time"`      // 发布时间戳（秒）
	Type  string  `json:"type"`      // item 类型：story/job/comment...
}

// Fetch 实现 types.Crawler 接口的第二个方法。
// 流程分三步：
//   1) 拿 top stories 的 ID 列表
//   2) 取前 limit 个 ID 并发拉详情（本阶段先顺序拉简单起见）
//   3) 把 hnStory 转成统一的 HotItem
func (c *HackerNewsCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	// 防御：limit 不合法就规整一下
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 用 http.Client 设置 10 秒超时，避免外部 API 卡死我们。
	// 注意：超时设置是个长期工程，阶段 5 接 9 源时要整体调优。
	client := &http.Client{Timeout: 10 * time.Second}

	// 第 1 步：拉 ID 列表
	// http.Get 是简化写法（不传 header），生产代码用 client.Do 更灵活
	idsURL := "https://hacker-news.firebaseio.com/v0/topstories.json"
	req, err := http.NewRequestWithContext(context.Background(), "GET", idsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("拉取 HN top stories 失败: %w", err)
	}
	// Go 没有 try/finally，资源清理用 defer，会自动在函数返回前执行
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN top stories 非 200: %d", resp.StatusCode)
	}

	// 把响应体解析成 []int（HN 返回的是 JSON 数组）
	var ids []int
	if err := json.NewDecoder(resp.Body).Decode(&ids); err != nil {
		return nil, fmt.Errorf("解析 HN top stories 失败: %w", err)
	}

	// 截断到 limit 条（容量可能很大，但只取前 N 个）
	if len(ids) > limit {
		ids = ids[:limit]
	}

	// 第 2 步：依次拉每条故事详情
	// 阶段 1 先用循环顺序拉简单清晰；阶段 5 会改并发拉
	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(ids))
	for _, id := range ids {
		story, err := c.fetchStory(client, id)
		if err != nil {
			// 单条失败不中断整体，跳过继续——爬虫的韧性
			// 真实项目可换成把错误记到日志，本期不展开
			continue
		}
		// 只要 story 类型，过滤掉 job / comment / poll
		if story.Type != "story" {
			continue
		}

		// 第 3 步：hnStory → HotItem 转换
		item := types.HotItem{
			Title:       story.Title,
			URL:         "", // 默认空，下面再判
			Summary:     "", // 阶段 3 让 DeepSeek 填
			Source:      c.Source(),
			Hot:         0,
			Author:      story.By,
			PublishedAt: story.Time,
			FetchedAt:   now,
		}
		if story.URL != nil {
			item.URL = *story.URL // 解引用拿到字符串
		}
		if story.Score != nil {
			item.Hot = *story.Score
		}

		// 阶段 1 暂不做关键词过滤：先把 HN top 整批返回
		// 阶段 3 接 AI 后，让 AI 判断相关性、keyword 命中再过滤
		_ = keyword

		items = append(items, item)
	}

	return items, nil
}

// fetchStory 拉 HN 单条 item 详情。
//
// Go 习惯对小写开头的方法视为包内私有（不可被其他包调用）。
// 大写开头（如 Fetch / Source）才是公开的。
func (c *HackerNewsCrawler) fetchStory(client *http.Client, id int) (*hnStory, error) {
	// fmt.Sprintf 把 %d 替换成 id，组装 URL
	url := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HN item %d 非 200: %d", id, resp.StatusCode)
	}
	var story hnStory
	if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
		return nil, err
	}
	return &story, nil
}