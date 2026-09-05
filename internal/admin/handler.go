package admin

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/pkg/response"
)

// Handler expose l'éditeur (JWT obligatoire).
type Handler struct {
	svc     *Service
	authSvc *auth.Service
}

func NewHandler(svc *Service, authSvc *auth.Service) *Handler {
	return &Handler{svc: svc, authSvc: authSvc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	g := r.Group("/admin")
	g.Use(auth.Bearer(h.authSvc))
	g.GET("/stops", h.searchStops)
	g.POST("/routes", h.create)
	g.GET("/routes/:routeId", h.draft)
	g.PATCH("/routes/:routeId/stops", h.patchStop)
	g.PATCH("/routes/:routeId/type", h.patchRouteType)
	g.POST("/routes/:routeId/recalculate", h.recalculate)
	g.POST("/routes/:routeId/match", h.match)
	g.POST("/routes/:routeId/save", h.save)
}

func (h *Handler) searchStops(c *gin.Context) {
	q := c.Query("q")
	limit := 20
	out, err := h.svc.SearchStops(c.Request.Context(), q, limit)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) draft(c *gin.Context) {
	op, depot := scope(c)
	out, err := h.svc.Draft(c.Request.Context(), op, depot, c.Param("routeId"), c.Query("trip_id"))
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) create(c *gin.Context) {
	var req dto.CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Requête invalide.")
		return
	}
	out, err := h.svc.CreateRoute(c.Request.Context(), req)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) patchStop(c *gin.Context) {
	var req dto.PatchStopRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Coordonnées invalides.")
		return
	}
	out, err := h.svc.PatchStop(c.Request.Context(), req.OperatorCode, req.DepotCode, c.Param("routeId"), req.StopID, req.Lat, req.Lng)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) patchRouteType(c *gin.Context) {
	var req dto.PatchRouteTypeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_route_type", "Type de ligne invalide.")
		return
	}
	out, err := h.svc.PatchRouteType(c.Request.Context(), req.OperatorCode, req.DepotCode, c.Param("routeId"), req.RouteType)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) recalculate(c *gin.Context) {
	var req dto.RecalcRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Requête invalide.")
		return
	}
	out, err := h.svc.Recalculate(c.Request.Context(), req.OperatorCode, req.DepotCode, c.Param("routeId"), req)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) match(c *gin.Context) {
	var req dto.MatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Requête invalide.")
		return
	}
	out, err := h.svc.MatchShape(c.Request.Context(), req.OperatorCode, req.DepotCode, c.Param("routeId"), req)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func (h *Handler) save(c *gin.Context) {
	var req dto.SaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Requête invalide.")
		return
	}
	out, err := h.svc.Save(c.Request.Context(), req.OperatorCode, req.DepotCode, c.Param("routeId"), req)
	if err != nil {
		writeAdminErr(c, err)
		return
	}
	response.OK(c, out)
}

func scope(c *gin.Context) (string, string) {
	op := c.Query("operator_code")
	depot := c.Query("depot_code")
	if cl, ok := c.Get("auth"); ok {
		a := cl.(auth.Claims)
		if op == "" {
			op = a.OperatorCode
		}
		if depot == "" {
			depot = a.DepotCode
		}
	}
	return op, depot
}

func writeAdminErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog.ErrScopeRequired):
		response.Error(c, http.StatusBadRequest, "scope_required", "Transporteur et dépôt obligatoires.")
	case errors.Is(err, catalog.ErrRouteNotFound), errors.Is(err, catalog.ErrTripNotFound):
		response.Error(c, http.StatusNotFound, "route_not_found", "Ligne ou course introuvable.")
	case errors.Is(err, ErrInvalidRouteType):
		response.Error(c, http.StatusBadRequest, "invalid_route_type", "Type de ligne invalide.")
	case errors.Is(err, ErrInvalidCalendar):
		response.Error(c, http.StatusBadRequest, "invalid_calendar", "Jours de circulation ou période invalides.")
	case errors.Is(err, ErrInvalidSchedule):
		response.Error(c, http.StatusBadRequest, "invalid_schedule", "Horaires de course invalides.")
	case errors.Is(err, ErrInvalidCoords), errors.Is(err, ErrShapeTooShort), errors.Is(err, ErrTooFewStops):
		response.Error(c, http.StatusBadRequest, "invalid_coords", "Coordonnées ou séquence d’arrêts invalides.")
	case errors.Is(err, ErrOSRM):
		response.Error(c, http.StatusBadGateway, "osrm_failed", "Impossible de coller le tracé sur les rues.")
	case errors.Is(err, ErrTripActive):
		response.Error(c, http.StatusConflict, "trip_active", "Une course est guidée en ce moment. Réessayez plus tard.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal", "Erreur interne.")
	}
}
