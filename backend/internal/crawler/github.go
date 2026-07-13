// Package crawler - GitHub Trending 爬虫
//
// GitHub Trending 是 https://github.com/trending 页面，
// 展示最近热门的开源项目。没有官方 API，只能爬 HTML。
//
// 用 Go 标准库 + 正则解析：GitHub 的 HTML 结构稳定，
// 用 queries 太重，正则足够轻快。
//
// 实现统一 types.Crawler 接口：Source() + Fetch()
package crawler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"trend-graph/internal/types"
)

// GitHubCrawler 爬 github.com/trending
type GitHubCrawler struct{}

// NewGitHubCrawler 构造
func NewGitHubCrawler() *GitHubCrawler { return &GitHubCrawler{} }

// Source 实现 types.Crawler
func (c *GitHubCrawler) Source() string { return "github" }

// Fetch 抓 GitHub Trending 列表
//
// 流程：拉 HTML → 正则切出每条 article → 提取字段 → 转 HotItem
// 被 keyword 过滤（标题/描述包含关键词）
func (c *GitHubCrawler) Fetch(keyword string, limit int) ([]types.HotItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 限定抓取范围（只能按 since 切，不能按条数）
	// GitHub Trending URL: https://github.com/trending?since=daily
	url := "https://github.com/trending?since=daily"
	body, err := httpGet(context.Background(), url, nil)
	if err != nil {
		return nil, fmt.Errorf("拉取 GitHub Trending 失败: %w", err)
	}

	// GitHub 每个项目是一个 <article class="Box-row"> 块
	// 用正则分块后再逐块提取字段
	articleRe := regexp.MustCompile(`<article[^>]*class="[^"]*Box-row[^"]*"[^>]*>([\s\S]*?)</article>`)
	hrefRe := regexp.MustCompile(`<h2[^>]*>[\s\S]*?<a[^>]*href="([^"]+)"`)
	descRe := regexp.MustCompile(`<p[^>]*class="[^"]*col-9[^"]*"[^>]*>([\s\S]*?)</p>`)
	starRe := regexp.MustCompile(`<a[^>]*href="[^"]*/stargazers"[^>]*>\s*([\d,]+)\s*</a>`)
	todayRe := regexp.MustCompile(`([\d,]+)\s+stars today`)

	blocks := articleRe.FindAllStringSubmatch(body, -1)
	now := time.Now().Unix()
	items := make([]types.HotItem, 0, len(blocks))

	for _, b := range blocks {
		inner := b[1]

		// 提取 href，例如 "/ant-design/ant-design"
		href := ""
		if m := hrefRe.FindStringSubmatch(inner); m != nil {
			href = strings.TrimSpace(m[1])
		}
		if href == "" {
			continue
		}

		// 项目名 = href 去掉前导 /
		name := strings.TrimPrefix(href, "/")
		// 完整 URL
		itemURL := "https://github.com" + href

		// 描述
		desc := ""
		// HTML 中可能没有描述，要兜底
		if m := descRe.FindStringSubmatch(inner); m != nil {
			// 去标签
			desc = stripTags(strings.TrimSpace(m[1]))
		}

		// 总 star 数
		star := 0
		if m := starRe.FindStringSubmatch(inner); m != nil {
			star = atoi(m[1])
		}

		// 今日新增 star 数（当作 hot）
		hot := 0
		if m := todayRe.FindStringSubmatch(inner); m != nil {
			hot = atoi(m[1])
		}
		if hot == 0 {
			hot = star // 兜底
		}

		// 标题：项目名 + 描述前缀
		title := name
		if desc != "" {
			if len([]rune(desc)) > 80 {
				desc = string([]rune(desc)[:80]) + "..."
			}
			title = name + " - " + desc
		}

		// 关键词过滤（无关键词则全收）
		if keyword != "" {
			combined := name + " " + desc
			if !strings.Contains(strings.ToLower(combined), strings.ToLower(keyword)) {
				continue
			}
		}

		items = append(items, types.HotItem{
			Title:       title,
			URL:         itemURL,
			Summary:     desc,
			Source:      c.Source(),
			Hot:         hot,
			Author:      strings.SplitN(name, "/", 2)[0], // 仓库 owner
			PublishedAt: now, // GitHub Trending 没给出精确时间
			FetchedAt:   now,
		})

		if len(items) >= limit {
			break
		}
	}

	return items, nil
}

// ===== 公共辅助函数（GitHub/Bing 都用，放公共方法在 package 内可复用） =====

// httpGet 把 GET 请求 + headers + body 读取一行包，多个爬虫复用
func httpGet(ctx context.Context, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	// 默认 UA，避免被反爬过滤
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; trend-graph/0.1)")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// stripTags 去除所有 HTML 标签，只留纯文本
func stripTags(s string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(s, ""))
}

// atoi 把 "1,234" 这种带逗号的字符串转成 int
func atoi(s string) int {
	s = strings.ReplaceAll(s, ",", "")
	n, _ := strconv.Atoi(strings.TrimSpace(s))
	return n
}