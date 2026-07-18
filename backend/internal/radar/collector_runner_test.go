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

type fakeCollectionTopicStore struct{ topics []store.Keyword }

func (f *fakeCollectionTopicStore) List(bool) ([]store.Keyword, error) { return f.topics, nil }

func TestCollectionRunnerSkipsOverlappingRounds(t *testing.T) {
	repo := &fakeCollectionStore{configs: []store.SourceConfig{{
		Source: types.SourceGitHub, Enabled: true, SettingsJSON: "{}",
	}}}
	runner := NewCollectionRunner(repo, &fakeCollectionTopicStore{topics: []store.Keyword{{Word: "AI"}}}, "/collector", "http://127.0.0.1:8080", "secret")
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
		{Source: types.SourceGitHub, Enabled: true, SettingsJSON: "{}"},
		{Source: types.SourceReddit, Enabled: true, SettingsJSON: `{"communities":["r/claudeai","r/cursor"]}`},
		{Source: types.SourceBluesky, Enabled: true},
		{Source: types.SourceRSS, Enabled: true, SettingsJSON: `{"feeds":["https://example.com/feed.xml"]}`},
	}}
	runner := NewCollectionRunner(repo, &fakeCollectionTopicStore{topics: []store.Keyword{{Word: "AI"}, {Word: "机器人"}}}, "/collector", "http://127.0.0.1:8080", "secret")
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
	if want := []string{types.SourceDEV, types.SourceGitHub, types.SourceReddit, types.SourceBluesky, types.SourceRSS}; !reflect.DeepEqual(sources, want) {
		t.Fatalf("sources = %v, want %v", sources, want)
	}
	if got := valueAfter(redditArgs, "--communities"); got != "r/claudeai,r/cursor" {
		t.Fatalf("reddit communities = %q", got)
	}
	if len(repo.runs) != 5 {
		t.Fatalf("recorded runs = %d, want 5", len(repo.runs))
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
	if got := valueAfter(sourceArgs[types.SourceDEV], "--query"); got != "AI,机器人" {
		t.Fatalf("dev query = %q", got)
	}
	if got := valueAfter(sourceArgs[types.SourceBluesky], "--query"); got != "AI,机器人" {
		t.Fatalf("bluesky query = %q", got)
	}
	if got := valueAfter(sourceArgs[types.SourceRSS], "--feeds"); got != "https://example.com/feed.xml" {
		t.Fatalf("rss feeds = %q", got)
	}
	if got := valueAfter(sourceArgs[types.SourceGitHub], "--topics"); got != "AI,机器人" {
		t.Fatalf("github topics = %q", got)
	}
}

func TestCollectionRunnerSkipsTopicSourcesWhenNoTopics(t *testing.T) {
	repo := &fakeCollectionStore{configs: []store.SourceConfig{
		{Source: types.SourceDEV, Enabled: true, SettingsJSON: "{}"},
		{Source: types.SourceGitHub, Enabled: true, SettingsJSON: `{"repositories":["openai/codex"]}`},
		{Source: types.SourceRSS, Enabled: true, SettingsJSON: `{"feeds":["https://example.com/feed.xml"]}`},
	}}
	runner := NewCollectionRunner(repo, &fakeCollectionTopicStore{}, "/collector", "http://127.0.0.1:8080", "secret")
	var called []string
	runner.runCommand = func(_ context.Context, _ string, _ []string, args ...string) ([]byte, error) {
		called = append(called, valueAfter(args, "--source"))
		return []byte(`{"collected":1}`), nil
	}
	if err := runner.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(called, []string{types.SourceGitHub}) {
		t.Fatalf("called sources = %v", called)
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
