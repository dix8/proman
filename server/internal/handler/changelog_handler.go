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
	"proman/server/internal/repository"
	"proman/server/internal/service"
)

type ChangelogHandler struct {
	changelogService *service.ChangelogService
}

type changelogRequest struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

type changelogReorderRequest struct {
	Items []struct {
		ID        uint64 `json:"id"`
		SortOrder uint   `json:"sort_order"`
	} `json:"items"`
}

type changelogData struct {
	ID        uint64 `json:"id"`
	VersionID uint64 `json:"version_id"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	SortOrder uint   `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func NewChangelogHandler(changelogService *service.ChangelogService) *ChangelogHandler {
	return &ChangelogHandler{changelogService: changelogService}
}

func (h *ChangelogHandler) List(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
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

	result, err := h.changelogService.List(c.Request.Context(), service.ListChangelogsInput{
		UserID:        middleware.CurrentUserID(c),
		VersionID:     versionID,
		Page:          page,
		PageSize:      pageSize,
		ChangelogType: strings.TrimSpace(c.Query("type")),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	list := make([]changelogData, 0, len(result.List))
	for _, changelog := range result.List {
		changelogCopy := changelog
		list = append(list, newChangelogData(&changelogCopy))
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *ChangelogHandler) Create(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req changelogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	changelog, err := h.changelogService.Create(c.Request.Context(), service.CreateChangelogInput{
		UserID:        middleware.CurrentUserID(c),
		VersionID:     versionID,
		ChangelogType: strings.TrimSpace(req.Type),
		Content:       strings.TrimSpace(req.Content),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newChangelogData(changelog))
}

func (h *ChangelogHandler) Update(c *gin.Context) {
	changelogID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || changelogID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req changelogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	changelog, err := h.changelogService.Update(c.Request.Context(), service.UpdateChangelogInput{
		UserID:        middleware.CurrentUserID(c),
		ChangelogID:   changelogID,
		ChangelogType: strings.TrimSpace(req.Type),
		Content:       strings.TrimSpace(req.Content),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newChangelogData(changelog))
}

func (h *ChangelogHandler) Delete(c *gin.Context) {
	changelogID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || changelogID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	if err := h.changelogService.Delete(c.Request.Context(), middleware.CurrentUserID(c), changelogID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{})
}

func (h *ChangelogHandler) Reorder(c *gin.Context) {
	versionID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || versionID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req changelogReorderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	items := make([]repository.ChangelogReorderItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, repository.ChangelogReorderItem{
			ID:        item.ID,
			SortOrder: item.SortOrder,
		})
	}

	if err := h.changelogService.Reorder(c.Request.Context(), service.ReorderChangelogsInput{
		UserID:    middleware.CurrentUserID(c),
		VersionID: versionID,
		Items:     items,
	}); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{})
}

func newChangelogData(changelog *model.Changelog) changelogData {
	return changelogData{
		ID:        changelog.ID,
		VersionID: changelog.VersionID,
		Type:      changelog.Type,
		Content:   changelog.Content,
		SortOrder: changelog.SortOrder,
		CreatedAt: changelog.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt: changelog.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
