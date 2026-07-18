package radar

import (
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode"

	"trend-graph/internal/store"
)

type RankedSignal struct {
	Item          store.RadarSignal
	RankScore     int
	MatchedTopics []string
}

type rankingAnalysis struct {
	MatchedTopics []string `json:"matchedTopics"`
	ValueScore    int      `json:"valueScore"`
	AlertEligible bool     `json:"alertEligible"`
}

func RankSignals(items []store.RadarSignal, now time.Time) []RankedSignal {
	ranked := make([]RankedSignal, 0, len(items))
	for _, item := range items {
		if item.Signal.LifecycleState == store.LifecycleDismissed {
			continue
		}
		analysis := rankingAnalysis{ValueScore: 3}
		if item.Analysis != nil {
			var decoded rankingAnalysis
			if json.Unmarshal([]byte(item.Analysis.AnalysisJSON), &decoded) == nil {
				analysis = decoded
				if analysis.ValueScore < 1 || analysis.ValueScore > 5 {
					analysis.ValueScore = 3
				}
			}
		}
		score := analysis.ValueScore*10 + recencyBonus(signalRecency(item.Signal), now)
		if analysis.AlertEligible {
			score += 20
		}
		ranked = append(ranked, RankedSignal{Item: item, RankScore: score, MatchedTopics: analysis.MatchedTopics})
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].RankScore != ranked[j].RankScore {
			return ranked[i].RankScore > ranked[j].RankScore
		}
		left, right := signalRecency(ranked[i].Item.Signal), signalRecency(ranked[j].Item.Signal)
		if !left.Equal(right) {
			return left.After(right)
		}
		return ranked[i].Item.Signal.ID > ranked[j].Item.Signal.ID
	})

	deduplicated := make([]RankedSignal, 0, len(ranked))
	seenURLs := make(map[string]struct{}, len(ranked))
	seenTitles := make(map[string]struct{}, len(ranked))
	for _, item := range ranked {
		urlKey := strings.ToLower(strings.TrimSpace(item.Item.Signal.CanonicalURL))
		if urlKey == "" {
			urlKey = strings.ToLower(strings.TrimSpace(item.Item.Signal.OriginalURL))
		}
		titleKey := normalizedTitle(item.Item.Signal.OriginalTitle)
		if duplicateKey(seenURLs, urlKey) || duplicateKey(seenTitles, titleKey) {
			continue
		}
		if urlKey != "" {
			seenURLs[urlKey] = struct{}{}
		}
		if titleKey != "" {
			seenTitles[titleKey] = struct{}{}
		}
		deduplicated = append(deduplicated, item)
	}
	return deduplicated
}

func SelectDigestSignals(items []store.RadarSignal, now time.Time, limit int) []RankedSignal {
	if limit <= 0 {
		return nil
	}
	selected := make([]RankedSignal, 0, limit)
	perTopic := make(map[string]int)
	for _, item := range RankSignals(items, now) {
		state := item.Item.Signal.LifecycleState
		if item.Item.Signal.Qualification != "qualified" || item.Item.Analysis == nil || item.Item.Signal.LastDeliveredAt != nil || (state != "" && state != store.LifecycleInbox && state != store.LifecycleSaved) {
			continue
		}
		topic := "__unmatched__"
		if len(item.MatchedTopics) > 0 {
			topic = strings.ToLower(strings.TrimSpace(item.MatchedTopics[0]))
		}
		if perTopic[topic] >= 2 {
			continue
		}
		selected = append(selected, item)
		perTopic[topic]++
		if len(selected) == limit {
			break
		}
	}
	return selected
}

func recencyBonus(timestamp, now time.Time) int {
	if timestamp.IsZero() {
		return 0
	}
	age := now.Sub(timestamp)
	if age < 0 {
		age = 0
	}
	switch {
	case age <= 24*time.Hour:
		return 8
	case age <= 7*24*time.Hour:
		return 5
	case age <= 30*24*time.Hour:
		return 1
	default:
		return 0
	}
}

func duplicateKey(seen map[string]struct{}, key string) bool {
	if key == "" {
		return false
	}
	_, ok := seen[key]
	return ok
}

func normalizedTitle(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}), " ")
}
