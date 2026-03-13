package repository

import (
	"context"
	"slices"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
)

type VersionRepository struct {
	db *gorm.DB
}

func NewVersionRepository(db *gorm.DB) *VersionRepository {
	return &VersionRepository{db: db}
}

func (r *VersionRepository) Create(ctx context.Context, version *model.Version) error {
	return r.db.WithContext(ctx).Create(version).Error
}

func (r *VersionRepository) ListByProjectIDAndUserID(ctx context.Context, projectID, userID uint64, status string, page, pageSize int) ([]model.Version, int64, error) {
	query := r.scopedVersionQuery(ctx, userID).
		Where("versions.project_id = ?", projectID)

	if status != "" {
		query = query.Where("versions.status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var versions []model.Version
	if err := query.
		Order("versions.major DESC").
		Order("versions.minor DESC").
		Order("versions.patch DESC").
		Order("versions.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&versions).Error; err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

func (r *VersionRepository) FindByIDAndUserID(ctx context.Context, versionID, userID uint64) (*model.Version, error) {
	var version model.Version
	err := r.scopedVersionQuery(ctx, userID).
		Where("versions.id = ?", versionID).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *VersionRepository) FindByIDAndProjectIDAndUserID(ctx context.Context, versionID, projectID, userID uint64) (*model.Version, error) {
	var version model.Version
	err := r.scopedVersionQuery(ctx, userID).
		Where("versions.project_id = ?", projectID).
		Where("versions.id = ?", versionID).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *VersionRepository) FindByIDsAndProjectIDAndUserID(ctx context.Context, projectID, userID uint64, versionIDs []uint64) ([]model.Version, error) {
	ids := slices.Clone(versionIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	var versions []model.Version
	err := r.scopedVersionQuery(ctx, userID).
		Where("versions.project_id = ?", projectID).
		Where("versions.id IN ?", ids).
		Order("versions.major ASC").
		Order("versions.minor ASC").
		Order("versions.patch ASC").
		Order("versions.id ASC").
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *VersionRepository) ListPublishedByProjectIDAndUserID(ctx context.Context, projectID, userID uint64) ([]model.Version, error) {
	var versions []model.Version
	err := r.scopedVersionQuery(ctx, userID).
		Where("versions.project_id = ?", projectID).
		Where("versions.status = ?", model.VersionStatusPublished).
		Order("versions.major ASC").
		Order("versions.minor ASC").
		Order("versions.patch ASC").
		Order("versions.id ASC").
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *VersionRepository) ListPublishedByProjectID(ctx context.Context, projectID uint64, page, pageSize int) ([]model.Version, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Version{}).
		Where("versions.deleted_at IS NULL").
		Where("versions.project_id = ?", projectID).
		Where("versions.status = ?", model.VersionStatusPublished)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var versions []model.Version
	if err := query.
		Order("versions.major DESC").
		Order("versions.minor DESC").
		Order("versions.patch DESC").
		Order("versions.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&versions).Error; err != nil {
		return nil, 0, err
	}

	return versions, total, nil
}

func (r *VersionRepository) ListByProjectIDAndUserIDForExport(ctx context.Context, projectID, userID uint64) ([]model.Version, error) {
	var versions []model.Version
	err := r.scopedVersionQuery(ctx, userID).
		Where("versions.project_id = ?", projectID).
		Order("versions.major DESC").
		Order("versions.minor DESC").
		Order("versions.patch DESC").
		Order("versions.id DESC").
		Find(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *VersionRepository) FindPublishedByProjectIDAndParts(ctx context.Context, projectID uint64, major, minor, patch uint) (*model.Version, error) {
	var version model.Version
	err := r.db.WithContext(ctx).
		Model(&model.Version{}).
		Where("versions.deleted_at IS NULL").
		Where("project_id = ?", projectID).
		Where("status = ?", model.VersionStatusPublished).
		Where("major = ? AND minor = ? AND patch = ?", major, minor, patch).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *VersionRepository) UpdateDraftSemverByIDAndUserID(ctx context.Context, versionID, userID uint64, major, minor, patch uint) (*model.Version, error) {
	var version model.Version

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedVersion, err := lockVersionForUpdateByIDAndUserID(tx.WithContext(ctx), versionID, userID)
		if err != nil {
			return err
		}
		if lockedVersion.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}
		version = *lockedVersion

		if err := tx.Model(&version).Updates(map[string]interface{}{
			"major": major,
			"minor": minor,
			"patch": patch,
		}).Error; err != nil {
			return err
		}

		if err := scopedVersionQuery(tx.WithContext(ctx), userID).Where("versions.id = ?", versionID).First(&version).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func (r *VersionRepository) DeleteDraftByIDAndUserID(ctx context.Context, versionID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		version, err := lockVersionForUpdateByIDAndUserID(tx.WithContext(ctx), versionID, userID)
		if err != nil {
			return err
		}
		if version.Status != model.VersionStatusDraft {
			return apperror.New(409, 40903, "非草稿版本不可删除")
		}

		if err := tx.Where("version_id = ?", version.ID).Delete(&model.Changelog{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(version).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *VersionRepository) CountChangelogs(ctx context.Context, versionID uint64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Changelog{}).
		Where("version_id = ?", versionID).
		Count(&count).Error
	return count, err
}

func (r *VersionRepository) PublishByIDAndUserID(ctx context.Context, versionID, userID uint64, publishedAt time.Time) (*model.Version, error) {
	var version model.Version

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedVersion, err := lockVersionForUpdateByIDAndUserID(tx.WithContext(ctx), versionID, userID)
		if err != nil {
			return err
		}
		if lockedVersion.Status != model.VersionStatusDraft {
			return apperror.New(409, 40902, "版本已发布，不可编辑")
		}
		version = *lockedVersion

		var count int64
		if err := tx.Model(&model.Changelog{}).
			Where("version_id = ?", version.ID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return apperror.New(409, 40904, "发布版本前至少需要一条日志")
		}

		if err := tx.Model(&version).Updates(map[string]interface{}{
			"status":       model.VersionStatusPublished,
			"published_at": publishedAt,
		}).Error; err != nil {
			return err
		}

		if err := scopedVersionQuery(tx.WithContext(ctx), userID).Where("versions.id = ?", versionID).First(&version).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &version, nil
}

func lockVersionForUpdateByIDAndUserID(tx *gorm.DB, versionID, userID uint64) (*model.Version, error) {
	var version model.Version
	err := scopedVersionQuery(tx, userID).
		Where("versions.id = ?", versionID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&version).Error
	if err != nil {
		return nil, err
	}
	return &version, nil
}

func (r *VersionRepository) scopedVersionQuery(ctx context.Context, userID uint64) *gorm.DB {
	return scopedVersionQuery(r.db.WithContext(ctx), userID)
}

func scopedVersionQuery(db *gorm.DB, userID uint64) *gorm.DB {
	return db.Model(&model.Version{}).
		Where("versions.deleted_at IS NULL").
		Joins("JOIN projects ON projects.id = versions.project_id AND projects.deleted_at IS NULL").
		Where("projects.user_id = ?", userID)
}
