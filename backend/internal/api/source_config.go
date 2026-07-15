package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"trend-graph/internal/store"
	"trend-graph/internal/types"
)

type sourceConfigStore interface {
	List() ([]store.SourceConfig, error)
	Save(store.SourceConfig) (*store.SourceConfig, error)
}

// SourceConfigHandler exposes the administrator's source controls. It is
// registered below the authenticated /api group by main.
type SourceConfigHandler struct {
	configs sourceConfigStore
}

func NewSourceConfigHandler(configs sourceConfigStore) *SourceConfigHandler {
	return &SourceConfigHandler{configs: configs}
}

func (h *SourceConfigHandler) Register(api *gin.RouterGroup) {
	api.GET("/source-configs", h.List)
	api.PUT("/source-configs/:source", h.Update)
}

type updateSourceConfigRequest struct {
	Enabled           *bool    `json:"enabled"`
	RedditCommunities []string `json:"redditCommunities"`
}

type sourceConfigResponse struct {
	Source        string          `json:"source"`
	Enabled       bool            `json:"enabled"`
	Settings      json.RawMessage `json:"settings"`
	LastSuccessAt *time.Time      `json:"lastSuccessAt,omitempty"`
	LastFailure   string          `json:"lastFailure,omitempty"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

func (h *SourceConfigHandler) List(c *gin.Context) {
	configs, err := h.configs.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list source configs"})
		return
	}
	views := make([]sourceConfigResponse, 0, len(configs))
	for _, config := range configs {
		view, err := sourceConfigView(config)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid stored source settings"})
			return
		}
		views = append(views, view)
	}
	c.JSON(http.StatusOK, gin.H{"data": views, "count": len(views)})
}

func (h *SourceConfigHandler) Update(c *gin.Context) {
	source := strings.ToLower(strings.TrimSpace(c.Param("source")))
	if !types.IsRadarSource(source) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported source"})
		return
	}
	var req updateSourceConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid source config payload"})
		return
	}
	if req.Enabled == nil && req.RedditCommunities == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one setting is required"})
		return
	}
	if source != types.SourceReddit && req.RedditCommunities != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "redditCommunities is only valid for reddit"})
		return
	}

	config, err := h.currentConfig(source)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read current source config"})
		return
	}
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if source == types.SourceReddit && req.RedditCommunities != nil {
		communities := normalizeRedditCommunities(req.RedditCommunities)
		settings, err := json.Marshal(struct {
			Communities []string `json:"communities"`
		}{Communities: communities})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "could not encode source settings"})
			return
		}
		config.SettingsJSON = string(settings)
	}

	updated, err := h.configs.Save(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update source config"})
		return
	}
	view, err := sourceConfigView(*updated)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid stored source settings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": view})
}

func (h *SourceConfigHandler) currentConfig(source string) (store.SourceConfig, error) {
	configs, err := h.configs.List()
	if err != nil {
		return store.SourceConfig{}, err
	}
	for _, config := range configs {
		if config.Source == source {
			return config, nil
		}
	}
	return store.SourceConfig{Source: source, SettingsJSON: "{}"}, nil
}

func normalizeRedditCommunities(communities []string) []string {
	normalized := make([]string, 0, len(communities))
	for _, community := range store.NormalizedAllowlist(communities) {
		community = strings.TrimPrefix(community, "r/")
		if community != "" && community != "all" {
			normalized = append(normalized, "r/"+community)
		}
	}
	return store.NormalizedAllowlist(normalized)
}

func sourceConfigView(config store.SourceConfig) (sourceConfigResponse, error) {
	settings := json.RawMessage(config.SettingsJSON)
	if len(settings) == 0 {
		settings = json.RawMessage(`{}`)
	}
	if !json.Valid(settings) {
		return sourceConfigResponse{}, errors.New("invalid source settings JSON")
	}
	return sourceConfigResponse{
		Source:        config.Source,
		Enabled:       config.Enabled,
		Settings:      settings,
		LastSuccessAt: config.LastSuccessAt,
		LastFailure:   config.LastFailure,
		UpdatedAt:     config.UpdatedAt,
	}, nil
}
