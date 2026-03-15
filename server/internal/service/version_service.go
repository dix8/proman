package service

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/repository"
)

type VersionService struct {
	projectRepo   *repository.ProjectRepository
	versionRepo   *repository.VersionRepository
	changelogRepo *repository.ChangelogRepository
}

type ListVersionsInput struct {
	UserID    uint64
	ProjectID uint64
	Page      int
	PageSize  int
	Status    string
}

type CreateVersionInput struct {
	UserID    uint64
	ProjectID uint64
	Major     *int
	Minor     *int
	Patch     *int
	URL       *string
}

type UpdateVersionInput struct {
	UserID    uint64
	VersionID uint64
	Major     *int
	Minor     *int
	Patch     *int
	URL       *string
}

type ListVersionsResult struct {
	List     []model.Version
	Total    int64
	Page     int
	PageSize int
}

func NewVersionService(projectRepo *repository.ProjectRepository, versionRepo *repository.VersionRepository) *VersionService {
	return &VersionService{
		projectRepo:   projectRepo,
		versionRepo:   versionRepo,
		changelogRepo: nil,
	}
}

func NewVersionServiceWithCompare(projectRepo *repository.ProjectRepository, versionRepo *repository.VersionRepository, changelogRepo *repository.ChangelogRepository) *VersionService {
	return &VersionService{
		projectRepo:   projectRepo,
		versionRepo:   versionRepo,
		changelogRepo: changelogRepo,
	}
}

func (s *VersionService) List(ctx context.Context, input ListVersionsInput) (*ListVersionsResult, error) {
	if input.UserID == 0 || input.ProjectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	page, pageSize, err := normalizePage(input.Page, input.PageSize)
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(input.Status)
	if status != "" && status != model.VersionStatusDraft && status != model.VersionStatusPublished {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if _, err := s.projectRepo.FindByIDAndUserID(ctx, input.ProjectID, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	list, total, err := s.versionRepo.ListByProjectIDAndUserID(ctx, input.ProjectID, input.UserID, status, page, pageSize)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	return &ListVersionsResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *VersionService) GetByID(ctx context.Context, userID, versionID uint64) (*model.Version, error) {
	if userID == 0 || versionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	version, err := s.versionRepo.FindByIDAndUserID(ctx, versionID, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return nil, apperror.Internal(err)
	}

	return version, nil
}

func (s *VersionService) Create(ctx context.Context, input CreateVersionInput) (*model.Version, error) {
	if input.UserID == 0 || input.ProjectID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	major, minor, patch, err := validateSemverParts(input.Major, input.Minor, input.Patch)
	if err != nil {
		return nil, err
	}

	if _, err := s.projectRepo.FindByIDAndUserID(ctx, input.ProjectID, input.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	version := &model.Version{
		ProjectID: input.ProjectID,
		Major:     major,
		Minor:     minor,
		Patch:     patch,
		Status:    model.VersionStatusDraft,
	}

	if input.URL != nil {
		version.URL = *input.URL
	}

	if err := s.versionRepo.Create(ctx, version); err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, apperror.New(http.StatusConflict, 40901, "版本号重复")
		}
		return nil, apperror.Internal(err)
	}

	return version, nil
}

func (s *VersionService) Update(ctx context.Context, input UpdateVersionInput) (*model.Version, error) {
	if input.UserID == 0 || input.VersionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	major, minor, patch, err := validateSemverParts(input.Major, input.Minor, input.Patch)
	if err != nil {
		return nil, err
	}

	updated, err := s.versionRepo.UpdateDraftSemverByIDAndUserID(ctx, input.VersionID, input.UserID, major, minor, patch, input.URL)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) {
			return nil, appErr
		}
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, apperror.New(http.StatusConflict, 40901, "版本号重复")
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
		}
		return nil, apperror.Internal(err)
	}

	return updated, nil
}

func (s *VersionService) Delete(ctx context.Context, userID, versionID uint64) error {
	if userID == 0 || versionID == 0 {
		return apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	if err := s.versionRepo.DeleteDraftByIDAndUserID(ctx, versionID, userID); err != nil {
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

func (s *VersionService) Publish(ctx context.Context, userID, versionID uint64) (*model.Version, error) {
	if userID == 0 || versionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	published, err := s.versionRepo.PublishByIDAndUserID(ctx, versionID, userID, time.Now().UTC())
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

	return published, nil
}

func (s *VersionService) Unpublish(ctx context.Context, userID, versionID uint64) (*model.Version, error) {
	if userID == 0 || versionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}

	unpublished, err := s.versionRepo.UnpublishByIDAndUserID(ctx, versionID, userID)
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

	return unpublished, nil
}

func (s *VersionService) Compare(ctx context.Context, userID, projectID, fromVersionID, toVersionID uint64) (*CompareVersionsResult, error) {
	if userID == 0 || projectID == 0 || fromVersionID == 0 || toVersionID == 0 {
		return nil, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	if s.changelogRepo == nil {
		return nil, apperror.Internal(errors.New("compare requires changelog repository"))
	}

	if _, err := s.projectRepo.FindByIDAndUserID(ctx, projectID, userID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperror.New(http.StatusNotFound, 40401, "项目不存在")
		}
		return nil, apperror.Internal(err)
	}

	requestedIDs := []uint64{fromVersionID, toVersionID}
	candidates, err := s.versionRepo.FindByIDsAndProjectIDAndUserID(ctx, projectID, userID, requestedIDs)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	distinctIDs := slices.Clone(requestedIDs)
	slices.Sort(distinctIDs)
	distinctIDs = slices.Compact(distinctIDs)
	if len(candidates) != len(distinctIDs) {
		return nil, apperror.New(http.StatusNotFound, 40402, "版本不存在")
	}

	versionByID := make(map[uint64]model.Version, len(candidates))
	for _, version := range candidates {
		versionByID[version.ID] = version
		if version.Status != model.VersionStatusPublished {
			return nil, apperror.New(http.StatusConflict, 40906, "对比版本不属于同一项目或状态不满足要求")
		}
	}

	fromVersion := versionByID[fromVersionID]
	toVersion := versionByID[toVersionID]
	if compareSemver(fromVersion, toVersion) > 0 {
		fromVersion, toVersion = toVersion, fromVersion
	}

	allPublished, err := s.versionRepo.ListPublishedByProjectIDAndUserID(ctx, projectID, userID)
	if err != nil {
		return nil, apperror.Internal(err)
	}

	inRange := make([]model.Version, 0, len(allPublished))
	for _, version := range allPublished {
		if compareSemver(version, fromVersion) >= 0 && compareSemver(version, toVersion) <= 0 {
			inRange = append(inRange, version)
		}
	}

	versionIDs := make([]uint64, 0, len(inRange))
	compareVersions := make([]CompareVersionItem, 0, len(inRange))
	for _, version := range inRange {
		versionIDs = append(versionIDs, version.ID)
		compareVersions = append(compareVersions, CompareVersionItem{
			ID:      version.ID,
			Version: version.VersionString(),
			Status:  version.Status,
		})
	}

	grouped := CompareChangelogGroups{
		Added:      []CompareChangelogItem{},
		Changed:    []CompareChangelogItem{},
		Fixed:      []CompareChangelogItem{},
		Improved:   []CompareChangelogItem{},
		Deprecated: []CompareChangelogItem{},
		Removed:    []CompareChangelogItem{},
	}

	if len(versionIDs) > 0 {
		rows, err := s.changelogRepo.ListForCompareByVersionIDsAndProjectIDAndUserID(ctx, versionIDs, projectID, userID)
		if err != nil {
			return nil, apperror.Internal(err)
		}

		for _, row := range rows {
			item := CompareChangelogItem{
				ID:        row.ID,
				VersionID: row.VersionID,
				Version:   row.Version,
				Type:      row.Type,
				Content:   row.Content,
				SortOrder: row.SortOrder,
				CreatedAt: row.CreatedAt,
				UpdatedAt: row.UpdatedAt,
			}
			switch row.Type {
			case model.ChangelogTypeAdded:
				grouped.Added = append(grouped.Added, item)
			case model.ChangelogTypeChanged:
				grouped.Changed = append(grouped.Changed, item)
			case model.ChangelogTypeFixed:
				grouped.Fixed = append(grouped.Fixed, item)
			case model.ChangelogTypeImproved:
				grouped.Improved = append(grouped.Improved, item)
			case model.ChangelogTypeDeprecated:
				grouped.Deprecated = append(grouped.Deprecated, item)
			case model.ChangelogTypeRemoved:
				grouped.Removed = append(grouped.Removed, item)
			}
		}
	}

	return &CompareVersionsResult{
		FromVersion: CompareVersionItem{
			ID:      fromVersion.ID,
			Version: fromVersion.VersionString(),
			Status:  fromVersion.Status,
		},
		ToVersion: CompareVersionItem{
			ID:      toVersion.ID,
			Version: toVersion.VersionString(),
			Status:  toVersion.Status,
		},
		Versions:   compareVersions,
		Changelogs: grouped,
	}, nil
}

func validateSemverParts(major, minor, patch *int) (uint, uint, uint, error) {
	if major == nil || minor == nil || patch == nil {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	if *major < 0 || *minor < 0 || *patch < 0 {
		return 0, 0, 0, apperror.New(http.StatusBadRequest, 40001, "参数错误")
	}
	return uint(*major), uint(*minor), uint(*patch), nil
}

func compareSemver(left, right model.Version) int {
	if left.Major != right.Major {
		if left.Major < right.Major {
			return -1
		}
		return 1
	}
	if left.Minor != right.Minor {
		if left.Minor < right.Minor {
			return -1
		}
		return 1
	}
	if left.Patch != right.Patch {
		if left.Patch < right.Patch {
			return -1
		}
		return 1
	}
	if left.ID < right.ID {
		return -1
	}
	if left.ID > right.ID {
		return 1
	}
	return 0
}
