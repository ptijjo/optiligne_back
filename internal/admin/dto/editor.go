package dto

import "github.com/ptijjo/optiligne_back/internal/guidance/dto"

// EditorStop est un arrêt éditable (WGS84).
type EditorStop struct {
	StopID   string  `json:"stopId"`
	Name     string  `json:"name"`
	Sequence int     `json:"sequence"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
}

// StopHit est un résultat de recherche d'arrêt GTFS.
type StopHit struct {
	StopID string  `json:"stopId"`
	Name   string  `json:"name"`
	Lat    float64 `json:"lat"`
	Lng    float64 `json:"lng"`
}

// Waypoint est un point de passage (hors arrêt GTFS).
type Waypoint struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	AfterStopID string  `json:"afterStopId,omitempty"`
}

// Draft est l'état éditeur d'une ligne.
type Draft struct {
	RouteID     string         `json:"routeId"`
	ShortName   string         `json:"shortName"`
	LongName    string         `json:"longName"`
	TripID      string         `json:"tripId"`
	ShapeID     string         `json:"shapeId"`
	FeedID      string         `json:"-"`
	FeedVersion string         `json:"feedVersion"`
	Shape       dto.LineString `json:"shape"`
	Stops       []EditorStop   `json:"stops"`
}

// PatchStopRequest déplace un arrêt.
type PatchStopRequest struct {
	StopID       string  `json:"stopId" binding:"required"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
	OperatorCode string  `json:"operatorCode" binding:"required"`
	DepotCode    string  `json:"depotCode" binding:"required"`
}

// RecalcRequest demande un tracé OSRM (preview).
type RecalcRequest struct {
	OperatorCode string       `json:"operatorCode" binding:"required"`
	DepotCode    string       `json:"depotCode" binding:"required"`
	TripID       string       `json:"tripId" binding:"required"`
	Stops        []EditorStop `json:"stops" binding:"required,min=2"`
	Waypoints    []Waypoint   `json:"waypoints"`
}

// RecalcResponse est un tracé non persisté.
type RecalcResponse struct {
	Shape dto.LineString `json:"shape"`
}

// MatchRequest demande un map matching OSRM sur le tracé actuel (preview).
type MatchRequest struct {
	OperatorCode string         `json:"operatorCode" binding:"required"`
	DepotCode    string         `json:"depotCode" binding:"required"`
	TripID       string         `json:"tripId" binding:"required"`
	Shape        dto.LineString `json:"shape" binding:"required"`
}

// SaveRequest persiste arrêts + shape.
type SaveRequest struct {
	OperatorCode string         `json:"operatorCode" binding:"required"`
	DepotCode    string         `json:"depotCode" binding:"required"`
	TripID       string         `json:"tripId" binding:"required"`
	Stops        []EditorStop   `json:"stops" binding:"required,min=2"`
	Shape        dto.LineString `json:"shape" binding:"required"`
}

// SaveResponse confirme la sync GTFS / mobile.
type SaveResponse struct {
	FeedVersion string `json:"feedVersion"`
	Message     string `json:"message"`
}

// TimedStop est un arrêt avec horaires (secondes depuis minuit service).
type TimedStop struct {
	StopID       string
	StopSequence int
	ArrivalSec   int
	DepartureSec int
	Name         string
	Lat          float64
	Lng          float64
}
