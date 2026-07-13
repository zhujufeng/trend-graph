// Package crawler - Twitter / X 爬虫
//
// Twitter 没有无需 Key 的 API，使用官方 v2 API 需要：
//   - TWITTER_BEARER_TOKEN (Bearer token)
//   - 或者 OAuth1 + access token
//
// 配置方式：通过环境变量 TWITTER_BEARER_TOKEN 传给爬虫。
// 如果没有 token，Fetch 会返回 ErrTwitterNoToken 让上层跳过该源。
//
// 接口：GET https://api.twitter.com/2/tweets/search/recent
//     ?query={keyword}&max_results={limit}&tweet.fields=public_metrics,created_at,author_id
//
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// TwitterCrawler 推特爬虫
type TwitterCrawler struct {
	// bearerToken 是 Twitter API 的凭据
	bearerToken string
}

// NewTwitterCrawler 从环境变量读 token 构造
//
// 如果没配 token 仍然返回实例，但 Fetch 时会返回特定错误，
// 让调用层可以优雅跳过这个源而不是整个抓取失败。
func NewTwitterCrawler() *TwitterCrawler {
	return &TwitterCrawler{bearerToken: os.Getenv("TWITTER_BEARER_TOKEN")}
}

// Source 实现 types.Crawler
func (c *TwitterCrawler) Source() string { return "twitter" }

// ErrTwitterNoToken 是无 token 的哨兵错误，调用方用 errors.Is 判断
var ErrTwitterNoToken = fmt.Errorf("Twitter 未配置 TWITTER_BEARER_TOKEN")

// twitterResp 推特 v2 search/recent 响应
type twitterResp struct {
	Data []struct {
		ID         string    `json:"id"`
		Text       string    `json:"text"`
		CreatedAt  time.Time `json:"created_at"` // ISO 8601
		AuthorID   string    `json:"author_id"`
		PublicMetrics struct {
			Impressions   int `json:"impression_count"`
			Retweets      int `json:"retweet_count"`
			Replies       int `json:"reply_count"`
			Likes         int `json:"like_count"`
			Quotes        int `json:"quote_count"`
			Bookmarks     int `json:"bookmark_count"`
		} `json:"public_metrics"`
	} `json:"data"`
	Meta struct {
		ResultCount int `json:"result_count"`
	} `json:"meta"`
	Includes struct {
		Users []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			Name     string `json:"name"`
		} `json:"users"`
	} `json:"includes"`
}

// Fetch 实现 types.Crawler
func (c *TwitterCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if c.bearerToken == "" {
		return nil, ErrTwitterNoToken
	}
	if keyword == "" {
		// Twitter 必须有 query，没传就报错
		return nil, fmt.Errorf("Twitter 源要求提供 keyword")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// 拼查询：去掉空格拼成 URL 编码过
	// 包含 author username 让调用方拿到作者名
	url := fmt.Sprintf(
		"https://api.twitter.com/2/tweets/search/recent?query=%s&max_results=%d&tweet.fields=public_metrics,created_at,author_id&expansions=author_id&user.fields=username,name",
		urlEncode(keyword), limit,
	)
	headers := map[string]string{
		"Authorization": "Bearer " + c.bearerToken,
		"Accept":        "application/json",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, fmt.Errorf("拉取 Twitter 失败: %w", err)
	}

	var resp twitterResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析 Twitter JSON 失败: %w", err)
	}

	// 构造 author id → username 映射
	authorMap := make(map[string]string, len(resp.Includes.Users))
	for _, u := range resp.Includes.Users {
		authorMap[u.ID] = u.Username
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.Data))
	for _, tw := range resp.Data {
		// 标题：推特没有独立标题字段，用前 60 字符
		title := tw.Text
		if len([]rune(title)) > 60 {
			title = string([]rune(title)[:60]) + "…"
		}
		// 链接：https://twitter.com/{username}/status/{id}
		username := authorMap[tw.AuthorID]
		itemURL := fmt.Sprintf("https://twitter.com/%s/status/%s", username, tw.ID)

		// 热度：点赞+转发*3+回复*2
		hot := tw.PublicMetrics.Likes + tw.PublicMetrics.Retweets*3 + tw.PublicMetrics.Replies*2

		items = append(items, types.HotItem{
			Title:       strings.ReplaceAll(title, "\n", " "),
			URL:         itemURL,
			Summary:     tw.Text,
			Source:      c.Source(),
			Hot:         hot,
			Author:      username,
			PublishedAt: tw.CreatedAt.Unix(),
			FetchedAt:   now,
		})
	}

	return items, nil
}