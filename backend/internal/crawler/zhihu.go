// Package crawler - 知乎热榜爬虫
//
// 知乎热榜公开 API：
//   GET https://www.zhihu.com/api/v3/feed/topstory/hot-lists/total?limit=50
// 返回 JSON，无登录可访问。但官方有时会校验 Referer/UA。
//
// 关键词过滤：标题包含关键词
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// ZhihuCrawler 知乎热榜爬虫
type ZhihuCrawler struct{}

// NewZhihuCrawler 构造
func NewZhihuCrawler() *ZhihuCrawler { return &ZhihuCrawler{} }

// Source 实现 types.Crawler
func (c *ZhihuCrawler) Source() string { return "zhihu" }

// zhihuResp 知乎热榜响应
// JSON 结构比较深，只挑我们关心的字段
type zhihuResp struct {
	Data []struct {
		Target struct {
			ID      int64  `json:"id"`        // 问题 ID
			Title   string `json:"title"`
			Excerpt string `json:"excerpt"`   // 摘要
			Author  struct {
				Name string `json:"name"`
			} `json:"author"`
			CreatedTime int64 `json:"created"` // 创建时间
		} `json:"target"`
		DetailText string `json:"detail_text"`
	} `json:"data"`
}

// Fetch 实现 types.Crawler
func (c *ZhihuCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 知乎热榜接口
	url := fmt.Sprintf("https://www.zhihu.com/api/v3/feed/topstory/hot-lists/total?limit=%d", limit*2)
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36",
		"Referer":         "https://www.zhihu.com/hot",
		"Accept":          "application/json",
		"Accept-Language": "zh-CN,zh;q=0.9",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, fmt.Errorf("拉取知乎热榜失败: %w", err)
	}

	var resp zhihuResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析知乎 JSON 失败: %w", err)
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.Data))
	for i, d := range resp.Data {
		itemURL := fmt.Sprintf("https://www.zhihu.com/question/%d", d.Target.ID)
		// 关键词过滤
		if keyword != "" {
			combined := d.Target.Title + " " + d.Target.Excerpt
			if !strings.Contains(strings.ToLower(combined), strings.ToLower(keyword)) {
				continue
			}
		}
		// 知乎热榜没给"热度数值"，用排名估算：靠前=热
		hot := 1000 - i*50
		if hot < 0 {
			hot = 0
		}

		items = append(items, types.HotItem{
			Title:       d.Target.Title,
			URL:         itemURL,
			Summary:     d.Target.Excerpt,
			Source:      c.Source(),
			Hot:         hot,
			Author:      d.Target.Author.Name,
			PublishedAt: d.Target.CreatedTime,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}