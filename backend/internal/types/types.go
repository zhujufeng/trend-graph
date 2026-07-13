// Package types 定义全项目共用的数据结构。
//
// 为什么单独抽一个 types 包？
// - Go 没有"类"的概念，数据结构用 struct 表示。
// - 后端、爬虫、AI 分析、图谱构建都要用到"热点"这个概念。
// - 放一个公共包，避免循环依赖、统一字段。
package types

// HotItem 表示一条热点内容。
//
// 这是全项目最核心的数据结构，所有信息源抓到的内容
// 最终都会被转换成这个统一格式，方便存数据库、给前端、给 AI 分析。
//
// Go struct 的 json tag 用反引号括起来，告诉 encoding/json
// 在序列化成 JSON 时用什么字段名（驼峰风格给前端更友好）。
type HotItem struct {
	// 主键 ID（入库后由数据库自增）
	ID int64 `json:"id"`

	// 标题。例如 HN 上的帖子标题、微博热搜词条。
	Title string `json:"title"`

	// 原文链接，点开能跳到来源页面。
	URL string `json:"url"`

	// 摘要。可能是 AI 生成的，也可能是抓到的原文片段。
	// 阶段 1 先留空，阶段 3 接入 DeepSeek 后会填充。
	Summary string `json:"summary"`

	// 来源平台。用字符串简写以方便扩展，比如 "hn" "weibo" "bilibili"。
	Source string `json:"source"`

	// 热度分数。HN 是 score，微博是热搜排名，GitHub 是 star 数。
	// 统一存成 int，不同源的语义略有差异，前端显示时再加单位。
	Hot int `json:"hot"`

	// 作者/发布者。HN 是 by 字段，B 站是 up 主。
	Author string `json:"author"`

	// 原始发布时间戳（秒）。
	PublishedAt int64 `json:"publishedAt"`

	// 抓取时间戳（秒）。我们入库时间，方便后续按抓取时间排序。
	FetchedAt int64 `json:"fetchedAt"`
}

// Crawler 是信息源爬虫的统一接口。
//
// Go 的 interface 是隐式实现的：
// 只要某 struct 实现了 Fetch 方法，它就自动满足这个接口，无需显式声明。
// 这是 Go 与 Java/C# 的关键区别。
//
// 统一接口的好处：
// - 调用方不关心你抓的是 HN 还是微博，都按 Fetch 调
// - 阶段 5 加新源时，只要写一个新 struct 实现 Fetch 即可
// - 方便写并发抓取（所有爬虫扔进 channel 一起调度）
type Crawler interface {
	// Source 返回信息源标识符，比如 "hn" "weibo"，便于日志和过滤
	Source() string

	// Fetch 抓取该源的若干条热点，返回 HotItem 切片或错误
	// keyword 是监控关键词，阶段 3 之后会配合 AI 查询扩展
	// limit 是要抓的条数上限，让调用方控制流量
	Fetch(keyword string, limit int) ([]HotItem, error)
}