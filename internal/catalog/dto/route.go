package dto

// Route est une ligne catalogue.
type Route struct {
	ID        string `json:"id"`
	ShortName string `json:"shortName"`
	LongName  string `json:"longName"`
	RouteType int    `json:"routeType"`
}

// Trip est une course du jour.
type Trip struct {
	ID           string `json:"id"`
	Headsign     string `json:"headsign"`
	RouteID      string `json:"routeId"`
	DepartureSec int    `json:"departureSec"`
}

// Stop est un arrêt ordonné d'une course.
type Stop struct {
	StopID       string `json:"stopId"`
	Name         string `json:"name"`
	Sequence     int    `json:"sequence"`
	ArrivalSec   int    `json:"arrivalSec"`
	DepartureSec int    `json:"departureSec"`
}
