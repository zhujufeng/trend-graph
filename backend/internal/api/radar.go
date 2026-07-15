package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
)

type radarStore interface {
	ListRadarSignals(limit int) ([]store.RadarSignal, error)
}

type RadarHandler struct {
	signals radarStore
}

func NewRadarHandler(signals radarStore) *RadarHandler {
	return &RadarHandler{signals: signals}
}

func (h *RadarHandler) Register(api *gin.RouterGroup) {
	api.GET("/radar/signals", h.ListSignals)
}

type radarSignalResponse struct {
	ID                  int64                   `json:"id"`
	Source              string                  `json:"source"`
	Title               string                  `json:"title"`
	OriginalURL         string                  `json:"originalUrl"`
	Author              string                  `json:"author,omitempty"`
	Score               float64                 `json:"score"`
	Qualification       string                  `json:"qualification"`
	QualificationReason string                  `json:"qualificationReason,omitempty"`
	LifecycleState      string                  `json:"lifecycleState"`
	SourcePublishedAt   *time.Time              `json:"sourcePublishedAt,omitempty"`
	SourceUpdatedAt     *time.Time              `json:"sourceUpdatedAt,omitempty"`
	CreatedAt           time.Time               `json:"createdAt"`
	Evidence            *store.EvidenceSnapshot `json:"evidence,omitempty"`
	Analysis            json.RawMessage         `json:"analysis,omitempty"`
}

func (h *RadarHandler) ListSignals(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	items, err := h.signals.ListRadarSignals(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list radar signals"})
		return
	}
	response := make([]radarSignalResponse, 0, len(items))
	for _, item := range items {
		view := radarSignalResponse{
			ID:                  item.Signal.ID,
			Source:              item.Signal.Source,
			Title:               item.Signal.OriginalTitle,
			OriginalURL:         item.Signal.OriginalURL,
			Author:              item.Signal.Author,
			Score:               item.Signal.Score,
			Qualification:       item.Signal.Qualification,
			QualificationReason: item.Signal.QualificationReason,
			LifecycleState:      item.Signal.LifecycleState,
			SourcePublishedAt:   item.Signal.SourcePublishedAt,
			SourceUpdatedAt:     item.Signal.SourceUpdatedAt,
			CreatedAt:           item.Signal.CreatedAt,
			Evidence:            item.Evidence,
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
