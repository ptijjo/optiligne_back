package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/auth/dto"
	"github.com/ptijjo/optiligne_back/pkg/response"
)

// Handler expose login / refresh / logout / me.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/auth")
	g.POST("/login", h.login)
	g.POST("/refresh", h.refresh)
	g.POST("/logout", h.logout)
	g.GET("/me", Bearer(h.svc), h.me)
}

func (h *Handler) login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_credentials", "Identifiants invalides.")
		return
	}
	out, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) refresh(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Session expirée.")
		return
	}
	out, err := h.svc.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		writeAuthErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) logout(c *gin.Context) {
	var req dto.LogoutRequest
	_ = c.ShouldBindJSON(&req)
	_ = h.svc.Logout(c.Request.Context(), req.RefreshToken)
	response.OK(c, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	cClaims, ok := c.Get("auth")
	if !ok {
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Authentification requise.")
		return
	}
	cl := cClaims.(Claims)
	response.OK(c, dto.User{
		ID: cl.Sub, Email: cl.Email, OperatorCode: cl.OperatorCode, DepotCode: cl.DepotCode,
	})
}

func writeAuthErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "invalid_credentials", "Identifiants invalides.")
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrUnauthorized):
		response.Error(c, http.StatusUnauthorized, "unauthorized", "Session expirée.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal", "Erreur interne.")
	}
}

func bearerToken(c *gin.Context) string {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
}
