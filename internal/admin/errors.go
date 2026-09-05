package admin

import "errors"

var (
	ErrInvalidCoords    = errors.New("invalid_coords")
	ErrInvalidRouteType = errors.New("invalid_route_type")
	ErrOSRM             = errors.New("osrm_failed")
	ErrTripActive       = errors.New("trip_active")
	ErrShapeTooShort    = errors.New("shape_too_short")
	ErrTooFewStops      = errors.New("too_few_stops")
	ErrInvalidCalendar  = errors.New("invalid_calendar")
	ErrInvalidSchedule  = errors.New("invalid_schedule")
	ErrGTFSFiles        = errors.New("gtfs_file_failed")
)
