package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/response"
	"proman/server/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apperror.New(http.StatusBadRequest, 40001, "参数错误"))
		return
	}

	req.Username = strings.TrimSpace(req.Username)

	result, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		_ = c.Error(err)
		return
	}

	response.Success(c, gin.H{
		"token":      result.Token,
		"expires_at": result.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	})
}
