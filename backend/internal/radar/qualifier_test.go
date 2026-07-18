package radar

import (
	"testing"
	"time"

	"trend-graph/internal/store"
)

func TestQualifyRequiresVerifiedUsableEvidenceBeforeModelWork(t *testing.T) {
	now := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	recent := now.Add(-24 * time.Hour)

	tests := []struct {
		name     string
		signal   store.Signal
		evidence store.EvidenceSnapshot
		topics   []string
		eligible bool
		reason   string
	}{
		{
			name:     "documented DEV practice is eligible",
			signal:   store.Signal{Source: "dev", OriginalTitle: "I built an MCP workflow", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "documented_third_party_practice", Excerpt: "The article includes steps, observed results, and failures."},
			topics:   []string{"AI"},
			eligible: true,
			reason:   "eligible",
		},
		{
			name:     "bluesky requires discussion evidence",
			signal:   store.Signal{Source: "bluesky", OriginalTitle: "MCP workflow", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "catalog_discovery", Excerpt: "A listing."},
			topics:   []string{"AI"},
			eligible: false,
			reason:   "community_evidence_required",
		},
		{
			name:     "old project is not sent to the model",
			signal:   store.Signal{Source: "github", OriginalTitle: "AI agent", SourceUpdatedAt: timePointer(now.Add(-31 * 24 * time.Hour))},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Install and use this agent."},
			topics:   []string{"AI"},
			eligible: false,
			reason:   "outside_recency_window",
		},
		{
			name:     "ai is matched as a term rather than inside maintainer",
			signal:   store.Signal{Source: "github", OriginalTitle: "Maintainer helper", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Install and use this repository helper."},
			topics:   []string{"AI"},
			eligible: false,
			reason:   "outside_active_topics",
		},
		{
			name:     "custom Chinese topic matches publisher feed",
			signal:   store.Signal{Source: "rss", OriginalTitle: "机器人产业周报", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "publisher_feed", Excerpt: "本周人形机器人出货量上升。"},
			topics:   []string{"机器人"},
			eligible: true,
			reason:   "eligible",
		},
		{
			name:     "watched GitHub release works without active topics",
			signal:   store.Signal{Source: "github", OriginalTitle: "v2.0", OriginalURL: "https://github.com/acme/tool/releases/tag/v2.0", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Version 2.0 is available."},
			eligible: true,
			reason:   "eligible",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := Qualify(test.signal, test.evidence, test.topics, now)
			if decision.Eligible != test.eligible || decision.Reason != test.reason {
				t.Fatalf("decision = %#v", decision)
			}
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }
