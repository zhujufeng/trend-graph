package store

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"trend-graph/internal/types"
)

type SourceConfigRepo struct {
	db *gorm.DB
}

func NewSourceConfigRepo(db *gorm.DB) *SourceConfigRepo {
	return &SourceConfigRepo{db: db}
}

// EnsureDefaults creates the first-release source controls without replacing
// administrator changes made on earlier starts.
func (r *SourceConfigRepo) EnsureDefaults() error {
	configs := []SourceConfig{
		{Source: types.SourceDEV, Enabled: true, SettingsJSON: "{}"},
		{Source: types.SourceGitHub, Enabled: true, SettingsJSON: "{}"},
		{Source: types.SourceReddit, Enabled: true, SettingsJSON: `{"communities":["r/localllama","r/claudeai","r/claudecode","r/ai_agents","r/cursor","r/chatgptcoding"]}`},
		{Source: types.SourceBluesky, Enabled: true, SettingsJSON: "{}"},
		{Source: types.SourceRSS, Enabled: false, SettingsJSON: `{"feeds":[]}`},
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source IN ?", []string{"waytoagi", "skillsmp"}).Delete(&SourceConfig{}).Error; err != nil {
			return err
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "source"}},
			DoNothing: true,
		}).Create(&configs).Error
	})
}

func (r *SourceConfigRepo) List() ([]SourceConfig, error) {
	var configs []SourceConfig
	err := r.db.Order("source ASC").Find(&configs).Error
	return configs, err
}

func (r *SourceConfigRepo) LatestRuns() (map[string]CollectionRun, error) {
	configs, err := r.List()
	if err != nil {
		return nil, err
	}
	runs := make(map[string]CollectionRun, len(configs))
	// ponytail: four configured sources; use a batched DISTINCT ON query only if this list grows materially.
	for _, config := range configs {
		var run CollectionRun
		if err := r.db.Where("source = ?", config.Source).Order("started_at DESC").First(&run).Error; err == nil {
			runs[config.Source] = run
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
	}
	return runs, nil
}

func (r *SourceConfigRepo) Save(config SourceConfig) (*SourceConfig, error) {
	config.ID = 0
	config.CreatedAt = time.Time{}
	config.UpdatedAt = time.Now().UTC()
	if err := r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "source"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"enabled", "settings_json", "updated_at",
		}),
	}).Create(&config).Error; err != nil {
		return nil, err
	}
	if err := r.db.Where("source = ?", config.Source).First(&config).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

// RecordCollectionRun keeps the append-only run audit and the source's latest
// health status consistent in one transaction.
func (r *SourceConfigRepo) RecordCollectionRun(run CollectionRun) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&run).Error; err != nil {
			return err
		}
		updates := map[string]any{"updated_at": time.Now().UTC()}
		if run.Status == "success" {
			updates["last_success_at"] = run.FinishedAt
			updates["last_failure"] = ""
		} else {
			updates["last_failure"] = run.FailureReason
		}
		return tx.Model(&SourceConfig{}).
			Where("source = ?", run.Source).
			Updates(updates).Error
	})
}
