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
		r.Use(cors(a.cfg.CORSOrigins))
	}

	r.GET("/health", health)
	r.HEAD("/health", health)

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
	allowAll := false
	for _, o := range strings.Split(originsCSV, ",") {
		o = strings.TrimSpace(strings.Trim(o, `"'`))
		if o == "" {
			continue
		}
		if o == "*" {
			allowAll = true
			continue
		}
		allowed[o] = struct{}{}
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		ok := origin != "" && (allowAll || mapHas(allowed, origin))
		if ok {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
			c.Header("Access-Control-Max-Age", "86400")
			c.Header("Vary", "Origin")
		}
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func mapHas(m map[string]struct{}, k string) bool {
	_, ok := m[k]
	return ok
}
