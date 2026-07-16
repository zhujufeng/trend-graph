package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
)

func TestRadarSignalsAPIReturnsEvidenceAndStructuredAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC)
	repo := &fakeRadarStore{signals: []store.RadarSignal{
		{
			Signal: store.Signal{
				ID: 7, Source: "github", OriginalTitle: "MCP Inspector",
				OriginalURL: "https://github.com/owner/repo", Score: 42,
				Qualification: "qualified", LifecycleState: "new", CreatedAt: now,
			},
			Evidence: &store.EvidenceSnapshot{
				SourceURL:     "https://github.com/owner/repo/blob/main/SKILL.md",
				EvidenceClass: "original_documentation", Excerpt: "Install and run the inspector.", CapturedAt: now,
			},
			Analysis: &store.SignalAnalysis{AnalysisJSON: `{"whatChanged":"新增 MCP 检查流程","action":"用测试服务器复现"}`},
		},
	}}
	router := gin.New()
	NewRadarHandler(repo).Register(router.Group("/api"))

	request := httptest.NewRequest(http.MethodGet, "/api/radar/signals?limit=8", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload struct {
		Data []struct {
			ID            int64  `json:"id"`
			Title         string `json:"title"`
			Qualification string `json:"qualification"`
			Evidence      struct {
				SourceURL     string `json:"sourceUrl"`
				EvidenceClass string `json:"evidenceClass"`
			} `json:"evidence"`
			Analysis struct {
				WhatChanged string `json:"whatChanged"`
				Action      string `json:"action"`
			} `json:"analysis"`
		} `json:"data"`
		Count int `json:"count"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response JSON: %v", err)
	}
	if payload.Count != 1 || len(payload.Data) != 1 {
		t.Fatalf("payload = %#v", payload)
	}
	item := payload.Data[0]
	if item.ID != 7 || item.Title != "MCP Inspector" || item.Qualification != "qualified" {
		t.Fatalf("item = %#v", item)
	}
	if item.Evidence.EvidenceClass != "original_documentation" || item.Evidence.SourceURL == "" {
		t.Fatalf("evidence = %#v", item.Evidence)
	}
	if bytes.Contains(response.Body.Bytes(), []byte(`"excerpt"`)) {
		t.Fatalf("dashboard response leaked full evidence excerpt: %s", response.Body.String())
	}
	if item.Analysis.WhatChanged != "新增 MCP 检查流程" || item.Analysis.Action != "用测试服务器复现" {
		t.Fatalf("analysis = %#v", item.Analysis)
	}
}

func TestRadarSignalLifecycleAcceptsOnlyQualifiedWorkflowStates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeRadarStore{}
	router := gin.New()
	NewRadarHandler(repo).Register(router.Group("/api"))

	request := httptest.NewRequest(http.MethodPatch, "/api/radar/signals/7/lifecycle", bytes.NewBufferString(`{"state":"queued"}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK || repo.updatedID != 7 || repo.updatedState != "queued" {
		t.Fatalf("status=%d id=%d state=%q body=%s", response.Code, repo.updatedID, repo.updatedState, response.Body.String())
	}

	request = httptest.NewRequest(http.MethodPatch, "/api/radar/signals/7/lifecycle", bytes.NewBufferString(`{"state":"published"}`))
	request.Header.Set("Content-Type", "application/json")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid state status = %d", response.Code)
	}
}

type fakeRadarStore struct {
	signals      []store.RadarSignal
	updatedID    int64
	updatedState string
}

func (s *fakeRadarStore) ListRadarSignals(limit int) ([]store.RadarSignal, error) {
	return s.signals, nil
}

func (s *fakeRadarStore) UpdateLifecycleState(id int64, state string) error {
	s.updatedID, s.updatedState = id, state
	return nil
}
