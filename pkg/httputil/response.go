package httputil

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, data interface{}) {
	c.JSON(200, gin.H{"ok": true, "data": data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(201, gin.H{"ok": true, "data": data})
}

func NoContent(c *gin.Context) {
	c.Status(204)
}

func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"ok": false, "error": code, "message": message})
}

func Paginated(c *gin.Context, data interface{}, page, limit, total int) {
	c.JSON(200, gin.H{
		"ok":    true,
		"data":  data,
		"page":  page,
		"limit": limit,
		"total": total,
	})
}
