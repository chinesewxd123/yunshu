package model

import "time"

type RegistrationRequestStatus int

const (
	RegistrationPending  RegistrationRequestStatus = 0
	RegistrationApproved RegistrationRequestStatus = 1
	RegistrationRejected RegistrationRequestStatus = 2
)

type RegistrationRequest struct {
	ID        uint                     `json:"id" gorm:"primaryKey"`
	Username  string                   `json:"username" gorm:"size:64;not null;uniqueIndex"`
	Email     string                   `json:"email" gorm:"size:128;not null;uniqueIndex"`
	Nickname  string                   `json:"nickname" gorm:"size:128;not null"`
	Password  string                   `json:"-" gorm:"size:255;not null"`
	Status    RegistrationRequestStatus `json:"status" gorm:"not null;default:0"`
	ReviewerID *uint                    `json:"reviewer_id,omitempty"`
	ReviewComment string                `json:"review_comment,omitempty" gorm:"size:255"`
	ReviewedAt *time.Time               `json:"reviewed_at,omitempty"`
	CreatedAt time.Time                 `json:"created_at"`
	UpdatedAt time.Time                 `json:"updated_at"`
}

func (RegistrationRequest) TableName() string {
	return "registration_requests"
}
