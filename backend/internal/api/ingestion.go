package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

const internalIngestSecretHeader = "X-Internal-Ingest-Secret"

// IngestionHandler is a narrow private boundary for the Python collector. It
// validates collector payloads once before they can reach persistence.
type IngestionHandler struct {
	secret  []byte
	signals signalIngestor
}

type signalIngestor interface {
	IngestIfNew(store.Signal, store.EvidenceSnapshot) (bool, error)
}

func NewIngestionHandler(secret string, signals signalIngestor) *IngestionHandler {
	return &IngestionHandler{secret: []byte(secret), signals: signals}
}

func (h *IngestionHandler) Register(r *gin.Engine) {
	r.POST("/internal/ingest/signals", h.requireSecret(), h.IngestSignal)
}

type ingestSignalRequest struct {
	Source          string     `json:"source"`
	OriginalURL     string     `json:"originalUrl"`
	OriginalTitle   string     `json:"originalTitle"`
	Author          string     `json:"author"`
	Score           float64    `json:"score"`
	PublishedAt     *time.Time `json:"publishedAt"`
	UpdatedAt       *time.Time `json:"updatedAt"`
	EvidenceURL     string     `json:"evidenceUrl"`
	EvidenceTitle   string     `json:"evidenceTitle"`
	EvidenceClass   string     `json:"evidenceClass"`
	EvidenceExcerpt string     `json:"evidenceExcerpt"`
}

func (h *IngestionHandler) IngestSignal(c *gin.Context) {
	var req ingestSignalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ingestion payload"})
		return
	}
	if err := req.validate(); err != "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}
	created, err := h.signals.IngestIfNew(
		store.Signal{
			Source:            req.Source,
			OriginalURL:       req.OriginalURL,
			OriginalTitle:     req.OriginalTitle,
			Author:            req.Author,
			Score:             req.Score,
			SourcePublishedAt: req.PublishedAt,
			SourceUpdatedAt:   req.UpdatedAt,
		},
		store.EvidenceSnapshot{
			SourceURL:     req.EvidenceURL,
			EvidenceClass: req.EvidenceClass,
			Title:         req.EvidenceTitle,
			Excerpt:       req.EvidenceExcerpt,
			CapturedAt:    time.Now().UTC(),
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not ingest signal"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"created": created})
}

func (h *IngestionHandler) requireSecret() gin.HandlerFunc {
	return func(c *gin.Context) {
		provided := []byte(c.GetHeader(internalIngestSecretHeader))
		if len(h.secret) == 0 || len(provided) != len(h.secret) || subtle.ConstantTimeCompare(provided, h.secret) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "internal authentication required"})
			return
		}
		c.Next()
	}
}

func (r ingestSignalRequest) validate() string {
	if !isEnabledCollectorSource(r.Source) {
		return "unsupported source"
	}
	if strings.TrimSpace(r.OriginalURL) == "" || strings.TrimSpace(r.OriginalTitle) == "" {
		return "originalUrl and originalTitle are required"
	}
	if strings.TrimSpace(r.EvidenceURL) == "" || strings.TrimSpace(r.EvidenceClass) == "" || strings.TrimSpace(r.EvidenceExcerpt) == "" {
		return "evidenceUrl, evidenceClass, and evidenceExcerpt are required"
	}
	return ""
}

func isEnabledCollectorSource(source string) bool {
	return types.IsRadarSource(source)
}
