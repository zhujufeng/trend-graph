// Package crawler - Linux.do (L 站) 爬虫
//
// Linux.do 是个基于 Discourse 的中文社区论坛，
// 主要讨论 Linux/技术/AI/网络等话题，干货多。
//
// Discourse 有公开的列表 JSON API：
//   GET https://linux.do/latest.json
//   返回 topic_list.topics，每个话题含 title/slug/posts_count/created_at 等
//
// 关键词过滤 title 只做包含匹配。
package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// LinuxDoCrawler L 站爬虫
type LinuxDoCrawler struct{}

// NewLinuxDoCrawler 构造
func NewLinuxDoCrawler() *LinuxDoCrawler { return &LinuxDoCrawler{} }

// Source 实现 types.Crawler
func (c *LinuxDoCrawler) Source() string { return "linuxdo" }

// linuxdoResp Discourse latest.json 响应
type linuxdoResp struct {
	TopicList struct {
		Topics []struct {
			ID         int    `json:"id"`
			Title     string `json:"title"`
			Slug     string `json:"slug"`     // URL 段截
			Posts    int    `json:"posts_count"`
			Views    int    `json:"views"`
			LikeCount int   `json:"like_count"`
			ReplyCount int  `json:"reply_count"`
			PostedAt  string `json:"created_at"` // ISO 时间字符串
			LastPostedAt string `json:"last_posted_at"`
		} `json:"topics"`
	} `json:"topic_list"`
}

// Fetch 实现 types.Crawler
func (c *LinuxDoCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// latest.json 是论坛最近话题列表，按回帖时间倒序
	url := fmt.Sprintf("https://linux.do/latest.json?no_definitions=true&page=0")
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/126.0.0.0 Safari/537.36",
		"Accept":          "application/json",
		"Accept-Language": "zh-CN,zh;q=0.9",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, fmt.Errorf("拉取 Linux.do 失败: %w", err)
	}

	var resp linuxdoResp
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		return nil, fmt.Errorf("解析 Linux.do JSON 失败: %w", err)
	}

	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(resp.TopicList.Topics))
	for _, tp := range resp.TopicList.Topics {
		// 跳过置顶（pinned=true 的话题），这里简化处理
		// 用 slug+ID 拼 URL 是 Discourse 习惯
		itemURL := fmt.Sprintf("https://linux.do/t/%s/%d", tp.Slug, tp.ID)

		// 关键词过滤
		if keyword != "" {
			if !strings.Contains(strings.ToLower(tp.Title), strings.ToLower(keyword)) {
				continue
			}
		}

		// 热度：浏览 + 点赞*5 + 回复*3
		hot := tp.Views + tp.LikeCount*5 + tp.ReplyCount*3

		// postedAt 是 ISO 时间字符串，用 time.Parse 解析成秒
		publishedAt := now
		if t, err := time.Parse(time.RFC3339, tp.PostedAt); err == nil {
			publishedAt = t.Unix()
		}

		items = append(items, types.HotItem{
			Title:       tp.Title,
			URL:         itemURL,
			Summary:     "",
			Source:      c.Source(),
			Hot:         hot,
			Author:      "",
			PublishedAt: publishedAt,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}

	return items, nil
}