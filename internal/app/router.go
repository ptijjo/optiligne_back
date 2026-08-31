package app

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func newRouter(a *App) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery(), gin.Logger())
	if a.cfg != nil {
		r.Use(cors(a.cfg.AdminCORSOrigins))
	}

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{"status": "ok"},
		})
	})

	if a.auth != nil {
		a.auth.RegisterRoutes(r)
	}
	if a.catalog != nil {
		a.catalog.RegisterRoutes(r)
	}
	if a.guidance != nil {
		a.guidance.RegisterRoutes(r)
	}
	if a.admin != nil {
		a.admin.RegisterRoutes(r)
	}
	a.mountWebSocket(r)
	return r
}

func cors(originsCSV string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, o := range strings.Split(originsCSV, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allowed[o] = struct{}{}
		}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if _, ok := allowed[origin]; ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
