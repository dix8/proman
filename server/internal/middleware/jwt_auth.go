package middleware

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"proman/server/internal/pkg/apperror"
	"proman/server/internal/pkg/jwtutil"
)

const (
	CurrentUserIDKey   = "current_user_id"
	CurrentUsernameKey = "current_username"
)

func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		headerValue := strings.TrimSpace(c.GetHeader("Authorization"))
		if !strings.HasPrefix(headerValue, "Bearer ") {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40102, "JWT 无效或已过期"))
			c.Abort()
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(headerValue, "Bearer "))
		if tokenString == "" {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40102, "JWT 无效或已过期"))
			c.Abort()
			return
		}

		claims, err := jwtutil.ParseToken(tokenString, secret)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) ||
				errors.Is(err, jwt.ErrTokenMalformed) ||
				errors.Is(err, jwt.ErrTokenSignatureInvalid) ||
				errors.Is(err, jwt.ErrTokenInvalidClaims) ||
				errors.Is(err, jwt.ErrTokenUnverifiable) {
				_ = c.Error(apperror.New(http.StatusUnauthorized, 40102, "JWT 无效或已过期"))
				c.Abort()
				return
			}
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40102, "JWT 无效或已过期"))
			c.Abort()
			return
		}

		userID, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil {
			_ = c.Error(apperror.New(http.StatusUnauthorized, 40102, "JWT 无效或已过期"))
			c.Abort()
			return
		}

		c.Set(CurrentUserIDKey, userID)
		c.Set(CurrentUsernameKey, claims.Username)
		c.Next()
	}
}

func CurrentUserID(c *gin.Context) uint64 {
	value, ok := c.Get(CurrentUserIDKey)
	if !ok {
		return 0
	}

	userID, ok := value.(uint64)
	if !ok {
		return 0
	}

	return userID
}
