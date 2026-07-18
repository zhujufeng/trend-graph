package radar

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

type collectionStore interface {
	List() ([]store.SourceConfig, error)
	RecordCollectionRun(store.CollectionRun) error
}

type collectionTopicStore interface {
	List(activeOnly bool) ([]store.Keyword, error)
}

type commandFunc func(context.Context, string, []string, ...string) ([]byte, error)

// CollectionRunner lets Go own scheduling and audit state while Python owns
// source-specific collection and ingestion.
type CollectionRunner struct {
	store        collectionStore
	topics       collectionTopicStore
	collectorDir string
	backendURL   string
	secret       string
	runCommand   commandFunc
	// ponytail: one personal-server lock; use a distributed lease only with multiple replicas.
	runMu sync.Mutex
}

func NewCollectionRunner(repo collectionStore, topics collectionTopicStore, collectorDir, backendURL, secret string) *CollectionRunner {
	runner := &CollectionRunner{
		store:        repo,
		topics:       topics,
		collectorDir: collectorDir,
		backendURL:   backendURL,
		secret:       secret,
	}
	runner.runCommand = runner.execCommand
	return runner
}

func (r *CollectionRunner) Run(ctx context.Context) error {
	if !r.runMu.TryLock() {
		return nil
	}
	defer r.runMu.Unlock()

	configs, err := r.store.List()
	if err != nil {
		return fmt.Errorf("list source configs: %w", err)
	}
	topicRows, err := r.topics.List(true)
	if err != nil {
		return fmt.Errorf("list active topics: %w", err)
	}
	topics := make([]string, 0, len(topicRows))
	for _, topic := range topicRows {
		if word := strings.TrimSpace(topic.Word); word != "" {
			topics = append(topics, word)
		}
	}

	var runErrors []error
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		if err := r.runSource(ctx, config, topics); err != nil {
			runErrors = append(runErrors, err)
		}
	}
	return errors.Join(runErrors...)
}

func (r *CollectionRunner) runSource(ctx context.Context, config store.SourceConfig, topics []string) error {
	started := time.Now().UTC()
	args, skip, err := collectorArgs(config, topics)
	if skip {
		return nil
	}
	var output []byte
	if err == nil {
		output, err = r.runCommand(ctx, r.collectorDir, []string{
			"BACKEND_URL=" + r.backendURL,
			"INTERNAL_INGEST_SECRET=" + r.secret,
		}, args...)
	}

	finished := time.Now().UTC()
	run := store.CollectionRun{
		Source:     config.Source,
		Status:     "success",
		DurationMS: finished.Sub(started).Milliseconds(),
		StartedAt:  started,
		FinishedAt: &finished,
	}
	if err != nil {
		run.Status = "failed"
		reason := strings.TrimSpace(string(output))
		if reason == "" {
			reason = err.Error()
		}
		run.FailureReason = truncate(reason, 2000)
	} else {
		var result struct {
			Collected int `json:"collected"`
		}
		if decodeErr := json.Unmarshal(output, &result); decodeErr != nil {
			err = fmt.Errorf("decode collector output: %w", decodeErr)
			run.Status = "failed"
			run.FailureReason = truncate(err.Error(), 2000)
		} else {
			run.ItemCount = result.Collected
		}
	}

	if recordErr := r.store.RecordCollectionRun(run); recordErr != nil {
		return fmt.Errorf("%s record collection run: %w", config.Source, recordErr)
	}
	if err != nil {
		return fmt.Errorf("%s collection failed: %w", config.Source, err)
	}
	return nil
}

func collectorArgs(config store.SourceConfig, topics []string) ([]string, bool, error) {
	args := []string{
		"run", "--no-sync", "python", "-m", "signal_collector.cli",
		"--source", config.Source,
		"--limit", "20",
		"--ingest",
	}
	query := strings.Join(topics, ",")
	if query != "" {
		args = append(args, "--topics", query)
	}
	switch config.Source {
	case types.SourceDEV, types.SourceBluesky:
		if query == "" {
			return nil, true, nil
		}
		args = append(args, "--query", query)
	case types.SourceGitHub:
		var settings struct {
			Repositories []string `json:"repositories"`
		}
		if err := json.Unmarshal([]byte(config.SettingsJSON), &settings); err != nil {
			return nil, false, fmt.Errorf("decode github settings: %w", err)
		}
		if query != "" {
			args = append(args, "--query", query)
		}
		if len(settings.Repositories) > 0 {
			args = append(args, "--repositories", strings.Join(settings.Repositories, ","))
		}
		if query == "" && len(settings.Repositories) == 0 {
			return nil, true, nil
		}
	case types.SourceReddit:
		if query == "" {
			return nil, true, nil
		}
		var settings struct {
			Communities []string `json:"communities"`
		}
		if err := json.Unmarshal([]byte(config.SettingsJSON), &settings); err != nil {
			return nil, false, fmt.Errorf("decode reddit settings: %w", err)
		}
		if len(settings.Communities) == 0 {
			return nil, false, errors.New("reddit communities are empty")
		}
		args = append(args, "--communities", strings.Join(settings.Communities, ","))
	case types.SourceRSS:
		if query == "" {
			return nil, true, nil
		}
		var settings struct {
			Feeds []string `json:"feeds"`
		}
		if err := json.Unmarshal([]byte(config.SettingsJSON), &settings); err != nil {
			return nil, false, fmt.Errorf("decode rss settings: %w", err)
		}
		if len(settings.Feeds) == 0 {
			return nil, false, errors.New("rss feeds are empty")
		}
		args = append(args, "--feeds", strings.Join(settings.Feeds, ","))
	default:
		return nil, false, fmt.Errorf("unsupported source %q", config.Source)
	}
	return args, false, nil
}

func (r *CollectionRunner) execCommand(ctx context.Context, dir string, env []string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "uv", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	return cmd.CombinedOutput()
}

func truncate(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
