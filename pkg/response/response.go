package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// ErrorBody est le corps d'erreur API.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Data envoie { "data": ... }.
func Data(c *gin.Context, status int, payload any) {
	c.JSON(status, gin.H{"data": payload})
}

// Error envoie { "error": { "code", "message" } }.
func Error(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{"error": ErrorBody{Code: code, Message: message}})
}

// OK est Data 200.
func OK(c *gin.Context, payload any) {
	Data(c, http.StatusOK, payload)
}
