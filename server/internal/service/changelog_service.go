package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/repository"
)

type ChangelogService struct {
	changelogRepo *repository.ChangelogRepository
	versionRepo   *repository.VersionRepository
}

type ListChangelogsInput struct {
	UserID        uint64
	VersionID     uint64
	Page          int
	PageSize      int
	ChangelogType string
}

type ListChangelogsResult struct {
	List     []model.Changelog
	Total    int64
	Page     int
	PageSize int
}

type CreateChangelogInput struct {
	UserID        uint64
	VersionID     uint64
	ChangelogType string
	Content       string
}

type UpdateChangelogInput struct {
	UserID        uint64
	ChangelogID   uint64
	ChangelogType string
	Content       string
}

type ReorderChangelogsInput struct {
	UserID    uint64
	VersionID uint64
	Items     []repository.ChangelogReorderItem
}

func NewChangelogService(changelogRepo *repository.ChangelogRepository, versionRepo *repository.VersionRepository) *ChangelogService {
	return &ChangelogService{
		changelogRepo: changelogRepo,
		versionRepo:   versionRepo,
	}
}

func (s *ChangelogService) List(ctx context.Context, input ListChangelogsInput) (*ListChangelogsResult, error) {
	if input.UserID == 0 || input.VersionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	page, pageSize, err := normalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}

	changelogType := strings.TrimSpace(input.ChangelogType)
	if changelogType != "" && !isValidChangelogType(changelogType) {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if _, err := s.versionRepo.FindByIDAndUserID(ctx, input.VersionID, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return nil, apperror.Internal(err)
	}

	list, total, err := s.changelogRepo.ListByVersionIDAndUserID(ctx, input.VersionID, input.UserID, changelogType, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &ListChangelogsResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ChangelogService) Create(ctx context.Context, input CreateChangelogInput) (*model.Changelog, error) {
	if input.UserID == 0 || input.VersionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	changelogType := strings.TrimSpace(input.ChangelogType)
	content := strings.TrimSpace(input.Content)
	if !isValidChangelogType(changelogType) || content == "" || len([]rune(content)) > 20000 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	changelog, err := s.changelogRepo.CreateForDraftVersionByIDAndUserID(ctx, input.VersionID, input.UserID, changelogType, content)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return nil, apperror.Internal(err)
	}

	return changelog, nil
}

func (s *ChangelogService) Update(ctx context.Context, input UpdateChangelogInput) (*model.Changelog, error) {
	if input.UserID == 0 || input.ChangelogID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	changelogType := strings.TrimSpace(input.ChangelogType)
	content := strings.TrimSpace(input.Content)
	if !isValidChangelogType(changelogType) || content == "" || len([]rune(content)) > 20000 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	updated, err := s.changelogRepo.UpdateForDraftByIDAndUserID(ctx, input.ChangelogID, input.UserID, changelogType, content)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40403, "更新日志不存在")
		}
		return nil, apperror.Internal(err)
	}
	return updated, nil
}

func (s *ChangelogService) Delete(ctx context.Context, userID, changelogID uint64) error {
	if userID == 0 || changelogID == 0 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if err := s.changelogRepo.DeleteForDraftByIDAndUserID(ctx, changelogID, userID); err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(http.StatusNotFound, 40403, "更新日志不存在")
		}
		return apperror.Internal(err)
	}
	return nil
}

func (s *ChangelogService) Reorder(ctx context.Context, input ReorderChangelogsInput) error {
	if input.UserID == 0 || input.VersionID == 0 || len(input.Items) == 0 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if err := s.changelogRepo.ReorderForDraftVersionByIDAndUserID(ctx, input.VersionID, input.UserID, input.Items); err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return appErr
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return apperror.Internal(err)
	}
	return nil
}

func isValidChangelogType(changelogType string) bool {
	switch changelogType {
	case model.ChangelogTypeAdded,
		model.ChangelogTypeChanged,
		model.ChangelogTypeFixed,
		model.ChangelogTypeImproved,
		model.ChangelogTypeDeprecated,
		model.ChangelogTypeRemoved:
		return true
	default:
		return false
	}
}
