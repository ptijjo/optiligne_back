package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// health répond aux sondes Docker / Coolify (GET + HEAD, sans auth).
func health(c *gin.Context) {
	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{"status": "ok"},
	})
}
