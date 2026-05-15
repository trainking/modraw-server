package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/config"
)

func CORS(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		allowOrigin := ""
		for _, allowed := range cfg.CORSOrigins {
			if allowed == "*" {
				allowOrigin = "*"
				break
			}
			if allowed == origin {
				allowOrigin = origin
				break
			}
		}
		if allowOrigin == "" && len(cfg.CORSOrigins) > 0 {
			allowOrigin = cfg.CORSOrigins[0]
		}

		c.Header("Access-Control-Allow-Origin", allowOrigin)
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
