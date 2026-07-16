package radar

import (
	"encoding/json"
	"fmt"
	"time"

	"trend-graph/internal/store"
)

type Digest struct {
	Title                string
	Signals              []DigestSignal
	ContentOpportunities []DigestOpportunity
}

type DigestSignal struct {
	Title          string
	WhatChanged    string
	Interpretation string
	Action         string
	EvidenceClass  string
	LinkURL        string
}

type DigestOpportunity struct {
	Title string
	Angle string
}

type digestAnalysis struct {
	WhatChanged        string `json:"whatChanged"`
	PracticalUse       string `json:"practicalUse"`
	Action             string `json:"action"`
	ContentOpportunity string `json:"contentOpportunity"`
}

func BuildDigest(items []store.RadarSignal, now time.Time) (Digest, error) {
	edition := "晚报"
	if now.Hour() < 12 {
		edition = "早报"
	}
	digest := Digest{Title: fmt.Sprintf("AI 信号雷达 · %s %s", edition, now.Format("2006-01-02"))}
	for _, item := range items {
		if len(digest.Signals) >= 8 {
			break
		}
		if item.Signal.Qualification != "qualified" || item.Analysis == nil {
			continue
		}
		var analysis digestAnalysis
		if err := json.Unmarshal([]byte(item.Analysis.AnalysisJSON), &analysis); err != nil {
			return Digest{}, err
		}
		evidenceClass := ""
		if item.Evidence != nil {
			evidenceClass = item.Evidence.EvidenceClass
		}
		digest.Signals = append(digest.Signals, DigestSignal{
			Title: item.Signal.OriginalTitle, WhatChanged: analysis.WhatChanged,
			Interpretation: analysis.PracticalUse, Action: analysis.Action,
			EvidenceClass: evidenceClass, LinkURL: item.Signal.OriginalURL,
		})
		if analysis.ContentOpportunity != "" && len(digest.ContentOpportunities) < 3 {
			digest.ContentOpportunities = append(digest.ContentOpportunities, DigestOpportunity{
				Title: item.Signal.OriginalTitle, Angle: analysis.ContentOpportunity,
			})
		}
	}
	return digest, nil
}
