package store

import (
	"time"

	"gorm.io/gorm"
)

// AdminSessionRepo owns persistent session lookup so HTTP handlers never query
// GORM directly.
type AdminSessionRepo struct {
	db *gorm.DB
}

func NewAdminSessionRepo(db *gorm.DB) *AdminSessionRepo {
	return &AdminSessionRepo{db: db}
}

func (r *AdminSessionRepo) Create(tokenHash string, expiresAt time.Time) error {
	return r.db.Create(&AdminSession{TokenHash: tokenHash, ExpiresAt: expiresAt}).Error
}

func (r *AdminSessionRepo) IsActive(tokenHash string, now time.Time) (bool, error) {
	var session AdminSession
	err := r.db.Where("token_hash = ? AND expires_at > ?", tokenHash, now).First(&session).Error
	if err == gorm.ErrRecordNotFound {
		return false, nil
	}
	return err == nil, err
}

func (r *AdminSessionRepo) Delete(tokenHash string) error {
	return r.db.Where("token_hash = ?", tokenHash).Delete(&AdminSession{}).Error
}
