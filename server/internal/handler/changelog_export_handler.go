package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/middleware"
	"proman/server/internal/pkg/apperror"
	"proman/server/internal/service"
)

type ChangelogExportHandler struct {
	exportService *service.ChangelogExportService
}

func NewChangelogExportHandler(exportService *service.ChangelogExportService) *ChangelogExportHandler {
	return &ChangelogExportHandler{exportService: exportService}
}

func (h *ChangelogExportHandler) Export(c *gin.Context) {
	projectID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || projectID == 0 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	format := strings.TrimSpace(c.Query("format"))
	var versionID *uint64
	if rawVersionID := strings.TrimSpace(c.Query("version_id")); rawVersionID != "" {
		parsed, err := strconv.ParseUint(rawVersionID, 10, 64)
		if err != nil || parsed == 0 {
			_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
			return
		}
		versionID = &parsed
	}

	file, err := h.exportService.Export(c.Request.Context(), middleware.CurrentUserID(c), projectID, format, versionID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Header("Content-Type", file.ContentType)
	c.Header("Content-Disposition", file.ContentDisposition)
	c.Data(http.StatusOK, file.ContentType, file.Content)
}
