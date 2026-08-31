package app

import "github.com/gin-gonic/gin"

// mountWebSocket branche l'upgrade gorilla sur le hub (pas de métier guidage ici).
func (a *App) mountWebSocket(r *gin.Engine) {
	if a.hub == nil {
		return
	}
	a.hub.RegisterRoutes(r)
}
