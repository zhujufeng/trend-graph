package radar

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

type fakeCollectionStore struct {
	configs []store.SourceConfig
	runs    []store.CollectionRun
}

func TestCollectionRunnerSkipsOverlappingRounds(t *testing.T) {
	repo := &fakeCollectionStore{configs: []store.SourceConfig{{
		Source: types.SourceGitHub, Enabled: true,
	}}}
	runner := NewCollectionRunner(repo, "/collector", "http://127.0.0.1:8080", "secret")
	started := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan error, 1)
	var calls atomic.Int32
	runner.runCommand = func(_ context.Context, _ string, _ []string, _ ...string) ([]byte, error) {
		if calls.Add(1) == 1 {
			close(started)
			<-release
		}
		return []byte(`{"collected":1,"created":1}`), nil
	}

	go func() { firstDone <- runner.Run(context.Background()) }()
	<-started
	secondDone := make(chan error, 1)
	go func() { secondDone <- runner.Run(context.Background()) }()
	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("overlapping Run error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("overlapping Run did not skip promptly")
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("collector command calls = %d, want 1", got)
	}
}

func (f *fakeCollectionStore) List() ([]store.SourceConfig, error) {
	return f.configs, nil
}

func (f *fakeCollectionStore) RecordCollectionRun(run store.CollectionRun) error {
	f.runs = append(f.runs, run)
	return nil
}

func TestCollectionRunnerRunsEnabledSourcesAndRecordsEachResult(t *testing.T) {
	repo := &fakeCollectionStore{configs: []store.SourceConfig{
		{Source: types.SourceDEV, Enabled: true},
		{Source: types.SourceGitHub, Enabled: true},
		{Source: types.SourceReddit, Enabled: true, SettingsJSON: `{"communities":["r/claudeai","r/cursor"]}`},
		{Source: types.SourceBluesky, Enabled: true},
	}}
	runner := NewCollectionRunner(repo, "/collector", "http://127.0.0.1:8080", "secret")
	var sources []string
	var redditArgs []string
	sourceArgs := map[string][]string{}
	runner.runCommand = func(_ context.Context, _ string, _ []string, args ...string) ([]byte, error) {
		source := valueAfter(args, "--source")
		sources = append(sources, source)
		sourceArgs[source] = append([]string(nil), args...)
		if source == types.SourceReddit {
			redditArgs = append([]string(nil), args...)
			return []byte("reddit oauth denied"), errors.New("exit status 1")
		}
		return []byte(`{"collected":2,"created":1}`), nil
	}

	err := runner.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), types.SourceReddit) {
		t.Fatalf("Run error = %v, want reddit failure", err)
	}
	if want := []string{types.SourceDEV, types.SourceGitHub, types.SourceReddit, types.SourceBluesky}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("sources = %v, want %v", sources, want)
	}
	if got := valueAfter(redditArgs, "--communities"); got != "r/claudeai,r/cursor" {
		t.Fatalf("reddit communities = %q", got)
	}
	if len(repo.runs) != 4 {
		t.Fatalf("recorded runs = %d, want 4", len(repo.runs))
	}
	if repo.runs[0].Status != "success" || repo.runs[0].ItemCount != 2 {
		t.Fatalf("github run = %#v", repo.runs[0])
	}
	if repo.runs[2].Status != "failed" || !strings.Contains(repo.runs[2].FailureReason, "oauth denied") {
		t.Fatalf("reddit run = %#v", repo.runs[2])
	}
	if repo.runs[3].Status != "success" {
		t.Fatalf("bluesky run = %#v", repo.runs[3])
	}
	if got := valueAfter(sourceArgs[types.SourceDEV], "--query"); got != "mcp,claudecode,agents,ai" {
		t.Fatalf("dev query = %q", got)
	}
	if got := valueAfter(sourceArgs[types.SourceBluesky], "--query"); got != "MCP,Claude Code,Agent Skills,Codex" {
		t.Fatalf("bluesky query = %q", got)
	}
}

func valueAfter(values []string, key string) string {
	for i := 0; i+1 < len(values); i++ {
		if values[i] == key {
			return values[i+1]
		}
	}
	return ""
}
