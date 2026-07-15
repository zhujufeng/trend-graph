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
		eligible bool
		reason   string
	}{
		{
			name:     "verified skill documentation is eligible",
			signal:   store.Signal{Source: "skillsmp", OriginalTitle: "MCP Inspector Skill", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Install with uv, then use the MCP inspector against a local server."},
			eligible: true,
			reason:   "eligible",
		},
		{
			name:     "catalog entry still needs github verification",
			signal:   store.Signal{Source: "skillsmp", OriginalTitle: "MCP Inspector Skill", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "catalog_discovery", Excerpt: "A SkillsMP listing."},
			eligible: false,
			reason:   "github_verification_required",
		},
		{
			name:     "old project is not sent to the model",
			signal:   store.Signal{Source: "github", OriginalTitle: "AI agent", SourceUpdatedAt: timePointer(now.Add(-31 * 24 * time.Hour))},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Install and use this agent."},
			eligible: false,
			reason:   "outside_recency_window",
		},
		{
			name:     "ai is matched as a term rather than inside maintainer",
			signal:   store.Signal{Source: "github", OriginalTitle: "Maintainer helper", SourceUpdatedAt: &recent},
			evidence: store.EvidenceSnapshot{EvidenceClass: "original_documentation", Excerpt: "Install and use this repository helper."},
			eligible: false,
			reason:   "outside_ai_tracks",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision := Qualify(test.signal, test.evidence, now)
			if decision.Eligible != test.eligible || decision.Reason != test.reason {
				t.Fatalf("decision = %#v", decision)
			}
		})
	}
}

func timePointer(value time.Time) *time.Time { return &value }
