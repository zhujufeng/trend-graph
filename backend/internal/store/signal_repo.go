package store

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"trend-graph/internal/types"
)

type SignalRepo struct {
	db *gorm.DB
}

type RadarSignal struct {
	Signal   Signal
	Evidence *EvidenceSnapshot
	Analysis *SignalAnalysis
}

func NewSignalRepo(db *gorm.DB) *SignalRepo {
	return &SignalRepo{db: db}
}

// ListRadarSignals returns the small dashboard working set with the newest
// evidence and optional analysis for each signal.
func (r *SignalRepo) ListRadarSignals(limit int) ([]RadarSignal, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	var signals []Signal
	if err := activeRadarSignals(r.db).
		Order("CASE qualification WHEN 'qualified' THEN 0 WHEN 'pending' THEN 1 ELSE 2 END").
		Order("score DESC").
		Order("created_at DESC").
		Limit(limit).
		Find(&signals).Error; err != nil {
		return nil, err
	}

	// ponytail: this is at most 2*limit small queries for a personal dashboard;
	// replace with batched lookups only if measurements show it matters.
	result := make([]RadarSignal, 0, len(signals))
	for _, signal := range signals {
		item, err := r.loadRadarSignal(signal, false)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func (r *SignalRepo) GetRadarSignal(id int64) (RadarSignal, error) {
	var signal Signal
	if err := activeRadarSignals(r.db).First(&signal, id).Error; err != nil {
		return RadarSignal{}, err
	}
	return r.loadRadarSignal(signal, true)
}

func (r *SignalRepo) loadRadarSignal(signal Signal, includeEvidenceBody bool) (RadarSignal, error) {
	item := RadarSignal{Signal: signal}
	var evidence EvidenceSnapshot
	evidenceQuery := r.db.Where("signal_id = ?", signal.ID).Order("captured_at DESC")
	if !includeEvidenceBody {
		evidenceQuery = evidenceQuery.Select("id", "signal_id", "source_url", "evidence_class", "title", "captured_at")
	}
	if err := evidenceQuery.First(&evidence).Error; err == nil {
		item.Evidence = &evidence
	} else if err != gorm.ErrRecordNotFound {
		return RadarSignal{}, err
	}
	var analysis SignalAnalysis
	if err := r.db.Where("signal_id = ?", signal.ID).First(&analysis).Error; err == nil {
		item.Analysis = &analysis
	} else if err != gorm.ErrRecordNotFound {
		return RadarSignal{}, err
	}
	return item, nil
}

func (r *SignalRepo) CountAnalysesSince(since time.Time) (int, error) {
	var count int64
	err := r.db.Model(&SignalAnalysis{}).Where("created_at >= ?", since).Count(&count).Error
	return int(count), err
}

func (r *SignalRepo) ListPendingSignals(limit int) ([]RadarSignal, error) {
	var signals []Signal
	if err := activeRadarSignals(r.db).Where("qualification = ?", "pending").Order("score DESC").Order("created_at ASC").Limit(limit).Find(&signals).Error; err != nil {
		return nil, err
	}
	result := make([]RadarSignal, 0, len(signals))
	for _, signal := range signals {
		item := RadarSignal{Signal: signal}
		var evidence EvidenceSnapshot
		if err := r.db.Where("signal_id = ?", signal.ID).Order("captured_at DESC").First(&evidence).Error; err == nil {
			item.Evidence = &evidence
		} else if err != gorm.ErrRecordNotFound {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func activeRadarSignals(db *gorm.DB) *gorm.DB {
	return db.Where("source IN ?", types.RadarSources())
}

func (r *SignalRepo) SetQualification(id int64, qualification, reason string) error {
	return r.db.Model(&Signal{}).Where("id = ?", id).Updates(map[string]any{
		"qualification": qualification, "qualification_reason": reason,
	}).Error
}

func (r *SignalRepo) UpdateLifecycleState(id int64, state string) error {
	result := activeRadarSignals(r.db).Model(&Signal{}).
		Where("id = ? AND qualification = ?", id, "qualified").
		Update("lifecycle_state", state)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *SignalRepo) SaveQualifiedAnalysis(analysis SignalAnalysis, reason string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&analysis).Error; err != nil {
			return err
		}
		return tx.Model(&Signal{}).Where("id = ?", analysis.SignalID).Updates(map[string]any{
			"qualification": "qualified", "qualification_reason": reason,
		}).Error
	})
}

func (r *SignalRepo) CreateContentPackage(content *ContentPackage) error {
	return r.db.Create(content).Error
}

func (r *SignalRepo) GetContentPackage(id int64) (ContentPackage, error) {
	var content ContentPackage
	err := r.db.First(&content, id).Error
	return content, err
}

func (r *SignalRepo) GetEvidenceSnapshot(id int64) (EvidenceSnapshot, error) {
	var evidence EvidenceSnapshot
	err := r.db.First(&evidence, id).Error
	return evidence, err
}

func (r *SignalRepo) UpdateContentPackage(content ContentPackage) error {
	result := r.db.Model(&ContentPackage{}).
		Where("id = ? AND status <> ?", content.ID, "approved").
		Updates(map[string]any{
			"strategy_json": content.StrategyJSON, "xiaohongshu_json": content.XiaohongshuJSON,
			"wechat_json": content.WechatJSON, "x_json": content.XJSON,
			"visual_plan_json": content.VisualPlanJSON,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *SignalRepo) ApproveContentPackage(id int64, now time.Time) error {
	result := r.db.Model(&ContentPackage{}).Where("id = ?", id).Updates(map[string]any{
		"status": "approved", "approved_at": now,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CanonicalURL removes fragments and known tracking parameters so the same
// source article cannot become several signals.
func CanonicalURL(raw string) (string, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""
	query := u.Query()
	for key := range query {
		lower := strings.ToLower(key)
		if strings.HasPrefix(lower, "utm_") || lower == "fbclid" || lower == "gclid" {
			query.Del(key)
		}
	}
	u.RawQuery = query.Encode()
	if u.Path != "/" {
		u.Path = strings.TrimRight(u.Path, "/")
	}
	return u.String(), nil
}

// CreateIfNew is idempotent at both the application and database layers.
func (r *SignalRepo) CreateIfNew(signal Signal) (bool, error) {
	canonical, err := CanonicalURL(signal.OriginalURL)
	if err != nil {
		return false, err
	}
	signal.CanonicalURL = canonical
	if signal.OriginalURL == "" {
		signal.OriginalURL = canonical
	}
	result := r.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "source"}, {Name: "canonical_url"}},
		DoNothing: true,
	}).Create(&signal)
	return result.RowsAffected == 1, result.Error
}

// IngestIfNew stores the signal and the exact source material used for later
// analysis in one transaction. Duplicate signals intentionally do not create
// another snapshot or consume future model quota.
func (r *SignalRepo) IngestIfNew(signal Signal, evidence EvidenceSnapshot) (bool, error) {
	canonical, err := CanonicalURL(signal.OriginalURL)
	if err != nil {
		return false, err
	}
	signal.CanonicalURL = canonical
	if signal.OriginalURL == "" {
		signal.OriginalURL = canonical
	}
	if evidence.ContentHash == "" {
		sum := sha256.Sum256([]byte(evidence.Excerpt))
		evidence.ContentHash = hex.EncodeToString(sum[:])
	}
	if evidence.CapturedAt.IsZero() {
		evidence.CapturedAt = time.Now().UTC()
	}
	created := false
	err = r.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "source"}, {Name: "canonical_url"}},
			DoNothing: true,
		}).Create(&signal)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		created = true
		evidence.SignalID = signal.ID
		return tx.Create(&evidence).Error
	})
	return created, err
}

// NormalizedAllowlist makes source configuration comparisons deterministic.
func NormalizedAllowlist(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
