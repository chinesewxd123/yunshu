package model

import "time"

// AlertFiringDelivery 记录「该 fingerprint 至少有一次 firing 成功投递到通道」，
// 作为 Redis 缓存失效后的持久化依据（避免 resolved_no_prior 误判）。
type AlertFiringDelivery struct {
	Fingerprint string    `gorm:"column:fingerprint;primaryKey;size:512"`
	UpdatedAt   time.Time `gorm:"column:updated_at"`
}

func (AlertFiringDelivery) TableName() string {
	return "alert_firing_deliveries"
}
