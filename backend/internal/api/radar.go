package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"trend-graph/internal/radar"
	"trend-graph/internal/store"
)

type radarStore interface {
	ListRadarSignals(limit int) ([]store.RadarSignal, error)
	UpdateLifecycleState(id int64, state string) error
}

type RadarHandler struct {
	signals radarStore
}

func NewRadarHandler(signals radarStore) *RadarHandler {
	return &RadarHandler{signals: signals}
}

func (h *RadarHandler) Register(api *gin.RouterGroup) {
	api.GET("/radar/signals", h.ListSignals)
	api.PATCH("/radar/signals/:id/lifecycle", h.UpdateLifecycle)
}

type radarSignalResponse struct {
	ID                  int64                  `json:"id"`
	Source              string                 `json:"source"`
	Title               string                 `json:"title"`
	OriginalURL         string                 `json:"originalUrl"`
	Author              string                 `json:"author,omitempty"`
	Score               float64                `json:"score"`
	RankScore           int                    `json:"rankScore"`
	Qualification       string                 `json:"qualification"`
	QualificationReason string                 `json:"qualificationReason,omitempty"`
	LifecycleState      string                 `json:"lifecycleState"`
	SourcePublishedAt   *time.Time             `json:"sourcePublishedAt,omitempty"`
	SourceUpdatedAt     *time.Time             `json:"sourceUpdatedAt,omitempty"`
	CreatedAt           time.Time              `json:"createdAt"`
	Evidence            *radarEvidenceResponse `json:"evidence,omitempty"`
	Analysis            json.RawMessage        `json:"analysis,omitempty"`
}

// radarEvidenceResponse intentionally excludes the raw excerpt and content
// hash. The dashboard needs provenance, not the source document body.
type radarEvidenceResponse struct {
	ID            int64     `json:"id"`
	SignalID      int64     `json:"signalId"`
	SourceURL     string    `json:"sourceUrl"`
	EvidenceClass string    `json:"evidenceClass"`
	Title         string    `json:"title,omitempty"`
	CapturedAt    time.Time `json:"capturedAt"`
}

func (h *RadarHandler) ListSignals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	items, err := h.signals.ListRadarSignals(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list radar signals"})
		return
	}
	ranked := radar.RankSignals(items, time.Now())
	if len(ranked) > limit {
		ranked = ranked[:limit]
	}
	response := make([]radarSignalResponse, 0, len(ranked))
	for _, rankedItem := range ranked {
		item := rankedItem.Item
		view := radarSignalResponse{
			ID:                  item.Signal.ID,
			Source:              item.Signal.Source,
			Title:               item.Signal.OriginalTitle,
			OriginalURL:         item.Signal.OriginalURL,
			Author:              item.Signal.Author,
			Score:               item.Signal.Score,
			RankScore:           rankedItem.RankScore,
			Qualification:       item.Signal.Qualification,
			QualificationReason: item.Signal.QualificationReason,
			LifecycleState:      item.Signal.LifecycleState,
			SourcePublishedAt:   item.Signal.SourcePublishedAt,
			SourceUpdatedAt:     item.Signal.SourceUpdatedAt,
			CreatedAt:           item.Signal.CreatedAt,
		}
		if item.Evidence != nil {
			view.Evidence = &radarEvidenceResponse{
				ID: item.Evidence.ID, SignalID: item.Evidence.SignalID,
				SourceURL: item.Evidence.SourceURL, EvidenceClass: item.Evidence.EvidenceClass,
				Title: item.Evidence.Title, CapturedAt: item.Evidence.CapturedAt,
			}
		}
		if item.Analysis != nil {
			view.Analysis = json.RawMessage(item.Analysis.AnalysisJSON)
			if !json.Valid(view.Analysis) {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid stored signal analysis"})
				return
			}
		}
		response = append(response, view)
	}
	c.JSON(http.StatusOK, gin.H{"data": response, "count": len(response)})
}

func (h *RadarHandler) UpdateLifecycle(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	var request struct {
		State string `json:"state"`
	}
	if c.ShouldBindJSON(&request) != nil || !validLifecycleState(request.State) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "state must be inbox, saved, done, or dismissed"})
		return
	}
	if err := h.signals.UpdateLifecycleState(id, request.State); errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "qualified signal not found"})
		return
	} else if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update signal lifecycle"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "state": request.State})
}

func validLifecycleState(state string) bool {
	switch state {
	case store.LifecycleInbox, store.LifecycleSaved, store.LifecycleDone, store.LifecycleDismissed:
		return true
	default:
		return false
	}
}
