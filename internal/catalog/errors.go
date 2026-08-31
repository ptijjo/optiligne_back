package catalog

import "errors"

var (
	ErrScopeRequired = errors.New("scope_required")
	ErrRouteNotFound = errors.New("route_not_found")
	ErrTripNotFound  = errors.New("trip_not_found")
)
