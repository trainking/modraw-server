package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/config"
	jwtpkg "github.com/trainking/modraw-server/pkg/jwt"
)

const (
	CtxKeyUserID   = "user_id"
	CtxKeyEmail    = "email"
	CtxKeyNickname = "nickname"
)

func AuthRequired(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":      false,
				"error":   "MISSING_TOKEN",
				"message": "missing authorization header",
			})
			return
		}

		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := jwtpkg.ValidateToken(tokenStr, cfg.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"ok":      false,
				"error":   "INVALID_TOKEN",
				"message": "invalid or expired token",
			})
			return
		}

		c.Set(CtxKeyUserID, claims.UserID)
		c.Set(CtxKeyEmail, claims.Email)
		c.Set(CtxKeyNickname, claims.Nickname)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	return c.GetString(CtxKeyUserID)
}

func GetEmail(c *gin.Context) string {
	return c.GetString(CtxKeyEmail)
}
