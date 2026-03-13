package handler

import (
	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Get(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
	})
}
