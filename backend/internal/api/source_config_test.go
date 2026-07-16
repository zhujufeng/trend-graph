package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
)

func TestSourceConfigAPIUpdatesRedditAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeSourceConfigStore{
		configs: []store.SourceConfig{
			{Source: "waytoagi", Enabled: true, SettingsJSON: "{}"},
			{Source: "reddit", Enabled: true, SettingsJSON: `{"communities":["r/LocalLLaMA"]}`},
		},
	}
	router := gin.New()
	NewSourceConfigHandler(repo).Register(router.Group("/api"))

	body := []byte(`{"enabled":false,"redditCommunities":["r/ClaudeAI", "claudeai", " r/cursor ", "r/all", ""]}`)
	request := httptest.NewRequest(http.MethodPut, "/api/source-configs/reddit", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !repo.updated {
		t.Fatal("expected config update")
	}
	if repo.config.Source != "reddit" || repo.config.Enabled {
		t.Fatalf("stored config = %#v", repo.config)
	}
	var settings struct {
		Communities []string `json:"communities"`
	}
	if err := json.Unmarshal([]byte(repo.config.SettingsJSON), &settings); err != nil {
		t.Fatalf("settings JSON: %v", err)
	}
	want := []string{"r/claudeai", "r/cursor"}
	if len(settings.Communities) != len(want) {
		t.Fatalf("communities = %#v, want %#v", settings.Communities, want)
	}
	for index, community := range want {
		if settings.Communities[index] != community {
			t.Fatalf("communities = %#v, want %#v", settings.Communities, want)
		}
	}
}

func TestSourceConfigAPIListsCurrentSourceStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeSourceConfigStore{
		configs: []store.SourceConfig{
			{Source: "waytoagi", Enabled: true, SettingsJSON: "{}"},
			{Source: "reddit", Enabled: false, SettingsJSON: `{"communities":["r/cursor"]}`},
		},
		runs: map[string]store.CollectionRun{
			"reddit": {Source: "reddit", Status: "success", ItemCount: 6, DurationMS: 1200},
		},
	}
	router := gin.New()
	NewSourceConfigHandler(repo).Register(router.Group("/api"))

	request := httptest.NewRequest(http.MethodGet, "/api/source-configs", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []struct {
			Source   string `json:"source"`
			Enabled  bool   `json:"enabled"`
			Settings struct {
				Communities []string `json:"communities"`
			} `json:"settings"`
			LastRun *store.CollectionRun `json:"lastRun"`
		} `json:"data"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if payload.Count != 2 || len(payload.Data) != 2 || payload.Data[1].Source != "reddit" {
		t.Fatalf("payload = %#v", payload)
	}
	if len(payload.Data[1].Settings.Communities) != 1 || payload.Data[1].Settings.Communities[0] != "r/cursor" {
		t.Fatalf("settings = %#v", payload.Data[1].Settings)
	}
	if payload.Data[1].LastRun == nil || payload.Data[1].LastRun.ItemCount != 6 || payload.Data[1].LastRun.DurationMS != 1200 {
		t.Fatalf("last run = %#v", payload.Data[1].LastRun)
	}
}

func TestSourceConfigAPIUpdatePreservesUnchangedSettings(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeSourceConfigStore{
		configs: []store.SourceConfig{
			{Source: "reddit", Enabled: true, SettingsJSON: `{"communities":["r/claudeai"]}`},
		},
	}
	router := gin.New()
	NewSourceConfigHandler(repo).Register(router.Group("/api"))

	request := httptest.NewRequest(http.MethodPut, "/api/source-configs/reddit", bytes.NewBufferString(`{"enabled":false}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if repo.config.SettingsJSON != `{"communities":["r/claudeai"]}` {
		t.Fatalf("settings = %s", repo.config.SettingsJSON)
	}
}

func TestSourceConfigAPIRejectsUnsupportedSource(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	NewSourceConfigHandler(&fakeSourceConfigStore{}).Register(router.Group("/api"))

	request := httptest.NewRequest(http.MethodPut, "/api/source-configs/linuxdo", bytes.NewBufferString(`{"enabled":true}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

type fakeSourceConfigStore struct {
	configs []store.SourceConfig
	runs    map[string]store.CollectionRun
	config  store.SourceConfig
	updated bool
}

func (s *fakeSourceConfigStore) List() ([]store.SourceConfig, error) {
	return s.configs, nil
}

func (s *fakeSourceConfigStore) LatestRuns() (map[string]store.CollectionRun, error) {
	return s.runs, nil
}

func (s *fakeSourceConfigStore) Save(config store.SourceConfig) (*store.SourceConfig, error) {
	s.config = config
	s.updated = true
	return &s.config, nil
}
