// Package crawler - Bing 搜索爬虫
//
// 用搜索结果做"通用搜索"信息源。
// 没有官方 API，只能爬 HTML。
// 反爬策略：
//   - 模拟浏览器头
//   - 失败（遇到 Captcha 等）自动降级到 DuckDuckGo Lite HTML（更稳）
package crawler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// BingCrawler 爬搜索结果
type BingCrawler struct{}

// NewBingCrawler 构造
func NewBingCrawler() *BingCrawler { return &BingCrawler{} }

// Source 实现 types.Crawler
func (c *BingCrawler) Source() string { return "bing" }

// Fetch 抓搜索结果前 limit 条
//
// 必须传 keyword，没传就用 "AI" 兜底（搜索源本来就要关键词）
// 当 Bing 抓取失败或返回 0 条时，降级用 DuckDuckGo Lite
func (c *BingCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 30 {
		limit = 10
	}
	if keyword == "" {
		keyword = "AI"
	}

	// 先试 Bing
	items, err := c.fetchBing(keyword, limit)
	if err == nil && len(items) > 0 {
		return items, nil
	}

	// Bing 失败/空，降级到 DuckDuckGo（Lite HTML 接口稳定）
	items2, err2 := c.fetchDuckDuckGo(keyword, limit)
	if err2 == nil && len(items2) > 0 {
		return items2, nil
	}

	// 两个都失败，返回原始错误（Bing 更主流）
	if err != nil {
		return nil, err
	}
	return items, nil // 0 条但err=nil
}

// fetchBing 抓 www.bing.com/search?q=xxx
func (c *BingCrawler) fetchBing(keyword string, limit int) ([]types.HotItem, error) {
	url := fmt.Sprintf("https://www.bing.com/search?q=%s&count=%d", urlEncode(keyword), limit)
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
		"Accept-Language": "zh-CN,zh;q=0.9,en;q=0.8",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, err
	}

	// 检测是否被反爬（Captcha 页面）
	lower := strings.ToLower(body)
	if strings.Contains(lower, "captcha") || strings.Contains(lower, "verify") {
		return nil, fmt.Errorf("Bing 触发 Captcha")
	}

	return parseSearchHTML(body, "bing", limit), nil
}

// fetchDuckDuckGo 抓 lite.duckduckgo.com lite HTML 接口
//
// DuckDuckGo Lite 是个超干净的 HTML 表单搜索结果：
//   GET https://lite.duckduckgo.com/lite?q=xxx
// 每条结果是一个 <a rel="nofollow" href="...">...</a>
func (c *BingCrawler) fetchDuckDuckGo(keyword string, limit int) ([]types.HotItem, error) {
	url := "https://lite.duckduckgo.com/lite?q=" + urlEncode(keyword)
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	}

	body, err := httpGet(context.Background(), url, headers)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	// DDG lite 的结果链接都带 rel="nofollow" class="result-link"
	linkRe := regexp.MustCompile(`<a[^>]*rel="nofollow"[^>]*class="result-link"[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	// 摘要在结果 link 同级的下一个 <td> 里
	snippetRe := regexp.MustCompile(`<td[^>]*class="result-snippet"[^>]*>([\s\S]*?)</td>`)

	links := linkRe.FindAllStringSubmatch(body, -1)
	snippets := snippetRe.FindAllStringSubmatch(body, -1)

	items := make([]types.HotItem, 0, len(links))
	for i, m := range links {
		link := strings.TrimSpace(m[1])
		title := strings.TrimSpace(stripTags(m[2]))
		if link == "" || title == "" {
			continue
		}
		summary := ""
		if i < len(snippets) && snippets[i] != nil {
			summary = strings.TrimSpace(stripTags(snippets[i][1]))
		}
		items = append(items, types.HotItem{
			Title:       title,
			URL:         link,
			Summary:     summary,
			Source:      c.Source(), // 仍标 bing
			Hot:         0,
			PublishedAt: now,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}
	return items, nil
}

// parseSearchHTML 解析 Bing HTML 抽取结果
// 抽成函数让 bing 测试和 ddg 都能复用思路
func parseSearchHTML(body, source string, limit int) []types.HotItem {
	// Bing 每条结果： <li class="b_algo"><h2><a href="...">标题</a></h2><p>摘要</p></li>
	liRe := regexp.MustCompile(`<li[^>]*class="b_algo"[^>]*>([\s\S]*?)</li>`)
	hrefRe := regexp.MustCompile(`<h2>[\s\S]*?<a[^>]*href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	summaryRe := regexp.MustCompile(`<p[^>]*>([\s\S]*?)</p>`)

	blocks := liRe.FindAllStringSubmatch(body, -1)
	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(blocks))

	for _, b := range blocks {
		inner := b[1]
		hrefMatch := hrefRe.FindStringSubmatch(inner)
		if hrefMatch == nil {
			continue
		}
		link := strings.TrimSpace(hrefMatch[1])
		title := strings.TrimSpace(stripTags(hrefMatch[2]))
		if title == "" || link == "" {
			continue
		}
		summary := ""
		if m := summaryRe.FindStringSubmatch(inner); m != nil {
			summary = strings.TrimSpace(stripTags(m[1]))
		}
		items = append(items, types.HotItem{
			Title:       title,
			URL:         link,
			Summary:     summary,
			Source:      source,
			Hot:         0,
			PublishedAt: now,
			FetchedAt:   now,
		})
		if len(items) >= limit {
			break
		}
	}
	return items
}