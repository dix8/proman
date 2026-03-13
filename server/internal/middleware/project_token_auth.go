package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/projecttoken"
	"proman/server/internal/repository"
)

const CurrentProjectIDKey = "current_project_id"

func ProjectTokenAuth(projectRepo *repository.ProjectRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		headerValue := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(headerValue, "Bearer ") {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40103, "项目 Token 无效"))
			c.Abort()
			return
		}

		token := strings.TrimSpace(strings.TrimPrefix(headerValue, "Bearer "))
		if token == "" {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40103, "项目 Token 无效"))
			c.Abort()
			return
		}

		project, err := projectRepo.FindByTokenHash(c.Request.Context(), projecttoken.Hash(token))
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				_ = c.Error(apperror.New(http.StatusUnauthorized, 40103, "项目 Token 无效"))
				c.Abort()
				return
			}
			_ = c.Error(apperror.Internal(err))
			c.Abort()
			return
		}

		c.Set(CurrentProjectIDKey, project.ID)
		c.Next()
	}
}

func CurrentProjectID(c *gin.Context) uint64 {
	value, ok := c.Get(CurrentProjectIDKey)
	if !ok {
		return 0
	}
	projectID, ok := value.(uint64)
	if !ok {
		return 0
	}
	return projectID
}
