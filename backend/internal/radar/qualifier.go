package radar

import (
	"strings"
	"time"
	"unicode"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

const recencyWindow = 30 * 24 * time.Hour

type QualificationDecision struct {
	Eligible bool
	Reason   string
}

func Qualify(signal store.Signal, evidence store.EvidenceSnapshot, now time.Time) QualificationDecision {
	if !types.IsRadarSource(signal.Source) {
		return QualificationDecision{Reason: "unsupported_source"}
	}
	if signal.Source == types.SourceSkillsMP && evidence.EvidenceClass == "catalog_discovery" {
		return QualificationDecision{Reason: "github_verification_required"}
	}
	if timestamp := signalRecency(signal); timestamp.IsZero() || now.Sub(timestamp) > recencyWindow {
		return QualificationDecision{Reason: "outside_recency_window"}
	}
	text := strings.ToLower(signal.OriginalTitle + " " + evidence.Excerpt)
	if !containsTerm(text, "ai") && !containsAny(text, "agent", "skill", "mcp", "llm", "claude", "codex", "vibe coding", "人工智能", "智能体", "大模型", "提示词", "自动化") {
		return QualificationDecision{Reason: "outside_ai_tracks"}
	}
	if signal.Source == types.SourceGitHub || signal.Source == types.SourceSkillsMP {
		if evidence.EvidenceClass != "original_documentation" {
			return QualificationDecision{Reason: "original_documentation_required"}
		}
		if !containsAny(text, "install", "setup", "usage", "quickstart", "run ", "安装", "配置", "使用") {
			return QualificationDecision{Reason: "missing_usable_setup"}
		}
	}
	if signal.Source == types.SourceReddit && evidence.EvidenceClass != "community_discussion" {
		return QualificationDecision{Reason: "community_evidence_required"}
	}
	return QualificationDecision{Eligible: true, Reason: "eligible"}
}

func signalRecency(signal store.Signal) time.Time {
	if signal.SourceUpdatedAt != nil {
		return *signal.SourceUpdatedAt
	}
	if signal.SourcePublishedAt != nil {
		return *signal.SourcePublishedAt
	}
	return signal.CreatedAt
}

func containsTerm(text, term string) bool {
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		if token == term {
			return true
		}
	}
	return false
}

func containsAny(text string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(text, value) {
			return true
		}
	}
	return false
}
