package model

import (
	"time"

	"gorm.io/gorm"
)

type Project struct {
	ID             uint64         `gorm:"primaryKey"`
	UserID         uint64         `gorm:"column:user_id"`
	Name           string         `gorm:"column:name"`
	Description    string         `gorm:"column:description"`
	APITokenHash   string         `gorm:"column:api_token_hash"`
	TokenUpdatedAt time.Time      `gorm:"column:token_updated_at"`
	CreatedAt      time.Time      `gorm:"column:created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Project) TableName() string {
	return "projects"
}
