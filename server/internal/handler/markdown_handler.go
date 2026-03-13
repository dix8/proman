package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/markdownpreview"
	"proman/server/internal/pkg/response"
)

type MarkdownHandler struct {
	renderer *markdownpreview.Renderer
}

type markdownPreviewRequest struct {
	Content string `json:"content"`
}

func NewMarkdownHandler(renderer *markdownpreview.Renderer) *MarkdownHandler {
	return &MarkdownHandler{renderer: renderer}
}

func (h *MarkdownHandler) Preview(c *gin.Context) {
	var req markdownPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" || len([]rune(content)) > 20000 {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	html, err := h.renderer.Render(content)
	if err != nil {
		_ = c.Error(apperror.Internal(err))
		return
	}

	response.Success(c, gin.H{
		"html": html,
	})
}
