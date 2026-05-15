package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[recovery] panic: %v\n%s", r, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"ok":      false,
					"error":   "INTERNAL_ERROR",
					"message": "internal server error",
				})
			}
		}()
		c.Next()
	}
}
