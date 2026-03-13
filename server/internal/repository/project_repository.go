package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"

	"proman/server/internal/model"
)

type ProjectRepository struct {
	db *gorm.DB
}

func NewProjectRepository(db *gorm.DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

func (r *ProjectRepository) Create(ctx context.Context, project *model.Project) error {
	return r.db.WithContext(ctx).Create(project).Error
}

func (r *ProjectRepository) ListByUserID(ctx context.Context, userID uint64, keyword string, page, pageSize int) ([]model.Project, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Project{}).
		Where("user_id = ?", userID)

	if trimmedKeyword := strings.TrimSpace(keyword); trimmedKeyword != "" {
		query = query.Where("name LIKE ? ESCAPE '\\\\'", "%"+escapeLikeKeyword(trimmedKeyword)+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var projects []model.Project
	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&projects).Error; err != nil {
		return nil, 0, err
	}

	return projects, total, nil
}

func (r *ProjectRepository) FindByIDAndUserID(ctx context.Context, id, userID uint64) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) FindByID(ctx context.Context, id uint64) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*model.Project, error) {
	var project model.Project
	err := r.db.WithContext(ctx).
		Where("api_token_hash = ?", tokenHash).
		First(&project).Error
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func (r *ProjectRepository) UpdateByIDAndUserID(ctx context.Context, id, userID uint64, name, description string) (*model.Project, error) {
	var project model.Project

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return err
		}

		if err := tx.Model(&project).Updates(map[string]interface{}{
			"name":        name,
			"description": description,
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *ProjectRepository) RefreshTokenByIDAndUserID(ctx context.Context, id, userID uint64, tokenHash string, tokenUpdatedAt time.Time) (*model.Project, error) {
	var project model.Project

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return err
		}

		if err := tx.Model(&project).Updates(map[string]interface{}{
			"api_token_hash":   tokenHash,
			"token_updated_at": tokenUpdatedAt,
		}).Error; err != nil {
			return err
		}

		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &project, nil
}

func (r *ProjectRepository) SoftDeleteCascadeByIDAndUserID(ctx context.Context, id, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var project model.Project
		if err := tx.Where("id = ? AND user_id = ?", id, userID).First(&project).Error; err != nil {
			return err
		}

		versionIDs := tx.Unscoped().
			Model(&model.Version{}).
			Where("project_id = ?", project.ID).
			Select("id")

		if err := tx.Where("version_id IN (?)", versionIDs).Delete(&model.Changelog{}).Error; err != nil {
			return err
		}

		if err := tx.Where("project_id = ?", project.ID).Delete(&model.Version{}).Error; err != nil {
			return err
		}

		if err := tx.Where("project_id = ?", project.ID).Delete(&model.Announcement{}).Error; err != nil {
			return err
		}

		if err := tx.Delete(&project).Error; err != nil {
			return err
		}

		return nil
	})
}

func escapeLikeKeyword(keyword string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"%", "\\%",
		"_", "\\_",
	)
	return replacer.Replace(keyword)
}
