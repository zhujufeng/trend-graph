package analyzer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"trend-graph/internal/ai"
)

// SignalInput is the minimum source context needed for evidence-based analysis.
// Keeping this DTO here avoids coupling model prompts to persistence models.
type SignalInput struct {
	Source         string
	OriginalTitle  string
	OriginalURL    string
	Topics         []string
	AllowUnmatched bool
}

type EvidenceInput struct {
	SourceURL     string
	EvidenceClass string
	Excerpt       string
}

type SignalAnalysisOutput struct {
	JSON         json.RawMessage
	InputTokens  int
	OutputTokens int
}

type signalAnalysis struct {
	MatchedTopics      []string       `json:"matchedTopics"`
	ValueScore         int            `json:"valueScore"`
	EvidenceClass      string         `json:"evidenceClass"`
	Facts              []analysisFact `json:"facts"`
	WhatChanged        string         `json:"whatChanged"`
	Audience           string         `json:"audience"`
	PracticalUse       string         `json:"practicalUse"`
	Prerequisites      string         `json:"prerequisites"`
	ToolType           string         `json:"toolType"`
	CompatibleClients  []string       `json:"compatibleClients"`
	Installation       string         `json:"installation"`
	PainPoint          string         `json:"painPoint"`
	Action             string         `json:"action"`
	ContentOpportunity string         `json:"contentOpportunity"`
	Uncertainty        string         `json:"uncertainty"`
	AlertEligible      bool           `json:"alertEligible"`
	AlertCategory      string         `json:"alertCategory"`
	AlertReason        string         `json:"alertReason"`
}

type analysisFact struct {
	Claim     string `json:"claim"`
	SourceURL string `json:"sourceUrl"`
}

const maxEvidenceRunes = 12_000

func (a *Analyzer) AnalyzeSignal(ctx context.Context, signal SignalInput, evidence EvidenceInput) (SignalAnalysisOutput, error) {
	systemPrompt := `你是个人信息雷达的证据分析器。只依据给定证据输出中文 JSON。
必须区分事实、解释和行动；保留 evidenceClass；每条事实必须带 sourceUrl。
matchedTopics 只能从用户关注主题中选择，最多 3 个；明确订阅但不匹配主题时可以为空。valueScore 必须是 1 到 5 的整数，表示这条信息对用户的实际价值。
只有 evidenceClass=user_verified 才能使用“我测试过”等第一人称验证表达。
action 必须是可以立即执行的具体步骤；不能落地时明确说明，不得编造。
只有证据明确表明它是工具时，才填写 toolType、compatibleClients 和 installation；不能把“通用”写成没有依据的一键安装。
只有重大模型/平台/核心工具发布、显著效率提升、经多方讨论印证的真实痛点、或有来源支撑的重大内容机会才能 alertEligible=true。
alertEligible=true 时 alertCategory 必须是 major_release、material_efficiency_gain、corroborated_pain_point、source_backed_content_opportunity 之一，并提供 alertReason。
保持简洁，facts 最多 3 条，每个文本字段只写结论，不复述整段证据。
字段：matchedTopics,valueScore,evidenceClass,facts[{claim,sourceUrl}],whatChanged,audience,practicalUse,prerequisites,toolType,compatibleClients,installation,painPoint,action,contentOpportunity,uncertainty,alertEligible,alertCategory,alertReason。`
	userPrompt := fmt.Sprintf("用户关注主题：%s\n明确订阅（允许主题为空）：%t\n来源：%s\n原始标题：%s\n原始链接：%s\n证据链接：%s\n证据类别：%s\n证据正文：\n%s",
		strings.Join(signal.Topics, "、"), signal.AllowUnmatched, signal.Source, signal.OriginalTitle, signal.OriginalURL, evidence.SourceURL, evidence.EvidenceClass, boundedEvidence(evidence.Excerpt))
	content, response, err := a.ai.Chat(ctx, ai.ChatRequest{
		Model:          a.model,
		Messages:       []ai.Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
		Temperature:    0.1,
		MaxTokens:      2400,
		ResponseFormat: &ai.ResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return SignalAnalysisOutput{}, fmt.Errorf("AI signal analysis failed: %w", err)
	}
	if response != nil && len(response.Choices) > 0 && response.Choices[0].FinishReason == "length" {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis output was truncated")
	}
	var parsed signalAnalysis
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &parsed); err != nil {
		return SignalAnalysisOutput{}, fmt.Errorf("AI signal analysis returned invalid JSON: %w", err)
	}
	if parsed.WhatChanged == "" || parsed.Audience == "" || parsed.PracticalUse == "" || parsed.Action == "" || parsed.EvidenceClass == "" {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis is missing required fields")
	}
	if len(parsed.Facts) == 0 {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis is missing sourced facts")
	}
	for _, fact := range parsed.Facts {
		if fact.Claim == "" || (fact.SourceURL != evidence.SourceURL && fact.SourceURL != signal.OriginalURL) {
			return SignalAnalysisOutput{}, errors.New("AI signal analysis has an untraceable fact")
		}
	}
	if parsed.EvidenceClass != evidence.EvidenceClass {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis changed the evidence class")
	}
	if parsed.ValueScore < 1 || parsed.ValueScore > 5 {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis has an invalid value score")
	}
	if err := validateMatchedTopics(parsed.MatchedTopics, signal.Topics, signal.AllowUnmatched); err != nil {
		return SignalAnalysisOutput{}, err
	}
	if parsed.AlertEligible && (!ValidAlertCategory(parsed.AlertCategory) || parsed.AlertReason == "") {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis has an invalid alert decision")
	}
	normalized, err := json.Marshal(parsed)
	if err != nil {
		return SignalAnalysisOutput{}, err
	}
	output := SignalAnalysisOutput{JSON: normalized}
	if response != nil {
		output.InputTokens = response.Usage.PromptTokens
		output.OutputTokens = response.Usage.CompletionTokens
	}
	return output, nil
}

func validateMatchedTopics(matched, available []string, allowUnmatched bool) error {
	if len(matched) > 3 || (len(available) > 0 && len(matched) == 0 && !allowUnmatched) {
		return errors.New("AI signal analysis has invalid matched topics")
	}
	allowed := make(map[string]struct{}, len(available))
	for _, topic := range available {
		allowed[strings.ToLower(strings.TrimSpace(topic))] = struct{}{}
	}
	seen := make(map[string]struct{}, len(matched))
	for _, topic := range matched {
		key := strings.ToLower(strings.TrimSpace(topic))
		if _, ok := allowed[key]; !ok {
			return errors.New("AI signal analysis selected an unknown topic")
		}
		if _, ok := seen[key]; ok {
			return errors.New("AI signal analysis repeated a topic")
		}
		seen[key] = struct{}{}
	}
	return nil
}

func boundedEvidence(value string) string {
	runes := []rune(value)
	if len(runes) <= maxEvidenceRunes {
		return value
	}
	const tailRunes = 4_000
	return string(runes[:maxEvidenceRunes-tailRunes]) + "\n\n[...证据正文已截断...]\n\n" + string(runes[len(runes)-tailRunes:])
}

func ValidAlertCategory(category string) bool {
	switch category {
	case "major_release", "material_efficiency_gain", "corroborated_pain_point", "source_backed_content_opportunity":
		return true
	default:
		return false
	}
}
