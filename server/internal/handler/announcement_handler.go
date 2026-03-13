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

type AnnouncementHandler struct {
	announcementService *service.AnnouncementService
}

type announcementRequest struct {
	Title    string `json:"title"`
	Content  string `json:"content"`
	IsPinned bool   `json:"is_pinned"`
}

type announcementData struct {
	ID          uint64  `json:"id"`
	ProjectID   uint64  `json:"project_id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPinned    bool    `json:"is_pinned"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func NewAnnouncementHandler(announcementService *service.AnnouncementService) *AnnouncementHandler {
	return &AnnouncementHandler{announcementService: announcementService}
}

func (h *AnnouncementHandler) List(c *gin.Context) {
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

	result, err := h.announcementService.List(c.Request.Context(), service.ListAnnouncementsInput{
		UserID:    middleware.CurrentUserID(c),
		ProjectID: projectID,
		Page:      page,
		PageSize:  pageSize,
		Keyword:   strings.TrimSpace(c.Query("keyword")),
		Status:    strings.TrimSpace(c.Query("status")),
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	list := make([]announcementData, 0, len(result.List))
	for _, announcement := range result.List {
		announcementCopy := announcement
		list = append(list, newAnnouncementData(&announcementCopy))
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}

func (h *AnnouncementHandler) Get(c *gin.Context) {
	announcementID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || announcementID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	announcement, err := h.announcementService.GetByID(c.Request.Context(), middleware.CurrentUserID(c), announcementID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newAnnouncementData(announcement))
}

func (h *AnnouncementHandler) Create(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req announcementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	announcement, err := h.announcementService.Create(c.Request.Context(), service.CreateAnnouncementInput{
		UserID:    middleware.CurrentUserID(c),
		ProjectID: projectID,
		Title:     req.Title,
		Content:   req.Content,
		IsPinned:  req.IsPinned,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newAnnouncementData(announcement))
}

func (h *AnnouncementHandler) Update(c *gin.Context) {
	announcementID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || announcementID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	var req announcementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	announcement, err := h.announcementService.Update(c.Request.Context(), service.UpdateAnnouncementInput{
		UserID:         middleware.CurrentUserID(c),
		AnnouncementID: announcementID,
		Title:          req.Title,
		Content:        req.Content,
		IsPinned:       req.IsPinned,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newAnnouncementData(announcement))
}

func (h *AnnouncementHandler) Delete(c *gin.Context) {
	announcementID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || announcementID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	if err := h.announcementService.Delete(c.Request.Context(), middleware.CurrentUserID(c), announcementID); err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{})
}

func (h *AnnouncementHandler) Publish(c *gin.Context) {
	announcementID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || announcementID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	announcement, err := h.announcementService.Publish(c.Request.Context(), middleware.CurrentUserID(c), announcementID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newAnnouncementData(announcement))
}

func (h *AnnouncementHandler) Revoke(c *gin.Context) {
	announcementID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || announcementID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	announcement, err := h.announcementService.Revoke(c.Request.Context(), middleware.CurrentUserID(c), announcementID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, newAnnouncementData(announcement))
}

func newAnnouncementData(announcement *model.Announcement) announcementData {
	var publishedAt *string
	if announcement.PublishedAt != nil {
		value := announcement.PublishedAt.UTC().Format("2006-01-02T15:04:05Z")
		publishedAt = &value
	}

	return announcementData{
		ID:          announcement.ID,
		ProjectID:   announcement.ProjectID,
		Title:       announcement.Title,
		Content:     announcement.Content,
		IsPinned:    announcement.IsPinned,
		Status:      announcement.Status,
		PublishedAt: publishedAt,
		CreatedAt:   announcement.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   announcement.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}
