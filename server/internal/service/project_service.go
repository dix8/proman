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
	"proman/server/internal/pkg/projecttoken"
	"proman/server/internal/repository"
)

type ProjectService struct {
	projectRepo *repository.ProjectRepository
}

type ListProjectsInput struct {
	UserID   uint64
	Page     int
	PageSize int
	Keyword  string
}

type CreateProjectInput struct {
	UserID      uint64
	Name        string
	Description string
}

type CreateProjectResult struct {
	Project      *model.Project
	ProjectToken string
}

type ListProjectsResult struct {
	List     []model.Project
	Total    int64
	Page     int
	PageSize int
}

type UpdateProjectInput struct {
	UserID      uint64
	ProjectID   uint64
	Name        string
	Description string
}

type RefreshProjectTokenInput struct {
	UserID    uint64
	ProjectID uint64
}

type RefreshProjectTokenResult struct {
	Project      *model.Project
	ProjectToken string
}

func NewProjectService(projectRepo *repository.ProjectRepository) *ProjectService {
	return &ProjectService{
		projectRepo: projectRepo,
	}
}

func (s *ProjectService) List(ctx context.Context, input ListProjectsInput) (*ListProjectsResult, error) {
	if input.UserID == 0 {
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

	list, total, err := s.projectRepo.ListByUserID(ctx, input.UserID, keyword, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &ListProjectsResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ProjectService) Create(ctx context.Context, input CreateProjectInput) (*CreateProjectResult, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if input.UserID == 0 || input.Name == "" || len([]rune(input.Name)) > 100 || len([]rune(input.Description)) > 1000 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	token, err := projecttoken.Generate()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	now := time.Now().UTC()
	project := &model.Project{
		UserID:         input.UserID,
		Name:           input.Name,
		Description:    input.Description,
		APITokenHash:   projecttoken.Hash(token),
		TokenUpdatedAt: now,
	}

	if err := s.projectRepo.Create(ctx, project); err != nil {
		return nil, apperror.Internal(err)
	}

	return &CreateProjectResult{
		Project:      project,
		ProjectToken: token,
	}, nil
}

func (s *ProjectService) GetByID(ctx context.Context, userID, projectID uint64) (*model.Project, error) {
	if userID == 0 || projectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	project, err := s.projectRepo.FindByIDAndUserID(ctx, projectID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	return project, nil
}

func (s *ProjectService) Update(ctx context.Context, input UpdateProjectInput) (*model.Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Description = strings.TrimSpace(input.Description)

	if input.UserID == 0 || input.ProjectID == 0 || input.Name == "" || len([]rune(input.Name)) > 100 || len([]rune(input.Description)) > 1000 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	project, err := s.projectRepo.UpdateByIDAndUserID(ctx, input.ProjectID, input.UserID, input.Name, input.Description)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	return project, nil
}

func (s *ProjectService) RefreshToken(ctx context.Context, input RefreshProjectTokenInput) (*RefreshProjectTokenResult, error) {
	if input.UserID == 0 || input.ProjectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	token, err := projecttoken.Generate()
	if err != nil {
		return nil, apperror.Internal(err)
	}

	project, err := s.projectRepo.RefreshTokenByIDAndUserID(
		ctx,
		input.ProjectID,
		input.UserID,
		projecttoken.Hash(token),
		time.Now().UTC(),
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	return &RefreshProjectTokenResult{
		Project:      project,
		ProjectToken: token,
	}, nil
}

func (s *ProjectService) Delete(ctx context.Context, userID, projectID uint64) error {
	if userID == 0 || projectID == 0 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if err := s.projectRepo.SoftDeleteCascadeByIDAndUserID(ctx, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return apperror.Internal(err)
	}

	return nil
}

func normalizePage(page, pageSize int) (int, int, error) {
	if page <= 0 || pageSize <= 0 || pageSize > 100 {
		return 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	return page, pageSize, nil
}
