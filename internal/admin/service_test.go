package admin_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/admin"
	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	guidancedto "github.com/ptijjo/optiligne_back/internal/guidance/dto"
	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

type fakeStore struct {
	draft           *dto.Draft
	err             error
	lastTripID      string
	routeShapes     []string
	siblingTrips    []string
	updatedShapes   []string
	upserted        []dto.EditorStop
	replacedTimes   map[string][]dto.TimedStop
	searchHits      []dto.StopHit
	existingStopIDs map[string]bool
	lastRouteType   int
	scope           *admin.ScopeInfo
	created         *admin.CreateRouteInput
}

func (f *fakeStore) LoadDraft(_ context.Context, _, _, _, tripID string) (*dto.Draft, error) {
	f.lastTripID = tripID
	if f.err != nil {
		return nil, f.err
	}
	cp := *f.draft
	stops := append([]dto.EditorStop(nil), f.draft.Stops...)
	cp.Stops = stops
	return &cp, nil
}

func (f *fakeStore) SearchStops(_ context.Context, query string, limit int) ([]dto.StopHit, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]dto.StopHit, 0)
	q := strings.ToLower(query)
	for _, h := range f.searchHits {
		if strings.Contains(strings.ToLower(h.Name), q) {
			out = append(out, h)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (f *fakeStore) UpdateStop(_ context.Context, _, stopID string, lat, lng float64) error {
	for i := range f.draft.Stops {
		if f.draft.Stops[i].StopID == stopID {
			f.draft.Stops[i].Lat = lat
			f.draft.Stops[i].Lng = lng
		}
	}
	return nil
}

func (f *fakeStore) UpdateRouteType(_ context.Context, _, _ string, routeType int) error {
	f.lastRouteType = routeType
	f.draft.RouteType = routeType
	return nil
}

func (f *fakeStore) UpsertStop(_ context.Context, _ string, stop dto.EditorStop) error {
	f.upserted = append(f.upserted, stop)
	if f.existingStopIDs == nil {
		f.existingStopIDs = map[string]bool{}
	}
	f.existingStopIDs[stop.StopID] = true
	return nil
}

func (f *fakeStore) UpdateShape(_ context.Context, _, shapeID string, pts []gtfs.ShapePoint) error {
	f.updatedShapes = append(f.updatedShapes, shapeID)
	coords := make([][]float64, 0, len(pts))
	for _, p := range pts {
		coords = append(coords, []float64{p.Lon, p.Lat})
	}
	f.draft.Shape = guidancedto.LineString{Type: "LineString", Coordinates: coords}
	return nil
}

func (f *fakeStore) ListSiblingShapeIDs(_ context.Context, _, _, tripID string) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.routeShapes) > 0 {
		return append([]string(nil), f.routeShapes...), nil
	}
	if f.draft != nil && f.draft.TripID == tripID && f.draft.ShapeID != "" {
		return []string{f.draft.ShapeID}, nil
	}
	if f.draft != nil && f.draft.ShapeID != "" {
		return []string{f.draft.ShapeID}, nil
	}
	return nil, nil
}

func (f *fakeStore) ListSiblingTripIDs(_ context.Context, _, _, tripID string) ([]string, error) {
	if len(f.siblingTrips) > 0 {
		return append([]string(nil), f.siblingTrips...), nil
	}
	return []string{tripID}, nil
}

func (f *fakeStore) ListTripStopTimes(_ context.Context, _, tripID string) ([]dto.TimedStop, error) {
	out := make([]dto.TimedStop, 0, len(f.draft.Stops))
	for i, st := range f.draft.Stops {
		out = append(out, dto.TimedStop{
			StopID: st.StopID, StopSequence: i + 1,
			ArrivalSec: 3600 + i*100, DepartureSec: 3600 + i*100,
			Name: st.Name, Lat: st.Lat, Lng: st.Lng,
		})
	}
	_ = tripID
	return out, nil
}

func (f *fakeStore) ReplaceTripStopTimes(_ context.Context, _, tripID string, stops []dto.TimedStop) error {
	if f.replacedTimes == nil {
		f.replacedTimes = map[string][]dto.TimedStop{}
	}
	cp := append([]dto.TimedStop(nil), stops...)
	f.replacedTimes[tripID] = cp
	return nil
}

func (f *fakeStore) RebuildFracsForTrips(context.Context, string, []string) error { return nil }

func (f *fakeStore) RecomputeFracs(context.Context, string, []string) error { return nil }

func (f *fakeStore) ResolveScope(_ context.Context, _, _ string) (*admin.ScopeInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.scope != nil {
		return f.scope, nil
	}
	return &admin.ScopeInfo{
		FeedID: "fv1", FeedVersion: "2026",
		OperatorID: "op1", DepotID: "dep1",
		AgencyID: "AG1", ServiceID: "SVC1",
	}, nil
}

func (f *fakeStore) CreateRoute(_ context.Context, in admin.CreateRouteInput) error {
	if f.err != nil {
		return f.err
	}
	cp := in
	cp.Stops = append([]dto.EditorStop(nil), in.Stops...)
	cp.ShapePoints = append([]gtfs.ShapePoint(nil), in.ShapePoints...)
	cp.Trips = append([]admin.TripInput(nil), in.Trips...)
	for i := range cp.Trips {
		cp.Trips[i].Timed = append([]dto.TimedStop(nil), in.Trips[i].Timed...)
	}
	f.created = &cp
	return nil
}

type fakeRouter struct {
	coords [][]float64
	err    error
}

func (f fakeRouter) Route(context.Context, [][2]float64) ([][]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.coords, nil
}

func (f fakeRouter) Match(_ context.Context, lngLats [][2]float64) ([][]float64, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.coords) > 0 {
		return f.coords, nil
	}
	out := make([][]float64, len(lngLats))
	for i, p := range lngLats {
		out[i] = []float64{p[0], p[1]}
	}
	return out, nil
}

type fakeFiles struct {
	stops      int
	shapes     int
	upserts    int
	stopTimes  int
	routeTypes int
	routes     int
	trips      int
	calendars  int
	routeErr   error
	shapesErr  error
}

func (f *fakeFiles) PatchStop(string, float64, float64) error {
	f.stops++
	return nil
}

func (f *fakeFiles) PatchRouteType(string, int) error {
	f.routeTypes++
	return nil
}

func (f *fakeFiles) UpsertStop(string, string, float64, float64) error {
	f.upserts++
	return nil
}

func (f *fakeFiles) UpsertRoute(string, string, string, string, int) error {
	if f.routeErr != nil {
		return f.routeErr
	}
	f.routes++
	return nil
}

func (f *fakeFiles) UpsertTrip(string, string, string, string, string) error {
	f.trips++
	return nil
}

func (f *fakeFiles) UpsertCalendar(admin.CalendarFileRow) error {
	f.calendars++
	return nil
}

func (f *fakeFiles) ReplaceShape(string, []gtfs.ShapePoint) error {
	f.shapes++
	return nil
}

func (f *fakeFiles) ReplaceShapes([]string, []gtfs.ShapePoint) error {
	if f.shapesErr != nil {
		return f.shapesErr
	}
	f.shapes++
	return nil
}

func (f *fakeFiles) ReplaceStopTimes([]string, []admin.StopTimeFileRow) error {
	f.stopTimes++
	return nil
}

type fakeGuard struct{ active string }

func (g fakeGuard) HasActiveTrip(tripID string) bool { return g.active == tripID }

func sampleDraft() *dto.Draft {
	return &dto.Draft{
		RouteID: "R1", ShortName: "57S012", RouteType: 713, TripID: "T1", ShapeID: "SH1",
		FeedID: "fv1", FeedVersion: "2026",
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.91, 49.12}}},
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
			{StopID: "B", Name: "École", Sequence: 2, Lat: 49.12, Lng: 6.91},
		},
	}
}

func TestDraft_ScopeRequired(t *testing.T) {
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, nil, nil, "")
	_, err := svc.Draft(context.Background(), "transavold", "", "R1", "")
	if err != catalog.ErrScopeRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestDraft_PasseTripID(t *testing.T) {
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, nil, nil, "")
	_, err := svc.Draft(context.Background(), "transavold", "fluo57", "R1", "T9")
	if err != nil {
		t.Fatal(err)
	}
	if store.lastTripID != "T9" {
		t.Fatalf("tripID = %q", store.lastTripID)
	}
}

func TestPatchStop_RefuseCoords(t *testing.T) {
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.PatchStop(context.Background(), "transavold", "fluo57", "R1", "A", 200, 6.9)
	if err != admin.ErrInvalidCoords {
		t.Fatalf("err = %v", err)
	}
}

func TestPatchStop_RefuseCourseActive(t *testing.T) {
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, &fakeFiles{}, fakeGuard{active: "T1"}, "")
	_, err := svc.PatchStop(context.Background(), "transavold", "fluo57", "R1", "A", 49.2, 6.92)
	if err != admin.ErrTripActive {
		t.Fatalf("err = %v", err)
	}
}

func TestPatchStop_EcritFichier(t *testing.T) {
	files := &fakeFiles{}
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	out, err := svc.PatchStop(context.Background(), "transavold", "fluo57", "R1", "A", 49.201, 6.928)
	if err != nil {
		t.Fatal(err)
	}
	if files.stops != 1 {
		t.Fatalf("stops.txt patches = %d", files.stops)
	}
	if out.Stops[0].Lat != 49.201 {
		t.Fatalf("lat = %v", out.Stops[0].Lat)
	}
}

func TestPatchRouteType_RefuseTypeInvalide(t *testing.T) {
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.PatchRouteType(context.Background(), "transavold", "fluo57", "R1", 999)
	if err != admin.ErrInvalidRouteType {
		t.Fatalf("err = %v", err)
	}
}

func TestPatchRouteType_EcritFichier(t *testing.T) {
	files := &fakeFiles{}
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	out, err := svc.PatchRouteType(context.Background(), "transavold", "fluo57", "R1", 712)
	if err != nil {
		t.Fatal(err)
	}
	if files.routes != 1 {
		t.Fatalf("routes.txt upserts = %d", files.routes)
	}
	if store.lastRouteType != 712 {
		t.Fatalf("route_type = %d", store.lastRouteType)
	}
	if out.RouteType != 712 {
		t.Fatalf("draft route_type = %d", out.RouteType)
	}
}

func TestPatchRouteType_FichierKOApresPostGIS(t *testing.T) {
	files := &fakeFiles{routeErr: errors.New("permission denied")}
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	_, err := svc.PatchRouteType(context.Background(), "transavold", "fluo57", "R1", 712)
	if !errors.Is(err, admin.ErrGTFSFiles) {
		t.Fatalf("err = %v", err)
	}
	if store.lastRouteType != 712 {
		t.Fatalf("PostGIS aurait dû être mis à jour, got %d", store.lastRouteType)
	}
}

func TestRecalculate_InsereWaypoints(t *testing.T) {
	var got [][2]float64
	rt := captureRouter{fn: func(coords [][2]float64) ([][]float64, error) {
		got = coords
		return [][]float64{{6.9, 49.1}, {6.905, 49.11}, {6.91, 49.12}}, nil
	}}
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, rt, nil, nil, "")
	out, err := svc.Recalculate(context.Background(), "transavold", "fluo57", "R1", dto.RecalcRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops:     sampleDraft().Stops,
		Waypoints: []dto.Waypoint{{Lat: 49.11, Lng: 6.905}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || got[1][0] != 6.905 {
		t.Fatalf("via = %+v", got)
	}
	if len(out.Shape.Coordinates) != 3 {
		t.Fatalf("shape = %+v", out.Shape)
	}
}

func TestRecalculate_WaypointEntreArrets(t *testing.T) {
	var got [][2]float64
	rt := captureRouter{fn: func(coords [][2]float64) ([][]float64, error) {
		got = coords
		return [][]float64{{6.90, 49.10}, {6.905, 49.105}, {6.91, 49.11}, {6.92, 49.12}}, nil
	}}
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, rt, nil, nil, "")
	_, err := svc.Recalculate(context.Background(), "transavold", "fluo57", "R1", dto.RecalcRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Place Wendel", Sequence: 1, Lat: 49.10, Lng: 6.90},
			{StopID: "B", Name: "Collège", Sequence: 2, Lat: 49.11, Lng: 6.91},
			{StopID: "C", Name: "Terminus", Sequence: 3, Lat: 49.12, Lng: 6.92},
		},
		Waypoints: []dto.Waypoint{{Lat: 49.105, Lng: 6.905}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("via len = %d %+v", len(got), got)
	}
	if got[0][0] != 6.90 || got[1][0] != 6.905 || got[2][0] != 6.91 || got[3][0] != 6.92 {
		t.Fatalf("waypoint collé avant le dernier arrêt au lieu du segment : %+v", got)
	}
}

func TestRecalculate_AfterStopIdIgnoreLaProximite(t *testing.T) {
	var got [][2]float64
	rt := captureRouter{fn: func(coords [][2]float64) ([][]float64, error) {
		got = coords
		return [][]float64{{6.90, 49.10}, {6.901, 49.20}, {6.91, 49.11}, {6.92, 49.12}}, nil
	}}
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, rt, nil, nil, "")
	_, err := svc.Recalculate(context.Background(), "transavold", "fluo57", "R1", dto.RecalcRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Place Wendel", Sequence: 1, Lat: 49.10, Lng: 6.90},
			{StopID: "B", Name: "Collège", Sequence: 2, Lat: 49.11, Lng: 6.91},
			{StopID: "C", Name: "Terminus", Sequence: 3, Lat: 49.12, Lng: 6.92},
		},
		Waypoints: []dto.Waypoint{{Lat: 49.119, Lng: 6.919, AfterStopID: "A"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 {
		t.Fatalf("via len = %d %+v", len(got), got)
	}
	if got[0][0] != 6.90 || got[1][0] != 6.919 || got[2][0] != 6.91 || got[3][0] != 6.92 {
		t.Fatalf("afterStopId A doit intercaler avant B, pas avant C : %+v", got)
	}
}

func TestMatch_EnvoieLeShapePasLesArrets(t *testing.T) {
	var got [][2]float64
	rt := captureRouter{matchFn: func(coords [][2]float64) ([][]float64, error) {
		got = coords
		return [][]float64{{6.927, 49.2}, {6.927, 49.198}, {6.926, 49.196}}, nil
	}}
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, rt, nil, nil, "")
	out, err := svc.MatchShape(context.Background(), "transavold", "fluo57", "R1", dto.MatchRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Shape: guidancedto.LineString{
			Type: "LineString",
			Coordinates: [][]float64{
				{6.929, 49.2},
				{6.928, 49.201},
				{6.927, 49.2},
				{6.927, 49.196},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 4 || got[1][0] != 6.928 {
		t.Fatalf("match a reçu les arrêts au lieu du shape : %+v", got)
	}
	if len(out.Shape.Coordinates) != 3 || out.Shape.Coordinates[0][0] != 6.927 {
		t.Fatalf("shape = %+v", out.Shape)
	}
}

func TestMatch_RefuseShapeTropCourt(t *testing.T) {
	svc := admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, nil, nil, "")
	_, err := svc.MatchShape(context.Background(), "transavold", "fluo57", "R1", dto.MatchRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}}},
	})
	if err != admin.ErrShapeTooShort {
		t.Fatalf("err = %v", err)
	}
}

type captureRouter struct {
	fn      func([][2]float64) ([][]float64, error)
	matchFn func([][2]float64) ([][]float64, error)
}

func (c captureRouter) Route(_ context.Context, coords [][2]float64) ([][]float64, error) {
	return c.fn(coords)
}

func (c captureRouter) Match(_ context.Context, coords [][2]float64) ([][]float64, error) {
	if c.matchFn == nil {
		return nil, admin.ErrOSRM
	}
	return c.matchFn(coords)
}

func TestSave_EcritShapeEtStops(t *testing.T) {
	files := &fakeFiles{}
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	out, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: sampleDraft().Stops,
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.92, 49.13}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if files.upserts != 2 || files.shapes != 1 || files.stopTimes != 1 {
		t.Fatalf("files upserts=%d shapes=%d stopTimes=%d", files.upserts, files.shapes, files.stopTimes)
	}
	if out.FeedVersion != "2026" || out.Message == "" {
		t.Fatalf("%+v", out)
	}
}

func TestSave_FichierShapesKOApresPostGIS(t *testing.T) {
	files := &fakeFiles{shapesErr: errors.New("no space left on device")}
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	_, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: sampleDraft().Stops,
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.92, 49.13}}},
	})
	if !errors.Is(err, admin.ErrGTFSFiles) {
		t.Fatalf("err = %v", err)
	}
	if len(store.updatedShapes) == 0 {
		t.Fatal("PostGIS shape aurait dû être mis à jour avant l’échec fichier")
	}
}

func TestSave_PropageShapesDuMemeParcours(t *testing.T) {
	files := &fakeFiles{}
	store := &fakeStore{
		draft:        sampleDraft(),
		routeShapes:  []string{"SH-LUNDI", "SH-MARDI", "SH-LUNDI"},
		siblingTrips: []string{"T1", "T-MARDI"},
	}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	out, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: sampleDraft().Stops,
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.92, 49.13}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.updatedShapes) != 2 {
		t.Fatalf("updatedShapes = %+v (attendu 2 distincts du même parcours)", store.updatedShapes)
	}
	if files.shapes != 1 {
		t.Fatalf("shapes.txt doit être réécrit une seule fois, got %d", files.shapes)
	}
	if !strings.Contains(out.Message, "même parcours") {
		t.Fatalf("message = %q", out.Message)
	}
}

func TestSave_RefuseHorsPerimetre(t *testing.T) {
	svc := admin.NewService(&fakeStore{err: catalog.ErrRouteNotFound}, fakeRouter{}, nil, nil, "")
	_, err := svc.Save(context.Background(), "transavold", "fluo57", "XXX", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: sampleDraft().Stops,
		Shape: sampleDraft().Shape,
	})
	if err != catalog.ErrRouteNotFound {
		t.Fatalf("err = %v", err)
	}
}

func TestSearchStops_FiltreNom(t *testing.T) {
	store := &fakeStore{
		draft: sampleDraft(),
		searchHits: []dto.StopHit{
			{StopID: "1", Name: "Place Wendel", Lat: 49.2, Lng: 6.92},
			{StopID: "2", Name: "Gare Forbach", Lat: 49.18, Lng: 6.9},
		},
	}
	svc := admin.NewService(store, fakeRouter{}, nil, nil, "")
	got, err := svc.SearchStops(context.Background(), "wend", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].StopID != "1" {
		t.Fatalf("%+v", got)
	}
	empty, err := svc.SearchStops(context.Background(), "x", 20)
	if err != nil || len(empty) != 0 {
		t.Fatalf("query trop courte: %v %+v", err, empty)
	}
}

func TestSave_AjouteArretSurTripsSoeurs(t *testing.T) {
	store := &fakeStore{
		draft:        sampleDraft(),
		siblingTrips: []string{"T1", "T2"},
	}
	svc := admin.NewService(store, fakeRouter{}, &fakeFiles{}, nil, "")
	stops := []dto.EditorStop{
		{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
		{StopID: "ol-x", Name: "Nouveau", Sequence: 2, Lat: 49.11, Lng: 6.905},
		{StopID: "B", Name: "École", Sequence: 3, Lat: 49.12, Lng: 6.91},
	}
	_, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: stops,
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.92, 49.13}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.replacedTimes["T1"]) != 3 || len(store.replacedTimes["T2"]) != 3 {
		t.Fatalf("replacedTimes = %+v", store.replacedTimes)
	}
	if store.replacedTimes["T1"][1].StopID != "ol-x" {
		t.Fatalf("%+v", store.replacedTimes["T1"])
	}
	found := false
	for _, u := range store.upserted {
		if u.StopID == "ol-x" {
			found = true
		}
	}
	if !found {
		t.Fatal("nouveau stop non upserté")
	}
}

func TestSave_SupprimeArretSurTripsSoeurs(t *testing.T) {
	draft := sampleDraft()
	draft.Stops = []dto.EditorStop{
		{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
		{StopID: "B", Name: "Milieu", Sequence: 2, Lat: 49.11, Lng: 6.905},
		{StopID: "C", Name: "École", Sequence: 3, Lat: 49.12, Lng: 6.91},
	}
	store := &fakeStore{draft: draft, siblingTrips: []string{"T1", "T2"}}
	svc := admin.NewService(store, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
			{StopID: "C", Name: "École", Sequence: 2, Lat: 49.12, Lng: 6.91},
		},
		Shape: draft.Shape,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.replacedTimes["T2"]) != 2 {
		t.Fatalf("%+v", store.replacedTimes["T2"])
	}
}

func TestSave_CreeNouveauStop(t *testing.T) {
	store := &fakeStore{draft: sampleDraft()}
	svc := admin.NewService(store, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.Save(context.Background(), "transavold", "fluo57", "R1", dto.SaveRequest{
		OperatorCode: "transavold", DepotCode: "fluo57", TripID: "T1",
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
			{StopID: "ol-new", Name: "Créé", Sequence: 2, Lat: 49.105, Lng: 6.902},
			{StopID: "B", Name: "École", Sequence: 3, Lat: 49.12, Lng: 6.91},
		},
		Shape: sampleDraft().Shape,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(store.upserted) != 3 {
		t.Fatalf("upserted = %d", len(store.upserted))
	}
}

func createRouteReq() dto.CreateRouteRequest {
	return dto.CreateRouteRequest{
		OperatorCode: "transavold", DepotCode: "fluo57",
		ShortName: "57S999", LongName: "Nouvelle ligne test", RouteType: 712,
		Stops: []dto.EditorStop{
			{StopID: "A", Name: "Départ", Sequence: 1, Lat: 49.1, Lng: 6.9},
			{StopID: "B", Name: "École", Sequence: 2, Lat: 49.12, Lng: 6.91},
		},
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: [][]float64{{6.9, 49.1}, {6.91, 49.12}}},
		Calendar: dto.CreateCalendar{
			Monday: true, Tuesday: true, Wednesday: true, Thursday: true, Friday: true,
			StartDate: "20260901", EndDate: "20270630",
		},
		Trips: []dto.CreateTripTimes{
			{Headsign: "École", ArrivalSecs: []int{7 * 3600, 7*3600 + 5*60}},
		},
	}
}

func TestCreateRoute_OK(t *testing.T) {
	store := &fakeStore{}
	files := &fakeFiles{}
	svc := admin.NewService(store, fakeRouter{}, files, nil, "")
	out, err := svc.CreateRoute(context.Background(), createRouteReq())
	if err != nil {
		t.Fatal(err)
	}
	if out.RouteID == "" || out.TripID == "" || out.FeedVersion != "2026" {
		t.Fatalf("%+v", out)
	}
	if !strings.HasPrefix(out.RouteID, "ol-r-") || !strings.HasPrefix(out.TripID, "ol-t-") {
		t.Fatalf("ids = %s / %s", out.RouteID, out.TripID)
	}
	if store.created == nil || store.created.RouteType != 712 || len(store.created.Trips) != 1 {
		t.Fatalf("created = %+v", store.created)
	}
	timed := store.created.Trips[0].Timed
	if timed[0].ArrivalSec != 7*3600 || timed[1].ArrivalSec != 7*3600+5*60 {
		t.Fatalf("times = %+v", timed)
	}
	if store.created.Calendar.Monday != true || store.created.Calendar.StartDate != "20260901" {
		t.Fatalf("calendar = %+v", store.created.Calendar)
	}
	if files.routes != 1 || files.trips != 1 || files.calendars != 1 || files.shapes != 1 || files.stopTimes != 1 || files.upserts != 2 {
		t.Fatalf("files = %+v", files)
	}
	if !strings.Contains(out.Message, "Ligne créée") {
		t.Fatalf("message = %q", out.Message)
	}
}

func TestCreateRoute_RefuseTypeInvalide(t *testing.T) {
	req := createRouteReq()
	req.RouteType = 999
	svc := admin.NewService(&fakeStore{}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.CreateRoute(context.Background(), req)
	if err != admin.ErrInvalidRouteType {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRoute_ScopeRequired(t *testing.T) {
	req := createRouteReq()
	req.DepotCode = ""
	svc := admin.NewService(&fakeStore{}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.CreateRoute(context.Background(), req)
	if err != catalog.ErrScopeRequired {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRoute_TropPeuArrets(t *testing.T) {
	req := createRouteReq()
	req.Stops = []dto.EditorStop{
		{StopID: "A", Name: "Seul", Sequence: 1, Lat: 49.1, Lng: 6.9},
	}
	req.Trips = []dto.CreateTripTimes{{ArrivalSecs: []int{7 * 3600}}}
	svc := admin.NewService(&fakeStore{}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.CreateRoute(context.Background(), req)
	if err != admin.ErrTooFewStops {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRoute_RefuseCalendrierVide(t *testing.T) {
	req := createRouteReq()
	req.Calendar = dto.CreateCalendar{StartDate: "20260901", EndDate: "20270630"}
	svc := admin.NewService(&fakeStore{}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.CreateRoute(context.Background(), req)
	if err != admin.ErrInvalidCalendar {
		t.Fatalf("err = %v", err)
	}
}

func TestCreateRoute_RefuseHorairesIncoherents(t *testing.T) {
	req := createRouteReq()
	req.Trips = []dto.CreateTripTimes{{ArrivalSecs: []int{8 * 3600, 7 * 3600}}}
	svc := admin.NewService(&fakeStore{}, fakeRouter{}, &fakeFiles{}, nil, "")
	_, err := svc.CreateRoute(context.Background(), req)
	if err != admin.ErrInvalidSchedule {
		t.Fatalf("err = %v", err)
	}
}
