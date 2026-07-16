package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trend-graph/internal/analyzer"
	"trend-graph/internal/store"
)

type contentPackageStore interface {
	GetRadarSignal(id int64) (store.RadarSignal, error)
	CreateContentPackage(*store.ContentPackage) error
	GetContentPackage(id int64) (store.ContentPackage, error)
	GetEvidenceSnapshot(id int64) (store.EvidenceSnapshot, error)
	UpdateContentPackage(store.ContentPackage) error
	ApproveContentPackage(id int64, now time.Time) error
}

type contentPackageGenerator interface {
	GenerateContentPackage(context.Context, analyzer.SignalInput, analyzer.EvidenceInput, json.RawMessage) (analyzer.ContentPackageDraft, error)
}

type ContentPackageHandler struct {
	store     contentPackageStore
	generator contentPackageGenerator
}

func NewContentPackageHandler(contentStore contentPackageStore, generator contentPackageGenerator) *ContentPackageHandler {
	return &ContentPackageHandler{store: contentStore, generator: generator}
}

func (h *ContentPackageHandler) Register(api *gin.RouterGroup) {
	api.POST("/radar/signals/:id/content-packages", h.Create)
	api.GET("/content-packages/:id", h.Get)
	api.PUT("/content-packages/:id", h.Update)
	api.POST("/content-packages/:id/approve", h.Approve)
}

type contentPackageResponse struct {
	ID                 int64           `json:"id"`
	SignalID           int64           `json:"signalId"`
	EvidenceSnapshotID int64           `json:"evidenceSnapshotId"`
	Status             string          `json:"status"`
	Strategy           json.RawMessage `json:"strategy"`
	Xiaohongshu        json.RawMessage `json:"xiaohongshu"`
	Wechat             json.RawMessage `json:"wechat"`
	X                  json.RawMessage `json:"x"`
	VisualPlan         json.RawMessage `json:"visualPlan"`
	ApprovedAt         *time.Time      `json:"approvedAt,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

func (h *ContentPackageHandler) Create(c *gin.Context) {
	if h.generator == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "content model is not configured"})
		return
	}
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	item, err := h.store.GetRadarSignal(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "signal not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load signal"})
		return
	}
	if item.Signal.Qualification != "qualified" || item.Evidence == nil || item.Analysis == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "signal must be qualified with preserved evidence before content generation"})
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Minute)
	defer cancel()
	draft, err := h.generator.GenerateContentPackage(ctx,
		analyzer.SignalInput{Source: item.Signal.Source, OriginalTitle: item.Signal.OriginalTitle, OriginalURL: item.Signal.OriginalURL},
		analyzer.EvidenceInput{SourceURL: item.Evidence.SourceURL, EvidenceClass: item.Evidence.EvidenceClass, Excerpt: item.Evidence.Excerpt},
		json.RawMessage(item.Analysis.AnalysisJSON),
	)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "could not generate content package"})
		return
	}
	content, err := contentPackageFromDraft(item, draft)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode content package"})
		return
	}
	if err := h.store.CreateContentPackage(&content); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not save content package"})
		return
	}
	view, _ := contentPackageView(content)
	c.JSON(http.StatusCreated, gin.H{"data": view})
}

func (h *ContentPackageHandler) Get(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content package id"})
		return
	}
	content, err := h.store.GetContentPackage(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "content package not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load content package"})
		return
	}
	view, err := contentPackageView(content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid stored content package"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

type updateContentPackageRequest struct {
	Strategy    json.RawMessage `json:"strategy"`
	Xiaohongshu json.RawMessage `json:"xiaohongshu"`
	Wechat      json.RawMessage `json:"wechat"`
	X           json.RawMessage `json:"x"`
	VisualPlan  json.RawMessage `json:"visualPlan"`
}

func (h *ContentPackageHandler) Update(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content package id"})
		return
	}
	var req updateContentPackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content package payload"})
		return
	}
	existing, err := h.store.GetContentPackage(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content package not found"})
		return
	}
	evidence, err := h.store.GetEvidenceSnapshot(existing.EvidenceSnapshotID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not load package evidence"})
		return
	}
	if !validContentPackagePayload(req, evidence.EvidenceClass) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content package payload"})
		return
	}
	content := store.ContentPackage{
		ID: id, StrategyJSON: string(req.Strategy), XiaohongshuJSON: string(req.Xiaohongshu),
		WechatJSON: string(req.Wechat), XJSON: string(req.X), VisualPlanJSON: string(req.VisualPlan),
	}
	if err := h.store.UpdateContentPackage(content); errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusConflict, gin.H{"error": "content package not found or already approved"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update content package"})
		return
	}
	content, err = h.store.GetContentPackage(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not reload content package"})
		return
	}
	view, _ := contentPackageView(content)
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *ContentPackageHandler) Approve(c *gin.Context) {
	id, err := parseID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content package id"})
		return
	}
	if err := h.store.ApproveContentPackage(id, time.Now().UTC()); errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "content package not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not approve content package"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"approved": true})
}

func contentPackageFromDraft(item store.RadarSignal, draft analyzer.ContentPackageDraft) (store.ContentPackage, error) {
	parts := []any{draft.Strategy, draft.Xiaohongshu, draft.Wechat, draft.X, draft.VisualPlan}
	encoded := make([]string, len(parts))
	for index, part := range parts {
		value, err := json.Marshal(part)
		if err != nil {
			return store.ContentPackage{}, err
		}
		encoded[index] = string(value)
	}
	return store.ContentPackage{
		SignalID: item.Signal.ID, EvidenceSnapshotID: item.Evidence.ID, Status: "draft",
		StrategyJSON: encoded[0], XiaohongshuJSON: encoded[1], WechatJSON: encoded[2],
		XJSON: encoded[3], VisualPlanJSON: encoded[4],
	}, nil
}

func contentPackageView(content store.ContentPackage) (contentPackageResponse, error) {
	values := []json.RawMessage{
		json.RawMessage(content.StrategyJSON), json.RawMessage(content.XiaohongshuJSON),
		json.RawMessage(content.WechatJSON), json.RawMessage(content.XJSON), json.RawMessage(content.VisualPlanJSON),
	}
	for _, value := range values {
		if !json.Valid(value) {
			return contentPackageResponse{}, errors.New("invalid content package JSON")
		}
	}
	return contentPackageResponse{
		ID: content.ID, SignalID: content.SignalID, EvidenceSnapshotID: content.EvidenceSnapshotID,
		Status: content.Status, Strategy: values[0], Xiaohongshu: values[1], Wechat: values[2],
		X: values[3], VisualPlan: values[4], ApprovedAt: content.ApprovedAt,
		CreatedAt: content.CreatedAt, UpdatedAt: content.UpdatedAt,
	}, nil
}

func validContentPackagePayload(req updateContentPackageRequest, evidenceClass string) bool {
	for _, value := range []json.RawMessage{req.Strategy, req.Xiaohongshu, req.Wechat, req.X, req.VisualPlan} {
		if len(value) == 0 || !json.Valid(value) {
			return false
		}
	}
	var strategy analyzer.ContentStrategy
	var xiaohongshu, wechat analyzer.PlatformDraft
	var x analyzer.XDraft
	var visualPlan []analyzer.VisualAsset
	if json.Unmarshal(req.Strategy, &strategy) != nil || json.Unmarshal(req.Xiaohongshu, &xiaohongshu) != nil ||
		json.Unmarshal(req.Wechat, &wechat) != nil || json.Unmarshal(req.X, &x) != nil || json.Unmarshal(req.VisualPlan, &visualPlan) != nil {
		return false
	}
	if strategy.Angle == "" || xiaohongshu.Body == "" || wechat.Body == "" || x.Chinese == "" || x.English == "" ||
		len(xiaohongshu.SourceLinks) == 0 || len(wechat.SourceLinks) == 0 || len(x.SourceLinks) == 0 || len(visualPlan) == 0 {
		return false
	}
	for _, asset := range visualPlan {
		if asset.Purpose == "" || asset.AspectRatio == "" || asset.Prompt == "" {
			return false
		}
	}
	allText := strategy.Angle + xiaohongshu.Title + xiaohongshu.Body + wechat.Title + wechat.Body + x.Chinese + x.English
	if evidenceClass != "user_verified" && analyzer.ContainsFirstPersonVerification(allText) {
		return false
	}
	return true
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}
