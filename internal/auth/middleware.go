package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/pkg/response"
)

// Bearer exige un access JWT valide.
func Bearer(svc *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		tok := bearerToken(c)
		if tok == "" {
			response.Error(c, http.StatusUnauthorized, "unauthorized", "Authentification requise.")
			c.Abort()
			return
		}
		cl, err := svc.ParseAccess(tok)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, "unauthorized", "Session expirée.")
			c.Abort()
			return
		}
		c.Set("auth", cl)
		c.Next()
	}
}
