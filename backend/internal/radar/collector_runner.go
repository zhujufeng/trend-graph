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

type commandFunc func(context.Context, string, []string, ...string) ([]byte, error)

// CollectionRunner lets Go own scheduling and audit state while Python owns
// source-specific collection and ingestion.
type CollectionRunner struct {
	store        collectionStore
	collectorDir string
	backendURL   string
	secret       string
	runCommand   commandFunc
	// ponytail: one personal-server lock; use a distributed lease only with multiple replicas.
	runMu sync.Mutex
}

func NewCollectionRunner(repo collectionStore, collectorDir, backendURL, secret string) *CollectionRunner {
	runner := &CollectionRunner{
		store:        repo,
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

	var runErrors []error
	for _, config := range configs {
		if !config.Enabled {
			continue
		}
		if err := r.runSource(ctx, config); err != nil {
			runErrors = append(runErrors, err)
		}
	}
	return errors.Join(runErrors...)
}

func (r *CollectionRunner) runSource(ctx context.Context, config store.SourceConfig) error {
	started := time.Now().UTC()
	args, err := collectorArgs(config)
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

func collectorArgs(config store.SourceConfig) ([]string, error) {
	args := []string{
		"run", "--no-sync", "python", "-m", "signal_collector.cli",
		"--source", config.Source,
		"--limit", "20",
		"--ingest",
	}
	switch config.Source {
	case types.SourceSkillsMP, types.SourceGitHub:
		args = append(args, "--query", "agent skill mcp")
	case types.SourceReddit:
		var settings struct {
			Communities []string `json:"communities"`
		}
		if err := json.Unmarshal([]byte(config.SettingsJSON), &settings); err != nil {
			return nil, fmt.Errorf("decode reddit settings: %w", err)
		}
		if len(settings.Communities) == 0 {
			return nil, errors.New("reddit communities are empty")
		}
		args = append(args, "--communities", strings.Join(settings.Communities, ","))
	}
	return args, nil
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
