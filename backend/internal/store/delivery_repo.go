package store

import (
	"time"

	"gorm.io/gorm"
)

type DeliveryRepo struct{ db *gorm.DB }

func NewDeliveryRepo(db *gorm.DB) *DeliveryRepo { return &DeliveryRepo{db: db} }

func (r *DeliveryRepo) Begin(run *DeliveryRun) (bool, error) {
	created := false
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var existing DeliveryRun
		err := tx.Where("idempotency_key = ?", run.IdempotencyKey).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := tx.Create(run).Error; err != nil {
				return err
			}
			created = true
			return nil
		}
		if err != nil {
			return err
		}
		if existing.Status == "sent" || (existing.Status == "running" && existing.CreatedAt.After(time.Now().UTC().Add(-15*time.Minute))) {
			return nil
		}
		run.ID = existing.ID
		created = true
		return tx.Model(&existing).Updates(map[string]any{
			"status": "running", "failure_reason": "", "signal_ids_json": run.SignalIDsJSON,
			"created_at": time.Now().UTC(),
		}).Error
	})
	return created, err
}

func (r *DeliveryRepo) Finish(id int64, status, failure string, sentAt *time.Time) error {
	return r.db.Model(&DeliveryRun{}).Where("id = ?", id).Updates(map[string]any{
		"status": status, "failure_reason": failure, "sent_at": sentAt,
	}).Error
}

func (r *DeliveryRepo) Complete(id int64, signalIDs []int64, sentAt time.Time) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if len(signalIDs) > 0 {
			if err := tx.Model(&Signal{}).Where("id IN ?", signalIDs).Update("last_delivered_at", sentAt).Error; err != nil {
				return err
			}
		}
		return tx.Model(&DeliveryRun{}).Where("id = ?", id).Updates(map[string]any{
			"status": "sent", "failure_reason": "", "sent_at": sentAt,
		}).Error
	})
}

func (r *DeliveryRepo) CountSentSince(kind string, since time.Time) (int, error) {
	var count int64
	err := r.db.Model(&DeliveryRun{}).
		Where("kind = ? AND status = ? AND sent_at >= ?", kind, "sent", since).
		Count(&count).Error
	return int(count), err
}
