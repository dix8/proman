package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	ChangelogTypeAdded      = "added"
	ChangelogTypeChanged    = "changed"
	ChangelogTypeFixed      = "fixed"
	ChangelogTypeImproved   = "improved"
	ChangelogTypeDeprecated = "deprecated"
	ChangelogTypeRemoved    = "removed"
)

type Changelog struct {
	ID        uint64         `gorm:"primaryKey"`
	VersionID uint64         `gorm:"column:version_id"`
	Type      string         `gorm:"column:type"`
	Content   string         `gorm:"column:content"`
	SortOrder uint           `gorm:"column:sort_order"`
	CreatedAt time.Time      `gorm:"column:created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at"`
}

func (Changelog) TableName() string {
	return "changelogs"
}
