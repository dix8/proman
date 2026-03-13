package service

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/repository"
)

type AnnouncementService struct {
	projectRepo      *repository.ProjectRepository
	announcementRepo *repository.AnnouncementRepository
}

type ListAnnouncementsInput struct {
	UserID    uint64
	ProjectID uint64
	Page      int
	PageSize  int
	Keyword   string
	Status    string
}

type CreateAnnouncementInput struct {
	UserID    uint64
	ProjectID uint64
	Title     string
	Content   string
	IsPinned  bool
}

type UpdateAnnouncementInput struct {
	UserID         uint64
	AnnouncementID uint64
	Title          string
	Content        string
	IsPinned       bool
}

type ListAnnouncementsResult struct {
	List     []model.Announcement
	Total    int64
	Page     int
	PageSize int
}

func NewAnnouncementService(projectRepo *repository.ProjectRepository, announcementRepo *repository.AnnouncementRepository) *AnnouncementService {
	return &AnnouncementService{
		projectRepo:      projectRepo,
		announcementRepo: announcementRepo,
	}
}

func (s *AnnouncementService) List(ctx context.Context, input ListAnnouncementsInput) (*ListAnnouncementsResult, error) {
	if input.UserID == 0 || input.ProjectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	page, pageSize, err := normalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}

	keyword := strings.TrimSpace(input.Keyword)
	if len([]rune(keyword)) > 100 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	status := strings.TrimSpace(input.Status)
	if status != "" && status != model.AnnouncementStatusDraft && status != model.AnnouncementStatusPublished {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if _, err := s.projectRepo.FindByIDAndUserID(ctx, input.ProjectID, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	list, total, err := s.announcementRepo.ListByProjectIDAndUserID(ctx, input.ProjectID, input.UserID, keyword, status, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &ListAnnouncementsResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AnnouncementService) GetByID(ctx context.Context, userID, announcementID uint64) (*model.Announcement, error) {
	if userID == 0 || announcementID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	announcement, err := s.announcementRepo.FindByIDAndUserID(ctx, announcementID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40404, "公告不存在")
		}
		return nil, apperror.Internal(err)
	}

	return announcement, nil
}

func (s *AnnouncementService) Create(ctx context.Context, input CreateAnnouncementInput) (*model.Announcement, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)

	if input.UserID == 0 || input.ProjectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	if err := validateAnnouncementFields(input.Title, input.Content); err != nil {
		return nil, err
	}

	if _, err := s.projectRepo.FindByIDAndUserID(ctx, input.ProjectID, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	announcement := &model.Announcement{
		ProjectID: input.ProjectID,
		Title:     input.Title,
		Content:   input.Content,
		IsPinned:  input.IsPinned,
		Status:    model.AnnouncementStatusDraft,
	}

	if err := s.announcementRepo.Create(ctx, announcement); err != nil {
		return nil, apperror.Internal(err)
	}
	return announcement, nil
}

func (s *AnnouncementService) Update(ctx context.Context, input UpdateAnnouncementInput) (*model.Announcement, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Content = strings.TrimSpace(input.Content)

	if input.UserID == 0 || input.AnnouncementID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	if err := validateAnnouncementFields(input.Title, input.Content); err != nil {
		return nil, err
	}

	announcement, err := s.announcementRepo.UpdateByIDAndUserID(ctx, input.AnnouncementID, input.UserID, input.Title, input.Content, input.IsPinned)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40404, "公告不存在")
		}
		return nil, apperror.Internal(err)
	}
	return announcement, nil
}

func (s *AnnouncementService) Delete(ctx context.Context, userID, announcementID uint64) error {
	if userID == 0 || announcementID == 0 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if err := s.announcementRepo.DeleteByIDAndUserID(ctx, announcementID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(http.StatusNotFound, 40404, "公告不存在")
		}
		return apperror.Internal(err)
	}
	return nil
}

func (s *AnnouncementService) Publish(ctx context.Context, userID, announcementID uint64) (*model.Announcement, error) {
	if userID == 0 || announcementID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	announcement, err := s.announcementRepo.PublishByIDAndUserID(ctx, announcementID, userID, time.Now().UTC())
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40404, "公告不存在")
		}
		return nil, apperror.Internal(err)
	}
	return announcement, nil
}

func (s *AnnouncementService) Revoke(ctx context.Context, userID, announcementID uint64) (*model.Announcement, error) {
	if userID == 0 || announcementID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	announcement, err := s.announcementRepo.RevokeByIDAndUserID(ctx, announcementID, userID)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40404, "公告不存在")
		}
		return nil, apperror.Internal(err)
	}
	return announcement, nil
}

func validateAnnouncementFields(title, content string) error {
	if title == "" || len([]rune(title)) > 150 || content == "" || len([]rune(content)) > 20000 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	return nil
}
