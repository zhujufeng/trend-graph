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
	Eligible      bool
	Reason        string
	MatchedTopics []string
}

func Qualify(signal store.Signal, evidence store.EvidenceSnapshot, topics []string, now time.Time) QualificationDecision {
	if !types.IsRadarSource(signal.Source) {
		return QualificationDecision{Reason: "unsupported_source"}
	}
	if timestamp := signalRecency(signal); timestamp.IsZero() || now.Sub(timestamp) > recencyWindow {
		return QualificationDecision{Reason: "outside_recency_window"}
	}
	text := strings.ToLower(signal.OriginalTitle + " " + evidence.Excerpt)
	matched := matchTopics(text, topics)
	if len(matched) == 0 && !isWatchedGitHubRelease(signal) {
		return QualificationDecision{Reason: "outside_active_topics"}
	}
	if signal.Source == types.SourceGitHub {
		if evidence.EvidenceClass != "original_documentation" {
			return QualificationDecision{Reason: "original_documentation_required"}
		}
	}
	if signal.Source == types.SourceReddit && evidence.EvidenceClass != "community_discussion" {
		return QualificationDecision{Reason: "community_evidence_required"}
	}
	if signal.Source == types.SourceBluesky && evidence.EvidenceClass != "community_discussion" {
		return QualificationDecision{Reason: "community_evidence_required"}
	}
	if signal.Source == types.SourceRSS && evidence.EvidenceClass != "publisher_feed" {
		return QualificationDecision{Reason: "publisher_evidence_required"}
	}
	return QualificationDecision{Eligible: true, Reason: "eligible", MatchedTopics: matched}
}

func matchTopics(text string, topics []string) []string {
	matched := make([]string, 0, len(topics))
	for _, topic := range topics {
		trimmed := strings.TrimSpace(topic)
		if trimmed == "" {
			continue
		}
		lower := strings.ToLower(trimmed)
		isMatch := strings.Contains(text, lower)
		if strings.EqualFold(trimmed, "AI") {
			isMatch = containsTerm(text, "ai") || containsAny(text, "agent", "skill", "mcp", "llm", "claude", "codex", "vibe coding", "人工智能", "智能体", "大模型", "提示词", "自动化")
		}
		if isMatch {
			matched = append(matched, trimmed)
		}
	}
	return matched
}

func isWatchedGitHubRelease(signal store.Signal) bool {
	return signal.Source == types.SourceGitHub && strings.Contains(strings.ToLower(signal.OriginalURL), "/releases/")
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
