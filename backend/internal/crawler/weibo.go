// Package crawler - 微博热搜爬虫
//
// 微博热搜公开接口（多个备选）：
//   1) https://weibo.com/ajax/side/hotSearch     – 微博新版 JSON 接口
//   2) https://m.weibo.cn/api/container/getIndex?containerid=106003type%3D25%26t%3D3%26..._hz&oCatId=100
//   3) https://s.weibo.com/top/summary           – HTML 页面（最稳）
//
// 这里先用接口 1，不行就降级到 HTML 抓取。
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// WeiboCrawler 微博热搜爬虫
type WeiboCrawler struct{}

// NewWeiboCrawler 构造
func NewWeiboCrawler() *WeiboCrawler { return &WeiboCrawler{} }

// Source 实现 types.Crawler
func (c *WeiboCrawler) Source() string { return "weibo" }

// weiboResp 微博热搜接口响应
//
// data.realtime 是热搜词数组，每含结构:
//   word（热搜词）/ num（热度）/ label_num（标号：热/新/沸）
type weiboResp struct {
	OK   int `json:"ok"`
	Data struct {
		Realtime []struct {
			Word         string `json:"word"`
			Num         int    `json:"num"`
			Rank         int    `json:"rank"`
			LabelNum     string `json:"label_num"`  // "热"/"新"/"沸", 空=普通
			Note         string `json:"note"`        // 完整描述
			Category     string `json:"category"`    // 分类如"时政"
			HotSearchUrl string `json:"url"`          // 搜索结果相对路径
		} `json:"realtime"`
	} `json:"data"`
}

// Fetch 实现 types.Crawler
func (c *WeiboCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	url := "https://weibo.com/ajax/side/hotSearch"
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36",
		"Referer":         "https://weibo.com/hot/search",
		"Accept":          "application/json",
		"Accept-Language": "zh-CN,zh;q=0.9",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, fmt.Errorf("拉取微博热搜失败: %w", err)
	}

	var resp weiboResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析微博 JSON 失败: %w", err)
	}
	if resp.OK != 1 {
		return nil, fmt.Errorf("微博业务返回 ok=%d", resp.OK)
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.Data.Realtime))
	for i, r := range resp.Data.Realtime {
		// 微博热搜 URL：拼绝对路径
		itemURL := "https://s.weibo.com/weibo?q=" + urlEncode("#"+r.Word+"#")

		// 关键词过滤
		if keyword != "" {
			combined := r.Word + " " + r.Note + " " + r.Category
			if !strings.Contains(strings.ToLower(combined), strings.ToLower(keyword)) {
				continue
			}
		}

		// 标签转摘要："热/新/沸"
		tag := strings.TrimSpace(r.LabelNum)
		summary := r.Note
		if tag != "" {
			summary = fmt.Sprintf("[%s] %s", tag, r.Note)
		}

		// 用排名估算热度（第1名 5000，每降一名减 100）
		hot := r.Num
		if hot == 0 {
			hot = 5000 - i*100
			if hot < 0 {
				hot = 0
			}
		}

		items = append(items, types.HotItem{
			Title:       r.Word,
			URL:         itemURL,
			Summary:     summary,
			Source:      c.Source(),
			Hot:         hot,
			Author:      "微博热搜榜",
			PublishedAt: now,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}