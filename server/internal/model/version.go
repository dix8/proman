package model

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	VersionStatusDraft     = "draft"
	VersionStatusPublished = "published"
)

type Version struct {
	ID          uint64         `gorm:"primaryKey"`
	ProjectID   uint64         `gorm:"column:project_id"`
	Major       uint           `gorm:"column:major"`
	Minor       uint           `gorm:"column:minor"`
	Patch       uint           `gorm:"column:patch"`
	URL         string         `gorm:"column:url"`
	Status      string         `gorm:"column:status"`
	PublishedAt *time.Time     `gorm:"column:published_at"`
	CreatedAt   time.Time      `gorm:"column:created_at"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Version) TableName() string {
	return "versions"
}

func (v Version) VersionString() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}
