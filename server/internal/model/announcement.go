package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	AnnouncementStatusDraft     = "draft"
	AnnouncementStatusPublished = "published"
)

type Announcement struct {
	ID          uint64         `gorm:"primaryKey"`
	ProjectID   uint64         `gorm:"column:project_id"`
	Title       string         `gorm:"column:title"`
	Content     string         `gorm:"column:content"`
	IsPinned    bool           `gorm:"column:is_pinned"`
	Status      string         `gorm:"column:status"`
	PublishedAt *time.Time     `gorm:"column:published_at"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Announcement) TableName() string {
	return "announcements"
}
