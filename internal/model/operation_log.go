package model

import "time"

type OperationLog struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	CreatedAt      time.Time `json:"created_at"`
	UserID         uint      `json:"user_id" gorm:"index;not null"`
	Username       string    `json:"username" gorm:"size:64;not null"`
	Nickname       string    `json:"nickname" gorm:"size:128"`
	IP             string    `json:"ip" gorm:"size:64"`
	RequestHeaders string    `json:"request_headers" gorm:"type:text"`
	Method         string    `json:"method" gorm:"size:16;index;not null"`
	Path           string    `json:"path" gorm:"size:512;index;not null"`
	StatusCode     int       `json:"status_code" gorm:"index;not null"`
	RequestBody    string    `json:"request_body" gorm:"type:text"`
	ResponseBody   string    `json:"response_body" gorm:"type:text"`
	LatencyMs      int64     `json:"latency_ms"`
}

func (OperationLog) TableName() string {
	return "operation_logs"
}
