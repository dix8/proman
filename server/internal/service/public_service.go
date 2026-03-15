package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strconv"

	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/repository"
)

var versionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$`)

type PublicService struct {
	projectRepo      *repository.ProjectRepository
	versionRepo      *repository.VersionRepository
	changelogRepo    *repository.ChangelogRepository
	announcementRepo *repository.AnnouncementRepository
}

func NewPublicService(projectRepo *repository.ProjectRepository, versionRepo *repository.VersionRepository, changelogRepo *repository.ChangelogRepository, announcementRepo *repository.AnnouncementRepository) *PublicService {
	return &PublicService{
		projectRepo:      projectRepo,
		versionRepo:      versionRepo,
		changelogRepo:    changelogRepo,
		announcementRepo: announcementRepo,
	}
}

func (s *PublicService) GetProject(ctx context.Context, projectID uint64) (*PublicProjectDTO, error) {
	project, err := s.projectRepo.FindByID(ctx, projectID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusUnauthorized, 40103, "项目 Token 无效")
		}
		return nil, apperror.Internal(err)
	}
	return &PublicProjectDTO{
		Name:           project.Name,
		Description:    project.Description,
		TokenUpdatedAt: project.TokenUpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		CreatedAt:      project.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      project.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}, nil
}

func (s *PublicService) ListVersions(ctx context.Context, projectID uint64, page, pageSize int) (*PublicVersionsResult, error) {
	page, pageSize, err := normalizePage(page, pageSize)
	if err != nil {
		return nil, err
	}

	list, total, err := s.versionRepo.ListPublishedByProjectID(ctx, projectID, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]PublicVersionDTO, 0, len(list))
	for _, version := range list {
		items = append(items, toPublicVersionDTO(version))
	}

	return &PublicVersionsResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *PublicService) GetVersionChangelogs(ctx context.Context, projectID uint64, versionString string) (*PublicVersionChangelogsResult, error) {
	major, minor, patch, err := parseVersionString(versionString)
	if err != nil {
		return nil, err
	}

	version, err := s.versionRepo.FindPublishedByProjectIDAndParts(ctx, projectID, major, minor, patch)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return nil, apperror.Internal(err)
	}

	changelogs, err := s.changelogRepo.ListByVersionID(ctx, version.ID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]PublicChangelogDTO, 0, len(changelogs))
	for _, changelog := range changelogs {
		items = append(items, PublicChangelogDTO{
			Type:      changelog.Type,
			Content:   changelog.Content,
			SortOrder: changelog.SortOrder,
			CreatedAt: changelog.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt: changelog.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	var publishedAt *string
	if version.PublishedAt != nil {
		value := version.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		publishedAt = &value
	}

	return &PublicVersionChangelogsResult{
		Version: PublicVersionSubsetDTO{
			Version:     version.VersionString(),
			Status:      version.Status,
			PublishedAt: publishedAt,
		},
		Changelogs: items,
	}, nil
}

func (s *PublicService) ListAnnouncements(ctx context.Context, projectID uint64, page, pageSize int) (*PublicAnnouncementsResult, error) {
	page, pageSize, err := normalizePage(page, pageSize)
	if err != nil {
		return nil, err
	}

	list, total, err := s.announcementRepo.ListPublishedByProjectID(ctx, projectID, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	items := make([]PublicAnnouncementDTO, 0, len(list))
	for _, announcement := range list {
		var publishedAt *string
		if announcement.PublishedAt != nil {
			value := announcement.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
			publishedAt = &value
		}
		items = append(items, PublicAnnouncementDTO{
			Title:       announcement.Title,
			Content:     announcement.Content,
			IsPinned:    announcement.IsPinned,
			Status:      announcement.Status,
			PublishedAt: publishedAt,
			CreatedAt:   announcement.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:   announcement.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}

	return &PublicAnnouncementsResult{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func toPublicVersionDTO(version model.Version) PublicVersionDTO {
	var publishedAt *string
	if version.PublishedAt != nil {
		value := version.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		publishedAt = &value
	}
	return PublicVersionDTO{
		Major:       version.Major,
		Minor:       version.Minor,
		Patch:       version.Patch,
		URL:         version.URL,
		Version:     version.VersionString(),
		Status:      version.Status,
		PublishedAt: publishedAt,
		CreatedAt:   version.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   version.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func parseVersionString(version string) (uint, uint, uint, error) {
	matches := versionPattern.FindStringSubmatch(version)
	if matches == nil {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	patch, err := strconv.Atoi(matches[3])
	if err != nil {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	return uint(major), uint(minor), uint(patch), nil
}
