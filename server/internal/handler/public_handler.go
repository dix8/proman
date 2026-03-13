package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/middleware"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/response"
	"proman/server/internal/service"
)

type PublicHandler struct {
	publicService *service.PublicService
}

func NewPublicHandler(publicService *service.PublicService) *PublicHandler {
	return &PublicHandler{publicService: publicService}
}

func (h *PublicHandler) GetProject(c *gin.Context) {
	result, err := h.publicService.GetProject(c.Request.Context(), middleware.CurrentProjectID(c))
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, result)
}

func (h *PublicHandler) ListVersions(c *gin.Context) {
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

	result, err := h.publicService.ListVersions(c.Request.Context(), middleware.CurrentProjectID(c), page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, result)
}

func (h *PublicHandler) GetVersionChangelogs(c *gin.Context) {
	version := strings.TrimSpace(c.Param("version"))
	result, err := h.publicService.GetVersionChangelogs(c.Request.Context(), middleware.CurrentProjectID(c), version)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, result)
}

func (h *PublicHandler) ListAnnouncements(c *gin.Context) {
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

	result, err := h.publicService.ListAnnouncements(c.Request.Context(), middleware.CurrentProjectID(c), page, pageSize)
	if err != nil {
		_ = c.Error(err)
		return
	}
	response.Success(c, result)
}
