package radar

import (
	"fmt"
	"testing"
	"time"

	"trend-graph/internal/store"
)

func TestRankSignalsUsesValueRecencyAlertAndDeduplicates(t *testing.T) {
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	items := []store.RadarSignal{
		rankedFixture(1, "Release A", "https://example.com/a", now.Add(-2*time.Hour), 4, false, "AI"),
		rankedFixture(2, "Release B", "https://example.com/b", now.Add(-10*24*time.Hour), 3, true, "开发工具"),
		rankedFixture(3, "Release-A!", "https://example.com/c", now.Add(-time.Hour), 5, true, "AI"),
	}
	ranked := RankSignals(items, now)
	if len(ranked) != 2 {
		t.Fatalf("ranked count = %d, want duplicate title removed", len(ranked))
	}
	if ranked[0].Item.Signal.ID != 3 || ranked[0].RankScore != 78 {
		t.Fatalf("first = %#v", ranked[0])
	}
	if ranked[1].RankScore != 51 {
		t.Fatalf("second score = %d", ranked[1].RankScore)
	}
}

func TestSelectDigestSignalsCapsEachTopicAndSkipsDelivered(t *testing.T) {
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	items := make([]store.RadarSignal, 0, 6)
	for index := 1; index <= 4; index++ {
		items = append(items, rankedFixture(int64(index), fmt.Sprintf("AI %d", index), fmt.Sprintf("https://example.com/ai/%d", index), now, 5, false, "AI"))
	}
	items = append(items, rankedFixture(5, "Robot", "https://example.com/robot", now, 4, false, "机器人"))
	delivered := rankedFixture(6, "Delivered", "https://example.com/delivered", now, 5, true, "机器人")
	deliveredAt := now.Add(-time.Hour)
	delivered.Signal.LastDeliveredAt = &deliveredAt
	items = append(items, delivered)

	selected := SelectDigestSignals(items, now, 8)
	if len(selected) != 3 {
		t.Fatalf("selected = %#v", selected)
	}
	if selected[0].Item.Signal.ID != 4 || selected[1].Item.Signal.ID != 3 || selected[2].Item.Signal.ID != 5 {
		t.Fatalf("selected IDs = %d, %d, %d", selected[0].Item.Signal.ID, selected[1].Item.Signal.ID, selected[2].Item.Signal.ID)
	}
}

func TestRankSignalsDefaultsOldAnalysisToValueThree(t *testing.T) {
	now := time.Date(2026, 7, 18, 8, 0, 0, 0, time.UTC)
	published := now.Add(-2 * time.Hour)
	items := []store.RadarSignal{{
		Signal:   store.Signal{ID: 1, OriginalTitle: "Legacy", OriginalURL: "https://example.com/legacy", SourcePublishedAt: &published},
		Analysis: &store.SignalAnalysis{AnalysisJSON: `{"whatChanged":"old shape"}`},
	}}
	if got := RankSignals(items, now); len(got) != 1 || got[0].RankScore != 38 {
		t.Fatalf("ranked = %#v", got)
	}
}

func rankedFixture(id int64, title, url string, published time.Time, value int, alert bool, topic string) store.RadarSignal {
	return store.RadarSignal{
		Signal:   store.Signal{ID: id, OriginalTitle: title, OriginalURL: url, CanonicalURL: url, Qualification: "qualified", LifecycleState: store.LifecycleInbox, SourcePublishedAt: &published},
		Analysis: &store.SignalAnalysis{AnalysisJSON: fmt.Sprintf(`{"matchedTopics":[%q],"valueScore":%d,"alertEligible":%t}`, topic, value, alert)},
	}
}
