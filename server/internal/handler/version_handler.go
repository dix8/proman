package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/middleware"
	"proman/server/internal/model"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/response"
	"proman/server/internal/service"
)

type VersionHandler struct {
	versionService *service.VersionService
}

type versionRequest struct {
	Major *int `json:"major"`
	Minor *int `json:"minor"`
	Patch *int `json:"patch"`
}

type versionData struct {
	ID          uint64  `json:"id"`
	Major       uint    `json:"major"`
	Minor       uint    `json:"minor"`
	Patch       uint    `json:"patch"`
	Version     string  `json:"version"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func NewVersionHandler(versionService *service.VersionService) *VersionHandler {
	return &VersionHandler{versionService: versionService}
}

func (h *VersionHandler) List(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	page, err := parseIntQueryWithDefault(c.Query("page"), 1)
	if err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}
	pageSize, err := parseIntQueryWithDefault(c.Query("page_size"), 20)
	if err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	result, err := h.versionService.List(c.Request.Context(), service.ListVersionsInput{
		UserID:    middleware.CurrentUserID(c),
		ProjectID: projectID,
		Page:      page,
		PageSize:  pageSize,
		Status:    strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	list := make([]versionData, 0, len(result.List))
	for _, version := range result.List {
		versionCopy := version
		list = append(list, newVersionData(&versionCopy))
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *VersionHandler) Get(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	version, err := h.versionService.GetByID(c.Request.Context(), middleware.CurrentUserID(c), versionID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newVersionData(version))
}

func (h *VersionHandler) Create(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req versionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	version, err := h.versionService.Create(c.Request.Context(), service.CreateVersionInput{
		UserID:    middleware.CurrentUserID(c),
		ProjectID: projectID,
		Major:     req.Major,
		Minor:     req.Minor,
		Patch:     req.Patch,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newVersionData(version))
}

func (h *VersionHandler) Update(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req versionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	version, err := h.versionService.Update(c.Request.Context(), service.UpdateVersionInput{
		UserID:    middleware.CurrentUserID(c),
		VersionID: versionID,
		Major:     req.Major,
		Minor:     req.Minor,
		Patch:     req.Patch,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newVersionData(version))
}

func (h *VersionHandler) Delete(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	if err := h.versionService.Delete(c.Request.Context(), middleware.CurrentUserID(c), versionID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{})
}

func (h *VersionHandler) Publish(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	version, err := h.versionService.Publish(c.Request.Context(), middleware.CurrentUserID(c), versionID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newVersionData(version))
}

func (h *VersionHandler) Compare(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	fromVersionID, err := strconv.ParseUint(strings.TrimSpace(c.Query("from_version_id")), 10, 64)
	if err != nil || fromVersionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	toVersionID, err := strconv.ParseUint(strings.TrimSpace(c.Query("to_version_id")), 10, 64)
	if err != nil || toVersionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	result, err := h.versionService.Compare(c.Request.Context(), middleware.CurrentUserID(c), projectID, fromVersionID, toVersionID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, result)
}

func newVersionData(version *model.Version) versionData {
	var publishedAt *string
	if version.PublishedAt != nil {
		value := version.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		publishedAt = &value
	}

	return versionData{
		ID:          version.ID,
		Major:       version.Major,
		Minor:       version.Minor,
		Patch:       version.Patch,
		Version:     version.VersionString(),
		Status:      version.Status,
		PublishedAt: publishedAt,
		CreatedAt:   version.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   version.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
