// Package crawler - Reddit 爬虫
//
// Reddit 有简单好用的 JSON API：
//   GET https://www.reddit.com/r/{subreddit}/hot.json?limit=N
// 不需要 OAuth，只要一个像浏览器的 User-Agent 头就行。
//
// 默认抓 r/all（聚合所有 subreddit 的热门）。
// keyword 过滤：标题或 subreddit 包含关键词。
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// RedditCrawler 抓 reddit.com/r/all
type RedditCrawler struct {
	// Subreddit 默认 "all"，可改 "programming" "golang" 等
	Subreddit string
}

// NewRedditCrawler 默认抓 r/all
func NewRedditCrawler() *RedditCrawler {
	return &RedditCrawler{Subreddit: "all"}
}

// Source 实现 types.Crawler
func (c *RedditCrawler) Source() string { return "reddit" }

// redditResponse 是 Reddit JSON API 的响应结构
// 只关心 data.children 这个数组，里面每条是一条帖子
type redditResponse struct {
	Data struct {
		Children []struct {
			Data struct {
				Title     string  `json:"title"`
				Permalink string  `json:"permalink"` // 相对路径 /r/.../comments/xxx
				URL       string  `json:"url"`       // 外链地址（如果是图片/外站）
				Score     int     `json:"score"`     // 上热分数
				Author    string  `json:"author"`
				Subreddit string  `json:"subreddit"`
				Created   float64 `json:"created"` // 发布时间（浮点秒）
				Selftext  string  `json:"selftext"`
			} `json:"data"`
		} `json:"children"`
	} `json:"data"`
}

// Fetch 实现 types.Crawler
func (c *RedditCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	sub := c.Subreddit
	if sub == "" {
		sub = "all"
	}
	url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=%d", sub, limit)

	// Reddit 一定要带浏览器 UA，否则会 429
	body, err := httpGet(context.Background(), url, map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept":          "application/json, text/plain, */*",
		"Accept-Language": "en-US,en;q=0.9",
		"Accept-Encoding": "gzip, deflate, br",
	})
	if err != nil {
		return nil, fmt.Errorf("拉取 Reddit 失败: %w", err)
	}

	var resp redditResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析 Reddit JSON 失败: %w", err)
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.Data.Children))

	for _, c := range resp.Data.Children {
		post := c.Data
		// Reddit permalink 是相对路径，要拼基础设施
		itemURL := "https://www.reddit.com" + post.Permalink

		// 关键词过滤
		if keyword != "" {
			combined := post.Title + " " + post.Subreddit + " " + post.Selftext
			if !strings.Contains(strings.ToLower(combined), strings.ToLower(keyword)) {
				continue
			}
		}

		// 摘要从自拍文本截前 200 字符
		summary := post.Selftext
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}

		items = append(items, types.HotItem{
			Title:       post.Title,
			URL:         itemURL,
			Summary:     summary,
			Source:      "reddit",
			Hot:         post.Score,
			Author:      post.Author,
			// created 是浮点秒，转 int64
			PublishedAt: int64(post.Created),
			FetchedAt:   now,
		})
	}

	return items, nil
}