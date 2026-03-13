package repository

import (
	"context"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
)

type AnnouncementRepository struct {
	db *gorm.DB
}

func NewAnnouncementRepository(db *gorm.DB) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

func (r *AnnouncementRepository) Create(ctx context.Context, announcement *model.Announcement) error {
	return r.db.WithContext(ctx).Create(announcement).Error
}

func (r *AnnouncementRepository) ListByProjectIDAndUserID(ctx context.Context, projectID, userID uint64, keyword, status string, page, pageSize int) ([]model.Announcement, int64, error) {
	query := scopedAnnouncementQuery(r.db.WithContext(ctx), userID).
		Where("announcements.project_id = ?", projectID)

	if trimmedKeyword := strings.TrimSpace(keyword); trimmedKeyword != "" {
		query = query.Where("announcements.title LIKE ? ESCAPE '\\\\'", "%"+escapeLikeKeyword(trimmedKeyword)+"%")
	}
	if status != "" {
		query = query.Where("announcements.status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var announcements []model.Announcement
	if err := query.
		Order("announcements.is_pinned DESC").
		Order("announcements.published_at DESC").
		Order("announcements.created_at DESC").
		Order("announcements.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&announcements).Error; err != nil {
		return nil, 0, err
	}

	return announcements, total, nil
}

func (r *AnnouncementRepository) ListPublishedByProjectID(ctx context.Context, projectID uint64, page, pageSize int) ([]model.Announcement, int64, error) {
	query := r.db.WithContext(ctx).
		Model(&model.Announcement{}).
		Where("announcements.deleted_at IS NULL").
		Where("project_id = ?", projectID).
		Where("status = ?", model.AnnouncementStatusPublished)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var announcements []model.Announcement
	if err := query.
		Order("announcements.is_pinned DESC").
		Order("announcements.published_at DESC").
		Order("announcements.created_at DESC").
		Order("announcements.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&announcements).Error; err != nil {
		return nil, 0, err
	}

	return announcements, total, nil
}

func (r *AnnouncementRepository) FindByIDAndUserID(ctx context.Context, announcementID, userID uint64) (*model.Announcement, error) {
	var announcement model.Announcement
	err := scopedAnnouncementQuery(r.db.WithContext(ctx), userID).
		Where("announcements.id = ?", announcementID).
		First(&announcement).Error
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

func (r *AnnouncementRepository) UpdateByIDAndUserID(ctx context.Context, announcementID, userID uint64, title, content string, isPinned bool) (*model.Announcement, error) {
	var announcement model.Announcement
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedAnnouncement, err := lockAnnouncementForUpdateByIDAndUserID(tx.WithContext(ctx), announcementID, userID)
		if err != nil {
			return err
		}
		announcement = *lockedAnnouncement

		if err := tx.Model(&announcement).Updates(map[string]interface{}{
			"title":     title,
			"content":   content,
			"is_pinned": isPinned,
		}).Error; err != nil {
			return err
		}

		if err := scopedAnnouncementQuery(tx.WithContext(ctx), userID).
			Where("announcements.id = ?", announcementID).
			First(&announcement).Error; err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

func (r *AnnouncementRepository) PublishByIDAndUserID(ctx context.Context, announcementID, userID uint64, publishedAt time.Time) (*model.Announcement, error) {
	var announcement model.Announcement
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedAnnouncement, err := lockAnnouncementForUpdateByIDAndUserID(tx.WithContext(ctx), announcementID, userID)
		if err != nil {
			return err
		}
		if lockedAnnouncement.Status != model.AnnouncementStatusDraft {
			return apperror.New(409, 40905, "公告状态流转非法")
		}
		announcement = *lockedAnnouncement

		if err := tx.Model(&announcement).Updates(map[string]interface{}{
			"status":       model.AnnouncementStatusPublished,
			"published_at": publishedAt,
		}).Error; err != nil {
			return err
		}

		if err := scopedAnnouncementQuery(tx.WithContext(ctx), userID).
			Where("announcements.id = ?", announcementID).
			First(&announcement).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

func (r *AnnouncementRepository) RevokeByIDAndUserID(ctx context.Context, announcementID, userID uint64) (*model.Announcement, error) {
	var announcement model.Announcement
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		lockedAnnouncement, err := lockAnnouncementForUpdateByIDAndUserID(tx.WithContext(ctx), announcementID, userID)
		if err != nil {
			return err
		}
		if lockedAnnouncement.Status != model.AnnouncementStatusPublished {
			return apperror.New(409, 40905, "公告状态流转非法")
		}
		announcement = *lockedAnnouncement

		if err := tx.Model(&announcement).Updates(map[string]interface{}{
			"status":       model.AnnouncementStatusDraft,
			"published_at": nil,
		}).Error; err != nil {
			return err
		}

		if err := scopedAnnouncementQuery(tx.WithContext(ctx), userID).
			Where("announcements.id = ?", announcementID).
			First(&announcement).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}

func (r *AnnouncementRepository) DeleteByIDAndUserID(ctx context.Context, announcementID, userID uint64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		announcement, err := lockAnnouncementForUpdateByIDAndUserID(tx.WithContext(ctx), announcementID, userID)
		if err != nil {
			return err
		}

		if err := tx.Delete(announcement).Error; err != nil {
			return err
		}
		return nil
	})
}

func scopedAnnouncementQuery(db *gorm.DB, userID uint64) *gorm.DB {
	return db.Model(&model.Announcement{}).
		Where("announcements.deleted_at IS NULL").
		Joins("JOIN projects ON projects.id = announcements.project_id AND projects.deleted_at IS NULL").
		Where("projects.user_id = ?", userID)
}

func lockAnnouncementForUpdateByIDAndUserID(tx *gorm.DB, announcementID, userID uint64) (*model.Announcement, error) {
	var announcement model.Announcement
	err := scopedAnnouncementQuery(tx, userID).
		Where("announcements.id = ?", announcementID).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&announcement).Error
	if err != nil {
		return nil, err
	}
	return &announcement, nil
}
