package model

import "time"

// OperationLog 管理接口操作审计：用户、方法、路径、状态码、耗时与请求/响应体。
type OperationLog struct {
	ID             uint      `json:"id" gorm:"primaryKey;index:idx_oplogs_method_status_id,priority:3;index:idx_oplogs_user_id_id,priority:2;comment:主键ID"`
	CreatedAt      time.Time `json:"created_at" gorm:"index;comment:创建时间"`
	UserID         uint      `json:"user_id" gorm:"index;index:idx_oplogs_user_id_id,priority:1;not null;comment:操作用户ID"`
	Username       string    `json:"username" gorm:"size:64;not null;comment:操作用户名"`
	Nickname       string    `json:"nickname" gorm:"size:128;comment:操作人昵称"`
	IP             string    `json:"ip" gorm:"size:64;comment:请求IP"`
	RequestHeaders string    `json:"request_headers" gorm:"type:text;comment:请求头"`
	Method         string    `json:"method" gorm:"size:16;index;index:idx_oplogs_method_status_id,priority:1;not null;comment:请求方法"`
	Path           string    `json:"path" gorm:"size:512;index;not null;comment:请求路径"`
	StatusCode     int       `json:"status_code" gorm:"index;index:idx_oplogs_method_status_id,priority:2;not null;comment:响应状态码"`
	RequestBody    string    `json:"request_body" gorm:"type:text;comment:请求体"`
	ResponseBody   string    `json:"response_body" gorm:"type:text;comment:响应体"`
	LatencyMs      int64     `json:"latency_ms" gorm:"comment:耗时毫秒"`
}

// TableName 指定 GORM 表名为 operation_logs。
func (OperationLog) TableName() string {
	return "operation_logs"
}
