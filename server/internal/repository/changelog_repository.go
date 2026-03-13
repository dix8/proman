package repository

import (
	"context"
	"slices"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
)

type ChangelogRepository struct {
	db *gorm.DB
}

type ChangelogReorderItem struct {
	ID        uint64
	SortOrder uint
}

type CompareChangelogRow struct {
	ID        uint64
	VersionID uint64
	Version   string
	Type      string
	Content   string
	SortOrder uint
	CreatedAt string
	UpdatedAt string
}

func NewChangelogRepository(db *gorm.DB) *ChangelogRepository {
	return &ChangelogRepository{db: db}
}

func (r *ChangelogRepository) ListByVersionIDAndUserID(ctx context.Context, versionID, userID uint64, changelogType string, page, pageSize int) ([]model.Changelog, int64, error) {
	query := r.scopedChangelogQuery(ctx, userID).
		Where("changelogs.version_id = ?", versionID)

	if changelogType != "" {
		query = query.Where("changelogs.type = ?", changelogType)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var changelogs []model.Changelog
	if err := query.
		Order("changelogs.sort_order ASC").
		Order("changelogs.created_at ASC").
		Order("changelogs.id ASC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&changelogs).Error; err != nil {
		return nil, 0, err
	}

	return changelogs, total, nil
}

func (r *ChangelogRepository) FindByIDAndUserID(ctx context.Context, changelogID, userID uint64) (*model.Changelog, *model.Version, error) {
	var changelog model.Changelog
	var version model.Version

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := scopedChangelogQuery(tx.WithContext(ctx), userID).
			Where("changelogs.id = ?", changelogID)

		if err := query.First(&changelog).Error; err != nil {
			return err
		}

		if err := scopedVersionQuery(tx.WithContext(ctx), userID).
			Where("versions.id = ?", changelog.VersionID).
			First(&version).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return &changelog, &version, nil
}

func (r *ChangelogRepository) CreateForDraftVersionByIDAndUserID(ctx context.Context, versionID, userID uint64, changelogType, content string) (*model.Changelog, error) {
	var changelog model.Changelog

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := lockVersionForUpdateByIDAndUserID(tx.WithContext(ctx), versionID, userID)
		if err != nil {
			return err
		}
		if version.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}

		var maxSort uint
		if err := tx.Model(&model.Changelog{}).
			Where("version_id = ?", versionID).
			Select("COALESCE(MAX(sort_order), 0)").
			Scan(&maxSort).Error; err != nil {
			return err
		}

		changelog = model.Changelog{
			VersionID: versionID,
			Type:      changelogType,
			Content:   content,
			SortOrder: maxSort + 1,
		}

		if err := tx.Create(&changelog).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &changelog, nil
}

func (r *ChangelogRepository) UpdateForDraftByIDAndUserID(ctx context.Context, changelogID, userID uint64, changelogType, content string) (*model.Changelog, error) {
	var changelog model.Changelog

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := r.lockVersionByChangelogIDAndUserID(tx.WithContext(ctx), changelogID, userID)
		if err != nil {
			return err
		}
		if version.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}

		if err := tx.First(&changelog, changelogID).Error; err != nil {
			return err
		}

		if err := tx.Model(&changelog).Updates(map[string]interface{}{
			"type":    changelogType,
			"content": content,
		}).Error; err != nil {
			return err
		}

		if err := tx.First(&changelog, changelogID).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &changelog, nil
}

func (r *ChangelogRepository) DeleteForDraftByIDAndUserID(ctx context.Context, changelogID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := r.lockVersionByChangelogIDAndUserID(tx.WithContext(ctx), changelogID, userID)
		if err != nil {
			return err
		}
		if version.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}

		if err := tx.Delete(&model.Changelog{}, changelogID).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ChangelogRepository) ListAllByVersionIDAndUserID(ctx context.Context, versionID, userID uint64) ([]model.Changelog, error) {
	var changelogs []model.Changelog
	err := r.scopedChangelogQuery(ctx, userID).
		Where("changelogs.version_id = ?", versionID).
		Order("changelogs.sort_order ASC").
		Order("changelogs.created_at ASC").
		Order("changelogs.id ASC").
		Find(&changelogs).Error
	if err != nil {
		return nil, err
	}
	return changelogs, nil
}

func (r *ChangelogRepository) ListByVersionID(ctx context.Context, versionID uint64) ([]model.Changelog, error) {
	var changelogs []model.Changelog
	err := r.db.WithContext(ctx).
		Model(&model.Changelog{}).
		Where("changelogs.deleted_at IS NULL").
		Where("version_id = ?", versionID).
		Order("changelogs.sort_order ASC").
		Order("changelogs.created_at ASC").
		Order("changelogs.id ASC").
		Find(&changelogs).Error
	if err != nil {
		return nil, err
	}
	return changelogs, nil
}

func (r *ChangelogRepository) ReorderForDraftVersionByIDAndUserID(ctx context.Context, versionID, userID uint64, items []ChangelogReorderItem) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := lockVersionForUpdateByIDAndUserID(tx.WithContext(ctx), versionID, userID)
		if err != nil {
			return err
		}
		if version.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}

		var changelogs []model.Changelog
		if err := scopedChangelogQuery(tx.WithContext(ctx), userID).
			Where("changelogs.version_id = ?", versionID).
			Order("changelogs.sort_order ASC").
			Order("changelogs.created_at ASC").
			Order("changelogs.id ASC").
			Find(&changelogs).Error; err != nil {
			return err
		}

		if len(changelogs) != len(items) {
			return apperror.New(400, 40001, "参数错误")
		}

		expectedIDs := make([]uint64, 0, len(changelogs))
		for _, item := range changelogs {
			expectedIDs = append(expectedIDs, item.ID)
		}
		slices.Sort(expectedIDs)

		seenIDs := make([]uint64, 0, len(items))
		expectedSort := uint(1)
		for _, item := range items {
			if item.ID == 0 || item.SortOrder != expectedSort {
				return apperror.New(400, 40001, "参数错误")
			}
			seenIDs = append(seenIDs, item.ID)
			expectedSort++
		}
		slices.Sort(seenIDs)
		if !slices.Equal(expectedIDs, seenIDs) {
			return apperror.New(400, 40001, "参数错误")
		}

		for _, item := range items {
			if err := tx.Model(&model.Changelog{}).
				Where("id = ? AND version_id = ?", item.ID, versionID).
				Update("sort_order", item.SortOrder).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *ChangelogRepository) ListForCompareByVersionIDsAndProjectIDAndUserID(ctx context.Context, versionIDs []uint64, projectID, userID uint64) ([]CompareChangelogRow, error) {
	var rows []CompareChangelogRow
	err := scopedChangelogQuery(r.db.WithContext(ctx), userID).
		Select([]string{
			"changelogs.id AS id",
			"changelogs.version_id AS version_id",
			"CONCAT(versions.major, '.', versions.minor, '.', versions.patch) AS version",
			"changelogs.type AS type",
			"changelogs.content AS content",
			"changelogs.sort_order AS sort_order",
			"DATE_FORMAT(changelogs.created_at, '%Y-%m-%dT%H:%i:%sZ') AS created_at",
			"DATE_FORMAT(changelogs.updated_at, '%Y-%m-%dT%H:%i:%sZ') AS updated_at",
		}).
		Where("versions.project_id = ?", projectID).
		Where("changelogs.version_id IN ?", versionIDs).
		Order("versions.major ASC").
		Order("versions.minor ASC").
		Order("versions.patch ASC").
		Order("changelogs.sort_order ASC").
		Order("changelogs.created_at ASC").
		Order("changelogs.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *ChangelogRepository) lockVersionByChangelogIDAndUserID(tx *gorm.DB, changelogID, userID uint64) (*model.Version, error) {
	var version model.Version
	err := scopedVersionQuery(tx, userID).
		Joins("JOIN changelogs ON changelogs.version_id = versions.id AND changelogs.deleted_at IS NULL").
		Where("changelogs.id = ?", changelogID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *ChangelogRepository) scopedChangelogQuery(ctx context.Context, userID uint64) *gorm.DB {
	return scopedChangelogQuery(r.db.WithContext(ctx), userID)
}

func scopedChangelogQuery(db *gorm.DB, userID uint64) *gorm.DB {
	return db.Model(&model.Changelog{}).
		Where("changelogs.deleted_at IS NULL").
		Joins("JOIN versions ON versions.id = changelogs.version_id AND versions.deleted_at IS NULL").
		Joins("JOIN projects ON projects.id = versions.project_id AND projects.deleted_at IS NULL").
		Where("projects.user_id = ?", userID)
}
