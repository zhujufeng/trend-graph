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
	Source        string
	OriginalTitle string
	OriginalURL   string
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

func (a *Analyzer) AnalyzeSignal(ctx context.Context, signal SignalInput, evidence EvidenceInput) (SignalAnalysisOutput, error) {
	systemPrompt := `你是个人 AI 信号雷达的证据分析器。只依据给定证据输出中文 JSON。
必须区分事实、解释和行动；保留 evidenceClass；每条事实必须带 sourceUrl。
只有 evidenceClass=user_verified 才能使用“我测试过”等第一人称验证表达。
action 必须是可以立即执行的具体步骤；不能落地时明确说明，不得编造。
GitHub 工具必须从证据中提取 toolType、compatibleClients 和 installation；不能把“通用”写成没有依据的一键安装。
只有重大模型/平台/核心工具发布、显著效率提升、经多方讨论印证的真实痛点、或有来源支撑的重大内容机会才能 alertEligible=true。
alertEligible=true 时 alertCategory 必须是 major_release、material_efficiency_gain、corroborated_pain_point、source_backed_content_opportunity 之一，并提供 alertReason。
字段：evidenceClass,facts[{claim,sourceUrl}],whatChanged,audience,practicalUse,prerequisites,toolType,compatibleClients,installation,painPoint,action,contentOpportunity,uncertainty,alertEligible,alertCategory,alertReason。`
	userPrompt := fmt.Sprintf("来源：%s\n原始标题：%s\n原始链接：%s\n证据链接：%s\n证据类别：%s\n证据正文：\n%s",
		signal.Source, signal.OriginalTitle, signal.OriginalURL, evidence.SourceURL, evidence.EvidenceClass, evidence.Excerpt)
	content, response, err := a.ai.Chat(ctx, ai.ChatRequest{
		Model:          a.model,
		Messages:       []ai.Message{{Role: "system", Content: systemPrompt}, {Role: "user", Content: userPrompt}},
		Temperature:    0.1,
		MaxTokens:      1200,
		ResponseFormat: &ai.ResponseFormat{Type: "json_object"},
	})
	if err != nil {
		return SignalAnalysisOutput{}, fmt.Errorf("AI signal analysis failed: %w", err)
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
	if signal.Source == "github" && (parsed.Installation == "" || len(parsed.CompatibleClients) == 0) {
		return SignalAnalysisOutput{}, errors.New("AI tool analysis is missing compatibility or installation evidence")
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

func ValidAlertCategory(category string) bool {
	switch category {
	case "major_release", "material_efficiency_gain", "corroborated_pain_point", "source_backed_content_opportunity":
		return true
	default:
		return false
	}
}
