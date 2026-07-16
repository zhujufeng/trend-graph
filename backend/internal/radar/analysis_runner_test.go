package radar

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/store"
)

func TestAnalysisRunnerStopsBeforeModelAtDailyQuota(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, shanghai)
	data := &fakeAnalysisStore{analysesToday: 30}
	model := &fakeSignalAnalyzer{}

	result, err := NewAnalysisRunner(data, model, "deepseek-v4-pro").Run(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if result.QuotaRemaining != 0 || result.Analyzed != 0 {
		t.Fatalf("result = %#v", result)
	}
	if data.listCalled || model.calls != 0 {
		t.Fatalf("listCalled = %v, model calls = %d", data.listCalled, model.calls)
	}
}

func TestAnalysisRunnerPersistsOneQualifiedStructuredAnalysis(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 7, 15, 9, 0, 0, 0, shanghai)
	recent := now.Add(-time.Hour)
	data := &fakeAnalysisStore{pending: []store.RadarSignal{
		{
			Signal:   store.Signal{ID: 7, Source: "github", OriginalTitle: "MCP Inspector Skill", SourceUpdatedAt: &recent},
			Evidence: &store.EvidenceSnapshot{ID: 11, EvidenceClass: "original_documentation", Excerpt: "Install with uv, then use the MCP inspector against a local server."},
		},
	}}
	model := &fakeSignalAnalyzer{output: analyzer.SignalAnalysisOutput{
		JSON:         json.RawMessage(`{"whatChanged":"新增检查流程","action":"本地复现"}`),
		InputTokens:  120,
		OutputTokens: 80,
	}}

	result, err := NewAnalysisRunner(data, model, "deepseek-v4-pro").Run(context.Background(), now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Analyzed != 1 || result.QuotaRemaining != 29 || model.calls != 1 {
		t.Fatalf("result = %#v, model calls = %d", result, model.calls)
	}
	if data.qualification != "qualified" || data.reason != "eligible" {
		t.Fatalf("qualification = %q, reason = %q", data.qualification, data.reason)
	}
	if data.saved.SignalID != 7 || data.saved.EvidenceSnapshotID != 11 || data.saved.Model != "deepseek-v4-pro" {
		t.Fatalf("saved = %#v", data.saved)
	}
	if data.saved.AnalysisJSON != `{"whatChanged":"新增检查流程","action":"本地复现"}` || data.saved.InputTokens != 120 || data.saved.OutputTokens != 80 {
		t.Fatalf("saved analysis = %#v", data.saved)
	}
}

type fakeAnalysisStore struct {
	analysesToday int
	listCalled    bool
	pending       []store.RadarSignal
	qualification string
	reason        string
	saved         store.SignalAnalysis
}

func (s *fakeAnalysisStore) CountAnalysesSince(time.Time) (int, error) {
	return s.analysesToday, nil
}

func (s *fakeAnalysisStore) ListPendingSignals(limit int) ([]store.RadarSignal, error) {
	s.listCalled = true
	return s.pending, nil
}

func (s *fakeAnalysisStore) SetQualification(id int64, qualification, reason string) error {
	s.qualification = qualification
	s.reason = reason
	return nil
}

func (s *fakeAnalysisStore) SaveQualifiedAnalysis(analysis store.SignalAnalysis, reason string) error {
	s.saved = analysis
	s.qualification = "qualified"
	s.reason = reason
	return nil
}

type fakeSignalAnalyzer struct {
	calls  int
	output analyzer.SignalAnalysisOutput
}

func (a *fakeSignalAnalyzer) AnalyzeSignal(context.Context, analyzer.SignalInput, analyzer.EvidenceInput) (analyzer.SignalAnalysisOutput, error) {
	a.calls++
	return a.output, nil
}
