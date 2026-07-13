// Package crawler - B 站（Bilibili）爬虫
//
// 用 B 站公开的热门视频 API：
//   GET https://api.bilibili.com/x/web-interface/popular
//   返回 JSON，含分页 + 视频列表
//   不需要登录，不需要 cookie，UA 略像浏览器即可
//
// 关键词过滤：标题 + UP 主名 + 标签
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// BilibiliCrawler B 站热门爬虫
type BilibiliCrawler struct{}

// NewBilibiliCrawler 构造
func NewBilibiliCrawler() *BilibiliCrawler { return &BilibiliCrawler{} }

// Source 实现 types.Crawler
func (c *BilibiliCrawler) Source() string { return "bilibili" }

// bilibiliResp B 站热门 API 响应
type bilibiliResp struct {
	Code int `json:"code"` // 0=成功
	Data struct {
		List []struct {
			Aid   int    `json:"aid"`    // 视频 ID
			Bvid  string `json:"bvid"`   // BV 号（新一代 ID）
			Title string `json:"title"`
			Owner struct {
				Name string `json:"name"`
				Mid  int    `json:"mid"`
			} `json:"owner"`
			Stat struct {
				View   int `json:"view"`   // 播放量
				Danmu  int `json:"danmaku"`// 弹幕数
				Reply  int `json:"reply"`
				Favorite int `json:"favorite"`
				Like   int `json:"like"`
				Coin   int `json:"coin"`
				Share  int `json:"share"`
			} `json:"stat"`
			ShortLink   string `json:"short_link"`
			Desc        string `json:"desc"`
			PubDate     int64  `json:"pubdate"` // 发布时间（秒）
			Tag         string `json:"tname"`
		} `json:"list"`
	} `json:"data"`
}

// Fetch 实现 types.Crawler
func (c *BilibiliCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// popular API 一次最多返回 20 条，pn/pz 分页参数
	url := fmt.Sprintf("https://api.bilibili.com/x/web-interface/popular?pn=1&pz=%d", limit)
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36",
		"Referer":         "https://www.bilibili.com/",
		"Accept":          "application/json",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, fmt.Errorf("拉取 B 站热门失败: %w", err)
	}

	var resp bilibiliResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析 B 站 JSON 失败: %w", err)
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("B 站返回业务错误 code=%d", resp.Code)
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.Data.List))
	for _, v := range resp.Data.List {
		// 视频链接：优先用短链，否则用 BV 号拼
		itemURL := v.ShortLink
		if itemURL == "" && v.Bvid != "" {
			itemURL = "https://www.bilibili.com/video/" + v.Bvid
		}

		// 关键词过滤
		if keyword != "" {
			combined := v.Title + " " + v.Owner.Name + " " + v.Tag + " " + v.Desc
			if !strings.Contains(strings.ToLower(combined), strings.ToLower(keyword)) {
				continue
			}
		}

		// 热度近似公式：播放+点赞*5+投币*10+收藏*3（B 站官方综合考虑）
		// 这里用简化版：view + like*3 + coin*5
		hot := v.Stat.View + v.Stat.Like*3 + v.Stat.Coin*5

		items = append(items, types.HotItem{
			Title:       v.Title,
			URL:         itemURL,
			Summary:     v.Desc,
			Source:      c.Source(),
			Hot:         hot,
			Author:      v.Owner.Name,
			PublishedAt: v.PubDate,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}