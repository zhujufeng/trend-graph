package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/store"
)

func TestContentPackageRequiresQualifiedEvidenceAndPersistsGeneratedDraft(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &fakeContentPackageStore{signal: store.RadarSignal{
		Signal:   store.Signal{ID: 7, OriginalTitle: "Agent Workflow", OriginalURL: "https://example.com/release", Qualification: "qualified", LifecycleState: "practiced"},
		Evidence: &store.EvidenceSnapshot{ID: 8, SourceURL: "https://example.com/docs", EvidenceClass: "original_documentation", Excerpt: "Install then run."},
		Analysis: &store.SignalAnalysis{AnalysisJSON: `{"action":"复现"}`},
	}}
	handler := NewContentPackageHandler(repo, fakeContentGenerator{})
	router := gin.New()
	handler.Register(router.Group("/api"))

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/radar/signals/7/content-packages", nil))
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if repo.content.SignalID != 7 || repo.content.EvidenceSnapshotID != 8 || repo.content.Status != "draft" {
		t.Fatalf("stored content = %#v", repo.content)
	}
	if !json.Valid([]byte(repo.content.XJSON)) || !json.Valid([]byte(repo.content.VisualPlanJSON)) {
		t.Fatalf("stored JSON is invalid: %#v", repo.content)
	}

	repo.signal.Signal.LifecycleState = "queued"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/radar/signals/7/content-packages", nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("unpracticed status = %d", response.Code)
	}

	repo.signal.Signal.Qualification = "pending"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/radar/signals/7/content-packages", nil))
	if response.Code != http.StatusConflict {
		t.Fatalf("pending status = %d", response.Code)
	}
}

func TestContentPackageEditsCannotUpgradeThirdPartyEvidenceToPersonalTesting(t *testing.T) {
	req := updateContentPackageRequest{
		Strategy:    json.RawMessage(`{"angle":"复现","audience":"开发者","evidenceNote":"第三方实践"}`),
		Xiaohongshu: json.RawMessage(`{"title":"教程","body":"我实测提升十倍","sourceLinks":["https://example.com/docs"]}`),
		Wechat:      json.RawMessage(`{"title":"教程","body":"按文档复现","sourceLinks":["https://example.com/docs"]}`),
		X:           json.RawMessage(`{"chinese":"中文","english":"English","sourceLinks":["https://example.com/docs"]}`),
		VisualPlan:  json.RawMessage(`[{"purpose":"封面","aspectRatio":"3:4","prompt":"信息图"}]`),
	}
	if validContentPackagePayload(req, "documented_third_party_practice") {
		t.Fatal("third-party evidence must not allow first-person testing claims")
	}
	if !validContentPackagePayload(req, "user_verified") {
		t.Fatal("user-verified evidence should allow first-person testing claims")
	}
}

type fakeContentGenerator struct{}

func (fakeContentGenerator) GenerateContentPackage(context.Context, analyzer.SignalInput, analyzer.EvidenceInput, json.RawMessage) (analyzer.ContentPackageDraft, error) {
	return analyzer.ContentPackageDraft{
		Strategy:    analyzer.ContentStrategy{Angle: "三步复现", Audience: "AI 用户", EvidenceNote: "官方文档"},
		Xiaohongshu: analyzer.PlatformDraft{Title: "小红书", Body: "正文", SourceLinks: []string{"https://example.com/docs"}},
		Wechat:      analyzer.PlatformDraft{Title: "公众号", Body: "长文", SourceLinks: []string{"https://example.com/docs"}},
		X:           analyzer.XDraft{Chinese: "中文", English: "English", SourceLinks: []string{"https://example.com/docs"}},
		VisualPlan:  []analyzer.VisualAsset{{Purpose: "封面", AspectRatio: "3:4", Prompt: "中文信息图"}},
	}, nil
}

type fakeContentPackageStore struct {
	signal  store.RadarSignal
	content store.ContentPackage
}

func (f *fakeContentPackageStore) GetRadarSignal(int64) (store.RadarSignal, error) {
	return f.signal, nil
}
func (f *fakeContentPackageStore) CreateContentPackage(content *store.ContentPackage) error {
	content.ID = 11
	content.CreatedAt = time.Now()
	f.content = *content
	return nil
}
func (f *fakeContentPackageStore) GetContentPackage(int64) (store.ContentPackage, error) {
	return f.content, nil
}
func (f *fakeContentPackageStore) GetEvidenceSnapshot(int64) (store.EvidenceSnapshot, error) {
	return *f.signal.Evidence, nil
}
func (f *fakeContentPackageStore) UpdateContentPackage(content store.ContentPackage) error {
	f.content = content
	return nil
}
func (f *fakeContentPackageStore) ApproveContentPackage(int64, time.Time) error { return nil }
