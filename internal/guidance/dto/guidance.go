package dto

import "encoding/json"

// Guidance est l'état envoyé au chauffeur.
type Guidance struct {
	Frac     float64 `json:"frac"`
	OffsetM  float64 `json:"offset_m"`
	NextStop string  `json:"next_stop"`
	DelayS   int     `json:"delay_s"`
	State    string  `json:"state"`
}

// StartRequest démarre une session de guidage.
type StartRequest struct {
	TripID       string `json:"tripId" binding:"required"`
	Date         string `json:"date" binding:"required"`
	OperatorCode string `json:"operatorCode"`
	DepotCode    string `json:"depotCode"`
}

// LineString est le tracé de la course (GeoJSON, lon/lat).
type LineString struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

// StopPoint est un arrêt à afficher sur la carte.
type StopPoint struct {
	Name string  `json:"name"`
	Lon  float64 `json:"lon"`
	Lat  float64 `json:"lat"`
}

// StartResponse identifie la session et fournit le fond de carte.
type StartResponse struct {
	SessionID string     `json:"sessionId"`
	TripID    string     `json:"tripId"`
	Shape     LineString `json:"shape"`
	Stops     []StopPoint `json:"stops"`
}

func EmptyLineString() LineString {
	return LineString{Type: "LineString", Coordinates: [][]float64{}}
}

// DecodeLineString lit un ST_AsGeoJSON.
func DecodeLineString(raw string) LineString {
	out := EmptyLineString()
	if raw == "" {
		return out
	}
	var parsed struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return out
	}
	if parsed.Type != "LineString" || len(parsed.Coordinates) < 2 {
		return out
	}
	out.Coordinates = parsed.Coordinates
	return out
}

const (
	StateOnRoute   = "on_route"
	StateOffRoute  = "off_route"
	StateAmbiguous = "ambiguous"
	StateArrived   = "arrived"
)
