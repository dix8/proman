package middleware

import (
	"github.com/gin-gonic/gin"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/response"
)

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		appErr := apperror.From(c.Errors.Last().Err)
		response.Error(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
	}
}
