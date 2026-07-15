package radar

import (
	"fmt"
	"testing"
	"time"

	"trend-graph/internal/store"
)

func TestBuildDigestCapsSignalsAndContentOpportunities(t *testing.T) {
	items := make([]store.RadarSignal, 0, 10)
	for index := 0; index < 10; index++ {
		items = append(items, store.RadarSignal{
			Signal: store.Signal{
				ID: int64(index + 1), Source: "github", OriginalTitle: fmt.Sprintf("AI Tool %d", index+1),
				OriginalURL: fmt.Sprintf("https://github.com/owner/tool-%d", index+1), Qualification: "qualified",
			},
			Analysis: &store.SignalAnalysis{AnalysisJSON: fmt.Sprintf(`{"whatChanged":"更新 %d","action":"本地试用 %d","contentOpportunity":"选题 %d"}`, index+1, index+1, index+1)},
		})
	}

	digest, err := BuildDigest(items, time.Date(2026, 7, 15, 8, 0, 0, 0, time.FixedZone("CST", 8*60*60)))
	if err != nil {
		t.Fatal(err)
	}
	if digest.Title != "AI 信号雷达 · 早报 2026-07-15" {
		t.Fatalf("title = %q", digest.Title)
	}
	if len(digest.Signals) != 8 || len(digest.ContentOpportunities) != 3 {
		t.Fatalf("signals = %d, opportunities = %d", len(digest.Signals), len(digest.ContentOpportunities))
	}
	if digest.Signals[0].LinkURL != "https://github.com/owner/tool-1" || digest.Signals[0].Action != "本地试用 1" {
		t.Fatalf("first signal = %#v", digest.Signals[0])
	}
}
