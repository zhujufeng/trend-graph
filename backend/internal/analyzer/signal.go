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
	PainPoint          string         `json:"painPoint"`
	Action             string         `json:"action"`
	ContentOpportunity string         `json:"contentOpportunity"`
	Uncertainty        string         `json:"uncertainty"`
	AlertEligible      bool           `json:"alertEligible"`
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
字段：evidenceClass,facts[{claim,sourceUrl}],whatChanged,audience,practicalUse,prerequisites,painPoint,action,contentOpportunity,uncertainty,alertEligible,alertReason。`
	userPrompt := fmt.Sprintf("原始标题：%s\n原始链接：%s\n证据链接：%s\n证据类别：%s\n证据正文：\n%s",
		signal.OriginalTitle, signal.OriginalURL, evidence.SourceURL, evidence.EvidenceClass, evidence.Excerpt)
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
	if parsed.EvidenceClass != evidence.EvidenceClass {
		return SignalAnalysisOutput{}, errors.New("AI signal analysis changed the evidence class")
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
