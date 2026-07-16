package radar

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/store"
)

const dailyAnalysisQuota = 30

type analysisStore interface {
	CountAnalysesSince(time.Time) (int, error)
	ListPendingSignals(limit int) ([]store.RadarSignal, error)
	SetQualification(id int64, qualification, reason string) error
	SaveQualifiedAnalysis(store.SignalAnalysis, string) error
}

type signalAnalyzer interface {
	AnalyzeSignal(context.Context, analyzer.SignalInput, analyzer.EvidenceInput) (analyzer.SignalAnalysisOutput, error)
}

type AnalysisRunner struct {
	store analysisStore
	model signalAnalyzer
	name  string
}

type AnalysisRunResult struct {
	Analyzed       int
	Rejected       int
	QuotaRemaining int
}

func NewAnalysisRunner(store analysisStore, model signalAnalyzer, modelName string) *AnalysisRunner {
	return &AnalysisRunner{store: store, model: model, name: modelName}
}

func (r *AnalysisRunner) Run(ctx context.Context, now time.Time) (AnalysisRunResult, error) {
	_, start, err := shanghaiDay(now)
	used, err := r.store.CountAnalysesSince(start)
	if err != nil {
		return AnalysisRunResult{}, err
	}
	remaining := dailyAnalysisQuota - used
	if remaining <= 0 {
		return AnalysisRunResult{QuotaRemaining: 0}, nil
	}
	items, err := r.store.ListPendingSignals(remaining)
	if err != nil {
		return AnalysisRunResult{}, err
	}
	result := AnalysisRunResult{QuotaRemaining: remaining}
	for _, item := range items {
		if item.Evidence == nil {
			if err := r.store.SetQualification(item.Signal.ID, "rejected", "missing_evidence"); err != nil {
				return result, err
			}
			result.Rejected++
			continue
		}
		decision := Qualify(item.Signal, *item.Evidence, now)
		if !decision.Eligible {
			if err := r.store.SetQualification(item.Signal.ID, "rejected", decision.Reason); err != nil {
				return result, err
			}
			result.Rejected++
			continue
		}
		output, err := r.model.AnalyzeSignal(ctx,
			analyzer.SignalInput{Source: item.Signal.Source, OriginalTitle: item.Signal.OriginalTitle, OriginalURL: item.Signal.OriginalURL},
			analyzer.EvidenceInput{SourceURL: item.Evidence.SourceURL, EvidenceClass: item.Evidence.EvidenceClass, Excerpt: item.Evidence.Excerpt},
		)
		if err != nil {
			return result, err
		}
		if !json.Valid(output.JSON) {
			return result, errors.New("signal analysis is not valid JSON")
		}
		analysis := store.SignalAnalysis{
			SignalID: item.Signal.ID, EvidenceSnapshotID: item.Evidence.ID,
			Model: r.name, AnalysisJSON: string(output.JSON),
			InputTokens: output.InputTokens, OutputTokens: output.OutputTokens,
		}
		if err := r.store.SaveQualifiedAnalysis(analysis, decision.Reason); err != nil {
			return result, err
		}
		result.Analyzed++
		result.QuotaRemaining--
	}
	return result, nil
}
