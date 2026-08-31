package catalog_test

import (
	"context"
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/internal/catalog/dto"
)

type fakeStore struct {
	routes []dto.Route
	stops  []dto.Stop
	err    error
}

func (f fakeStore) SchoolRoutes(ctx context.Context, operatorCode, depotCode string) ([]dto.Route, error) {
	if operatorCode != "transavold" || depotCode != "fluo57" {
		return nil, nil
	}
	return f.routes, nil
}

func (f fakeStore) TripsOnDate(ctx context.Context, operatorCode, depotCode, routeID string, day time.Time) ([]dto.Trip, error) {
	return nil, nil
}

func (f fakeStore) TripStops(ctx context.Context, operatorCode, depotCode, tripID string) ([]dto.Stop, error) {
	if f.err != nil {
		return nil, f.err
	}
	if operatorCode != "transavold" || depotCode != "fluo57" || tripID == "" {
		return nil, catalog.ErrTripNotFound
	}
	return f.stops, nil
}

func TestListRoutes_SansDepot_ScopeRequired(t *testing.T) {
	svc := catalog.NewService(fakeStore{}, "")
	_, err := svc.ListRoutes(context.Background(), "transavold", "")
	if err != catalog.ErrScopeRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestListRoutes_InclutRegulieresDuDepot(t *testing.T) {
	svc := catalog.NewService(fakeStore{routes: []dto.Route{
		{ID: "R1", ShortName: "57SAV34", RouteType: 713},
		{ID: "R2", ShortName: "57R004", RouteType: 204},
		{ID: "R3", ShortName: "57ECR00", RouteType: 712},
	}}, "")
	got, err := svc.ListRoutes(context.Background(), "transavold", "fluo57")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("attendu 3 lignes du périmètre, got %+v", got)
	}
}

func TestListRoutes_IsoleAutreTransporteur(t *testing.T) {
	svc := catalog.NewService(fakeStore{routes: []dto.Route{
		{ID: "R1", ShortName: "57SAV34", RouteType: 713},
	}}, "")
	got, err := svc.ListRoutes(context.Background(), "concurrent", "fluo57")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("fuite %+v", got)
	}
}

func TestListRoutes_LockOperatorEnv(t *testing.T) {
	svc := catalog.NewService(fakeStore{routes: []dto.Route{
		{ID: "R1", ShortName: "57SAV34", RouteType: 713},
	}}, "transavold")
	got, err := svc.ListRoutes(context.Background(), "concurrent", "fluo57")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("le lock serveur doit forcer transavold, got %+v", got)
	}
}

func TestEnsureSchoolTrip_HorsPerimetre(t *testing.T) {
	if err := catalog.EnsureSchoolTrip(713, false); err != catalog.ErrTripNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestEnsureSchoolTrip_ReguliereDansPerimetre(t *testing.T) {
	if err := catalog.EnsureSchoolTrip(204, true); err != nil {
		t.Fatalf("une ligne régulière du dépôt doit être guidable: %v", err)
	}
}

func TestListTripStops_SansDepot_ScopeRequired(t *testing.T) {
	svc := catalog.NewService(fakeStore{}, "")
	_, err := svc.ListTripStops(context.Background(), "transavold", "", "T1")
	if err != catalog.ErrScopeRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestListTripStops_OrdreSequence(t *testing.T) {
	svc := catalog.NewService(fakeStore{stops: []dto.Stop{
		{StopID: "A", Name: "Départ", Sequence: 1, ArrivalSec: 25000, DepartureSec: 25000},
		{StopID: "B", Name: "École", Sequence: 2, ArrivalSec: 27000, DepartureSec: 27000},
	}}, "")
	got, err := svc.ListTripStops(context.Background(), "transavold", "fluo57", "T1")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Name != "Départ" || got[1].Name != "École" {
		t.Fatalf("got %+v", got)
	}
}

func TestListTripStops_HorsPerimetre(t *testing.T) {
	svc := catalog.NewService(fakeStore{err: catalog.ErrTripNotFound}, "")
	_, err := svc.ListTripStops(context.Background(), "transavold", "fluo57", "HORS")
	if err != catalog.ErrTripNotFound {
		t.Fatalf("err = %v", err)
	}
}
