package catalog

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/catalog/dto"
	"github.com/ptijjo/optiligne_back/pkg/response"
)

// Handler expose le catalogue HTTP.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes monte les routes catalogue.
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/catalog/routes", h.listRoutes)
	r.GET("/catalog/routes/:id/trips", h.listTrips)
	r.GET("/catalog/trips/:tripId/stops", h.listTripStops)
}

func (h *Handler) listRoutes(c *gin.Context) {
	// 1. Lire le périmètre (téléphone provisionné).
	op := c.Query("operator_code")
	depot := c.Query("depot_code")
	if depot == "" {
		depot = c.Query("depot_id")
	}
	routes, err := h.svc.ListRoutes(c.Request.Context(), op, depot)
	if err != nil {
		writeErr(c, err)
		return
	}
	if routes == nil {
		routes = []dto.Route{}
	}
	response.OK(c, routes)
}

func (h *Handler) listTrips(c *gin.Context) {
	op := c.Query("operator_code")
	depot := c.Query("depot_code")
	if depot == "" {
		depot = c.Query("depot_id")
	}
	trips, err := h.svc.ListTrips(c.Request.Context(), op, depot, c.Param("id"), c.Query("date"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, trips)
}

func (h *Handler) listTripStops(c *gin.Context) {
	// 1. Lire le périmètre (téléphone provisionné).
	op := c.Query("operator_code")
	depot := c.Query("depot_code")
	if depot == "" {
		depot = c.Query("depot_id")
	}
	// 2. Arrêts de la course dans l'ordre de passage.
	stops, err := h.svc.ListTripStops(c.Request.Context(), op, depot, c.Param("tripId"))
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, stops)
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrScopeRequired):
		response.Error(c, http.StatusBadRequest, "scope_required", "Transporteur et dépôt obligatoires.")
	case errors.Is(err, ErrTripNotFound):
		response.Error(c, http.StatusNotFound, "trip_not_found", "Course introuvable.")
	case errors.Is(err, ErrRouteNotFound):
		response.Error(c, http.StatusNotFound, "route_not_found", "Ligne ou course introuvable.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal", "Erreur interne.")
	}
}
