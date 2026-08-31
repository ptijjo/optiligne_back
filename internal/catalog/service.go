package catalog

import (
	"context"
	"time"

	"github.com/ptijjo/optiligne_back/internal/catalog/dto"
)

// Store charge les données catalogue déjà persistées.
type Store interface {
	SchoolRoutes(ctx context.Context, operatorCode, depotCode string) ([]dto.Route, error)
	TripsOnDate(ctx context.Context, operatorCode, depotCode, routeID string, day time.Time) ([]dto.Trip, error)
	TripStops(ctx context.Context, operatorCode, depotCode, tripID string) ([]dto.Stop, error)
}

// Service liste lignes et courses dans le périmètre.
type Service struct {
	store             Store
	lockedOperatorCode string
}

func NewService(store Store, lockedOperatorCode string) *Service {
	return &Service{store: store, lockedOperatorCode: lockedOperatorCode}
}

func (s *Service) resolveOperator(operatorCode string) (string, error) {
	if s.lockedOperatorCode != "" {
		return s.lockedOperatorCode, nil
	}
	if operatorCode == "" {
		return "", ErrScopeRequired
	}
	return operatorCode, nil
}

// ListRoutes retourne les lignes du dépôt (régulières, associées, scolaires).
func (s *Service) ListRoutes(ctx context.Context, operatorCode, depotCode string) ([]dto.Route, error) {
	op, err := s.resolveOperator(operatorCode)
	if err != nil {
		return nil, err
	}
	if depotCode == "" {
		return nil, ErrScopeRequired
	}
	return s.store.SchoolRoutes(ctx, op, depotCode)
}

// ListTrips retourne les courses d'une ligne du périmètre pour une date.
func (s *Service) ListTrips(ctx context.Context, operatorCode, depotCode, routeID, date string) ([]dto.Trip, error) {
	op, err := s.resolveOperator(operatorCode)
	if err != nil {
		return nil, err
	}
	if depotCode == "" || routeID == "" || date == "" {
		return nil, ErrScopeRequired
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}
	trips, err := s.store.TripsOnDate(ctx, op, depotCode, routeID, day)
	if err != nil {
		return nil, err
	}
	if trips == nil {
		trips = []dto.Trip{}
	}
	return trips, nil
}

// ListTripStops retourne les arrêts d'une course du périmètre, en ordre de passage.
func (s *Service) ListTripStops(ctx context.Context, operatorCode, depotCode, tripID string) ([]dto.Stop, error) {
	op, err := s.resolveOperator(operatorCode)
	if err != nil {
		return nil, err
	}
	if depotCode == "" || tripID == "" {
		return nil, ErrScopeRequired
	}
	stops, err := s.store.TripStops(ctx, op, depotCode, tripID)
	if err != nil {
		return nil, err
	}
	if stops == nil {
		stops = []dto.Stop{}
	}
	return stops, nil
}

// EnsureSchoolTrip vérifie qu'une course est dans le périmètre du dépôt (tous types).
func EnsureSchoolTrip(_ int, inScope bool) error {
	if !inScope {
		return ErrTripNotFound
	}
	return nil
}
