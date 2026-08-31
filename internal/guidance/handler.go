package guidance

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/internal/guidance/dto"
	"github.com/ptijjo/optiligne_back/pkg/response"
)

// Handler démarre les sessions de guidage.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.POST("/guidance/sessions", h.start)
}

func (h *Handler) start(c *gin.Context) {
	var req dto.StartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "invalid_position", "Requête invalide.")
		return
	}
	depot := req.DepotCode
	out, err := h.svc.Start(c.Request.Context(), req.OperatorCode, depot, req.TripID, req.Date)
	if err != nil {
		writeErr(c, err)
		return
	}
	response.OK(c, out)
}

func writeErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, catalog.ErrScopeRequired):
		response.Error(c, http.StatusBadRequest, "scope_required", "Transporteur et dépôt obligatoires.")
	case errors.Is(err, catalog.ErrTripNotFound), errors.Is(err, catalog.ErrRouteNotFound):
		response.Error(c, http.StatusNotFound, "trip_not_found", "Course introuvable.")
	case errors.Is(err, ErrInvalidPosition):
		response.Error(c, http.StatusBadRequest, "invalid_position", "Position GPS invalide.")
	default:
		response.Error(c, http.StatusInternalServerError, "internal", "Erreur interne.")
	}
}
