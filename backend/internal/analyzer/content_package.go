package analyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"trend-graph/internal/ai"
)

type ContentPackageDraft struct {
	Strategy    ContentStrategy `json:"strategy"`
	Xiaohongshu PlatformDraft   `json:"xiaohongshu"`
	Wechat      PlatformDraft   `json:"wechat"`
	X           XDraft          `json:"x"`
	VisualPlan  []VisualAsset   `json:"visualPlan"`
}

type ContentStrategy struct {
	Angle        string `json:"angle"`
	Audience     string `json:"audience"`
	EvidenceNote string `json:"evidenceNote"`
}

type PlatformDraft struct {
	Title       string   `json:"title"`
	Body        string   `json:"body"`
	Tags        []string `json:"tags"`
	SourceLinks []string `json:"sourceLinks"`
}

type XDraft struct {
	Chinese     string   `json:"chinese"`
	English     string   `json:"english"`
	SourceLinks []string `json:"sourceLinks"`
}

type VisualAsset struct {
	Purpose     string `json:"purpose"`
	AspectRatio string `json:"aspectRatio"`
	Prompt      string `json:"prompt"`
}

func (a *Analyzer) GenerateContentPackage(ctx context.Context, signal SignalInput, evidence EvidenceInput, analysisJSON json.RawMessage) (ContentPackageDraft, error) {
	systemPrompt := `你是证据优先的中文 AI 内容编辑。只输出 JSON，生成小红书、微信公众号和 X 三个平台的可编辑草稿。
不得把第三方实践写成本人实测；只有 evidenceClass=user_verified 才能使用“我测试过、亲测、实测”等第一人称验证表达。
X 必须提供中文稿和适配英文受众的英文稿。每个平台都保留来源链接。
visualPlan 每项必须含 purpose、aspectRatio 和可复现的中文图片生成 prompt，不生成图片。
JSON 字段：strategy{angle,audience,evidenceNote},xiaohongshu{title,body,tags},wechat{title,body,tags},x{chinese,english},visualPlan[{purpose,aspectRatio,prompt}]。`
	userPrompt := fmt.Sprintf("标题：%s\n原始链接：%s\n证据链接：%s\n证据等级：%s\n证据：\n%s\n已审核分析：\n%s",
		signal.OriginalTitle, signal.OriginalURL, evidence.SourceURL, evidence.EvidenceClass, evidence.Excerpt, analysisJSON)
	content, _, err := a.ai.Chat(ctx, ai.ChatRequest{
		Model: a.model,
		Messages: []ai.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature:    0.3,
		MaxTokens:      3600,
		ResponseFormat: &ai.ResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return ContentPackageDraft{}, fmt.Errorf("AI content package generation failed: %w", err)
	}
	var draft ContentPackageDraft
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &draft); err != nil {
		return ContentPackageDraft{}, fmt.Errorf("AI content package returned invalid JSON: %w", err)
	}
	if draft.Strategy.Angle == "" || draft.Xiaohongshu.Body == "" || draft.Wechat.Body == "" || draft.X.Chinese == "" || draft.X.English == "" {
		return ContentPackageDraft{}, errors.New("AI content package is missing required drafts")
	}
	if len(draft.VisualPlan) == 0 {
		return ContentPackageDraft{}, errors.New("AI content package is missing a visual plan")
	}
	for _, asset := range draft.VisualPlan {
		if asset.Purpose == "" || asset.AspectRatio == "" || asset.Prompt == "" {
			return ContentPackageDraft{}, errors.New("AI content package has an incomplete visual asset")
		}
	}
	allText := draft.Strategy.Angle + draft.Xiaohongshu.Title + draft.Xiaohongshu.Body + draft.Wechat.Title + draft.Wechat.Body + draft.X.Chinese + draft.X.English
	if evidence.EvidenceClass != "user_verified" && ContainsFirstPersonVerification(allText) {
		return ContentPackageDraft{}, errors.New("AI content package invented first-person verification")
	}
	links := []string{signal.OriginalURL}
	if evidence.SourceURL != "" && evidence.SourceURL != signal.OriginalURL {
		links = append(links, evidence.SourceURL)
	}
	draft.Xiaohongshu.SourceLinks = links
	draft.Wechat.SourceLinks = links
	draft.X.SourceLinks = links
	return draft, nil
}

func ContainsFirstPersonVerification(value string) bool {
	for _, phrase := range []string{"我测试过", "我实测", "亲测", "本人测试"} {
		if strings.Contains(value, phrase) {
			return true
		}
	}
	return false
}
