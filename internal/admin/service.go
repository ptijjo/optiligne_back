package admin

import (
	"context"
	"math"
	"sort"
	"strings"

	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	guidancedto "github.com/ptijjo/optiligne_back/internal/guidance/dto"
	"github.com/ptijjo/optiligne_back/pkg/id"
)

// SessionGuard refuse d'écraser une course guidée en live.
type SessionGuard interface {
	HasActiveTrip(tripID string) bool
}

// Service orchestre édition carte + OSRM + fichiers GTFS.
type Service struct {
	store  Store
	router Router
	files  Files
	guard  SessionGuard
	lockOp string
}

func NewService(store Store, router Router, files Files, guard SessionGuard, lockOp string) *Service {
	return &Service{store: store, router: router, files: files, guard: guard, lockOp: lockOp}
}

func (s *Service) resolve(operatorCode, depotCode string) (string, string, error) {
	if s.lockOp != "" {
		operatorCode = s.lockOp
	}
	if operatorCode == "" || depotCode == "" {
		return "", "", catalog.ErrScopeRequired
	}
	return operatorCode, depotCode, nil
}

func validCoords(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}

// Draft charge shape + arrêts d'une ligne (course optionnelle via tripID).
func (s *Service) Draft(ctx context.Context, operatorCode, depotCode, routeID, tripID string) (*dto.Draft, error) {
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	if routeID == "" {
		return nil, catalog.ErrScopeRequired
	}
	return s.store.LoadDraft(ctx, op, depot, routeID, tripID)
}

// PatchStop déplace un arrêt (PostGIS + stops.txt).
func (s *Service) PatchStop(ctx context.Context, operatorCode, depotCode, routeID, stopID string, lat, lng float64) (*dto.Draft, error) {
	if !validCoords(lat, lng) {
		return nil, ErrInvalidCoords
	}
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	if routeID == "" || stopID == "" {
		return nil, ErrInvalidCoords
	}
	draft, err := s.store.LoadDraft(ctx, op, depot, routeID, "")
	if err != nil {
		return nil, err
	}
	if s.guard != nil && s.guard.HasActiveTrip(draft.TripID) {
		return nil, ErrTripActive
	}
	// 1. PostGIS.
	if err := s.store.UpdateStop(ctx, draft.FeedID, stopID, lat, lng); err != nil {
		return nil, err
	}
	// 2. Fichier GTFS.
	if s.files != nil {
		if err := s.files.PatchStop(stopID, lat, lng); err != nil {
			return nil, err
		}
	}
	if err := s.store.RecomputeFracs(ctx, draft.FeedID, []string{draft.ShapeID}); err != nil {
		return nil, err
	}
	return s.store.LoadDraft(ctx, op, depot, routeID, "")
}

func allowedRouteType(routeType int) bool {
	switch routeType {
	case 204, 712, 713:
		return true
	default:
		return false
	}
}

// PatchRouteType change le type GTFS (PostGIS + routes.txt).
func (s *Service) PatchRouteType(ctx context.Context, operatorCode, depotCode, routeID string, routeType int) (*dto.Draft, error) {
	if !allowedRouteType(routeType) {
		return nil, ErrInvalidRouteType
	}
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	if routeID == "" {
		return nil, catalog.ErrScopeRequired
	}
	draft, err := s.store.LoadDraft(ctx, op, depot, routeID, "")
	if err != nil {
		return nil, err
	}
	if s.guard != nil && s.guard.HasActiveTrip(draft.TripID) {
		return nil, ErrTripActive
	}
	// 1. PostGIS.
	if err := s.store.UpdateRouteType(ctx, draft.FeedID, routeID, routeType); err != nil {
		return nil, err
	}
	// 2. Fichier GTFS.
	if s.files != nil {
		if err := s.files.PatchRouteType(routeID, routeType); err != nil {
			return nil, err
		}
	}
	return s.store.LoadDraft(ctx, op, depot, routeID, "")
}

// Recalculate demande OSRM (preview, pas d'écriture).
func (s *Service) Recalculate(ctx context.Context, operatorCode, depotCode, routeID string, req dto.RecalcRequest) (*dto.RecalcResponse, error) {
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.LoadDraft(ctx, op, depot, routeID, req.TripID); err != nil {
		return nil, err
	}
	coords := viaLngLats(req.Stops, req.Waypoints)
	line, err := s.router.Route(ctx, coords)
	if err != nil {
		return nil, err
	}
	return &dto.RecalcResponse{
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: line},
	}, nil
}

// MatchShape colle le LineString actuel sur OSM (OSRM /match, preview).
func (s *Service) MatchShape(ctx context.Context, operatorCode, depotCode, routeID string, req dto.MatchRequest) (*dto.RecalcResponse, error) {
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	if _, err := s.store.LoadDraft(ctx, op, depot, routeID, req.TripID); err != nil {
		return nil, err
	}
	if len(req.Shape.Coordinates) < 2 {
		return nil, ErrShapeTooShort
	}
	// 1. Lire le LineString tel que dessiné / importé (pas A→B).
	coords := make([][2]float64, 0, len(req.Shape.Coordinates))
	for _, pt := range req.Shape.Coordinates {
		if len(pt) < 2 || !validCoords(pt[1], pt[0]) {
			return nil, ErrInvalidCoords
		}
		coords = append(coords, [2]float64{pt[0], pt[1]})
	}
	// 2. OSRM /match (collage OSM). Le client sous-échantillonne si besoin.
	line, err := s.router.Match(ctx, coords)
	if err != nil {
		return nil, err
	}
	return &dto.RecalcResponse{
		Shape: guidancedto.LineString{Type: "LineString", Coordinates: line},
	}, nil
}

// SearchStops trouve des arrêts du feed actif par nom (ILIKE).
func (s *Service) SearchStops(ctx context.Context, query string, limit int) ([]dto.StopHit, error) {
	q := strings.TrimSpace(query)
	if len(q) < 2 {
		return []dto.StopHit{}, nil
	}
	return s.store.SearchStops(ctx, q, limit)
}

// Save persiste arrêts + shape (PostGIS + GTFS/).
func (s *Service) Save(ctx context.Context, operatorCode, depotCode, routeID string, req dto.SaveRequest) (*dto.SaveResponse, error) {
	op, depot, err := s.resolve(operatorCode, depotCode)
	if err != nil {
		return nil, err
	}
	draft, err := s.store.LoadDraft(ctx, op, depot, routeID, req.TripID)
	if err != nil {
		return nil, err
	}
	if s.guard != nil && s.guard.HasActiveTrip(draft.TripID) {
		return nil, ErrTripActive
	}
	if len(req.Shape.Coordinates) < 2 {
		return nil, ErrShapeTooShort
	}
	stops, err := normalizeStops(req.Stops)
	if err != nil {
		return nil, err
	}
	feedID := draft.FeedID

	// 1. Courses sœurs AVANT mutation de la séquence.
	tripIDs, err := s.store.ListSiblingTripIDs(ctx, feedID, routeID, draft.TripID)
	if err != nil {
		return nil, err
	}
	if len(tripIDs) == 0 {
		tripIDs = []string{draft.TripID}
	}
	shapeIDs, err := s.store.ListSiblingShapeIDs(ctx, feedID, routeID, draft.TripID)
	if err != nil {
		return nil, err
	}
	if len(shapeIDs) == 0 && draft.ShapeID != "" {
		shapeIDs = []string{draft.ShapeID}
	}
	shapeIDs = uniqueNonEmpty(shapeIDs)

	// 2. Upsert arrêts (nouveaux ol-* ou coords mises à jour).
	for _, st := range stops {
		if err := s.store.UpsertStop(ctx, feedID, st); err != nil {
			return nil, err
		}
		if s.files != nil {
			if err := s.files.UpsertStop(st.StopID, st.Name, st.Lat, st.Lng); err != nil {
				return nil, err
			}
		}
	}

	// 3. Remplacer stop_times (+ stop_fracs vides) sur chaque course sœur.
	var fileRows []StopTimeFileRow
	for _, tripID := range tripIDs {
		old, err := s.store.ListTripStopTimes(ctx, feedID, tripID)
		if err != nil {
			return nil, err
		}
		timed := interpolateStopTimes(old, stops)
		if err := s.store.ReplaceTripStopTimes(ctx, feedID, tripID, timed); err != nil {
			return nil, err
		}
		for _, t := range timed {
			fileRows = append(fileRows, StopTimeFileRow{
				TripID: tripID, StopID: t.StopID, StopSequence: t.StopSequence,
				ArrivalSec: t.ArrivalSec, DepartureSec: t.DepartureSec,
			})
		}
	}
	if s.files != nil && len(fileRows) > 0 {
		if err := s.files.ReplaceStopTimes(tripIDs, fileRows); err != nil {
			return nil, err
		}
	}

	// 4. Shapes PostGIS + shapes.txt.
	pts := coordsToPoints(req.Shape.Coordinates)
	for _, shapeID := range shapeIDs {
		if err := s.store.UpdateShape(ctx, feedID, shapeID, pts); err != nil {
			return nil, err
		}
	}
	if s.files != nil && len(shapeIDs) > 0 {
		if err := s.files.ReplaceShapes(shapeIDs, pts); err != nil {
			return nil, err
		}
	}

	// 5. Fractions spatiales.
	if err := s.store.RebuildFracsForTrips(ctx, feedID, tripIDs); err != nil {
		return nil, err
	}

	msg := "Tracé et arrêts enregistrés. Les téléphones du dépôt utiliseront ce circuit au prochain chargement du catalogue."
	if len(tripIDs) > 1 {
		msg = "Tracé et arrêts enregistrés pour toutes les courses du même parcours (tous les jours). Les autres circuits de la ligne sont inchangés."
	}
	return &dto.SaveResponse{
		FeedVersion: draft.FeedVersion,
		Message:     msg,
	}, nil
}

const placeholderStartSec = 7 * 3600
const placeholderStepSec = 5 * 60

// CreateRoute crée une ligne (PostGIS + GTFS/) avec calendrier et courses.
func (s *Service) CreateRoute(ctx context.Context, req dto.CreateRouteRequest) (*dto.CreateRouteResponse, error) {
	op, depot, err := s.resolve(req.OperatorCode, req.DepotCode)
	if err != nil {
		return nil, err
	}
	if !allowedRouteType(req.RouteType) {
		return nil, ErrInvalidRouteType
	}
	shortName := strings.TrimSpace(req.ShortName)
	longName := strings.TrimSpace(req.LongName)
	if shortName == "" || longName == "" {
		return nil, ErrInvalidCoords
	}
	if len(req.Shape.Coordinates) < 2 {
		return nil, ErrShapeTooShort
	}
	stops, err := normalizeStops(req.Stops)
	if err != nil {
		return nil, err
	}
	for _, pt := range req.Shape.Coordinates {
		if len(pt) < 2 || !validCoords(pt[1], pt[0]) {
			return nil, ErrInvalidCoords
		}
	}
	cal, err := normalizeCalendar(req.Calendar)
	if err != nil {
		return nil, err
	}
	tripReqs := req.Trips
	if len(tripReqs) == 0 {
		// Compat : une course placeholders si pas d'horaires fournis.
		secs := make([]int, len(stops))
		for i := range stops {
			secs[i] = placeholderStartSec + i*placeholderStepSec
		}
		tripReqs = []dto.CreateTripTimes{{Headsign: longName, ArrivalSecs: secs}}
	}
	trips, err := normalizeTrips(tripReqs, stops, longName)
	if err != nil {
		return nil, err
	}
	scope, err := s.store.ResolveScope(ctx, op, depot)
	if err != nil {
		return nil, err
	}

	routeID := "ol-r-" + id.New()
	shapeID := "ol-s-" + id.New()
	serviceID := "ol-c-" + id.New()
	pts := coordsToPoints(req.Shape.Coordinates)
	tripInputs := make([]TripInput, 0, len(trips))
	tripIDs := make([]string, 0, len(trips))
	for _, tr := range trips {
		tid := "ol-t-" + id.New()
		tripIDs = append(tripIDs, tid)
		tripInputs = append(tripInputs, TripInput{TripID: tid, Headsign: tr.Headsign, Timed: tr.Timed})
	}
	in := CreateRouteInput{
		FeedID: scope.FeedID, OperatorID: scope.OperatorID, DepotID: scope.DepotID,
		AgencyID: scope.AgencyID,
		RouteID: routeID, ShortName: shortName, LongName: longName, RouteType: req.RouteType,
		ShapeID: shapeID, Stops: stops, ShapePoints: pts,
		Calendar: CalendarInput{
			ServiceID: serviceID,
			Monday: cal.Monday, Tuesday: cal.Tuesday, Wednesday: cal.Wednesday,
			Thursday: cal.Thursday, Friday: cal.Friday, Saturday: cal.Saturday, Sunday: cal.Sunday,
			StartDate: cal.StartDate, EndDate: cal.EndDate,
		},
		Trips: tripInputs,
	}
	if err := s.store.CreateRoute(ctx, in); err != nil {
		return nil, err
	}
	if err := s.store.RebuildFracsForTrips(ctx, scope.FeedID, tripIDs); err != nil {
		return nil, err
	}
	if s.files != nil {
		if err := s.files.UpsertRoute(routeID, scope.AgencyID, shortName, longName, req.RouteType); err != nil {
			return nil, err
		}
		if err := s.files.UpsertCalendar(CalendarFileRow{
			ServiceID: serviceID,
			Monday: cal.Monday, Tuesday: cal.Tuesday, Wednesday: cal.Wednesday,
			Thursday: cal.Thursday, Friday: cal.Friday, Saturday: cal.Saturday, Sunday: cal.Sunday,
			StartDate: cal.StartDate, EndDate: cal.EndDate,
		}); err != nil {
			return nil, err
		}
		for _, st := range stops {
			if err := s.files.UpsertStop(st.StopID, st.Name, st.Lat, st.Lng); err != nil {
				return nil, err
			}
		}
		if err := s.files.ReplaceShapes([]string{shapeID}, pts); err != nil {
			return nil, err
		}
		var rows []StopTimeFileRow
		for _, tr := range tripInputs {
			if err := s.files.UpsertTrip(tr.TripID, routeID, serviceID, shapeID, tr.Headsign); err != nil {
				return nil, err
			}
			for _, t := range tr.Timed {
				rows = append(rows, StopTimeFileRow{
					TripID: tr.TripID, StopID: t.StopID, StopSequence: t.StopSequence,
					ArrivalSec: t.ArrivalSec, DepartureSec: t.DepartureSec,
				})
			}
		}
		if err := s.files.ReplaceStopTimes(tripIDs, rows); err != nil {
			return nil, err
		}
	}
	return &dto.CreateRouteResponse{
		RouteID: routeID, TripID: tripIDs[0], FeedVersion: scope.FeedVersion,
		Message: "Ligne créée. Les téléphones du dépôt la verront au prochain chargement du catalogue.",
	}, nil
}

func normalizeCalendar(c dto.CreateCalendar) (dto.CreateCalendar, error) {
	start, err := normalizeGTFSDate(c.StartDate)
	if err != nil {
		return c, ErrInvalidCalendar
	}
	end, err := normalizeGTFSDate(c.EndDate)
	if err != nil {
		return c, ErrInvalidCalendar
	}
	if start > end {
		return c, ErrInvalidCalendar
	}
	anyDay := c.Monday || c.Tuesday || c.Wednesday || c.Thursday || c.Friday || c.Saturday || c.Sunday
	if !anyDay {
		return c, ErrInvalidCalendar
	}
	c.StartDate = start
	c.EndDate = end
	return c, nil
}

func normalizeGTFSDate(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "-", "")
	if len(s) != 8 {
		return "", ErrInvalidCalendar
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return "", ErrInvalidCalendar
		}
	}
	return s, nil
}

type normalizedTrip struct {
	Headsign string
	Timed    []dto.TimedStop
}

func normalizeTrips(trips []dto.CreateTripTimes, stops []dto.EditorStop, defaultHeadsign string) ([]normalizedTrip, error) {
	if len(trips) == 0 {
		return nil, ErrInvalidSchedule
	}
	out := make([]normalizedTrip, 0, len(trips))
	for _, tr := range trips {
		if len(tr.ArrivalSecs) != len(stops) {
			return nil, ErrInvalidSchedule
		}
		timed := make([]dto.TimedStop, len(stops))
		prev := -1
		for i, st := range stops {
			sec := tr.ArrivalSecs[i]
			if sec < 0 || sec > 48*3600 {
				return nil, ErrInvalidSchedule
			}
			if sec < prev {
				return nil, ErrInvalidSchedule
			}
			prev = sec
			timed[i] = dto.TimedStop{
				StopID: st.StopID, StopSequence: i + 1,
				ArrivalSec: sec, DepartureSec: sec,
				Name: st.Name, Lat: st.Lat, Lng: st.Lng,
			}
		}
		hs := strings.TrimSpace(tr.Headsign)
		if hs == "" {
			hs = defaultHeadsign
		}
		out = append(out, normalizedTrip{Headsign: hs, Timed: timed})
	}
	return out, nil
}

func viaLngLats(stops []dto.EditorStop, wps []dto.Waypoint) [][2]float64 {
	if len(stops) == 0 {
		out := make([][2]float64, 0, len(wps))
		for _, w := range wps {
			out = append(out, [2]float64{w.Lng, w.Lat})
		}
		return out
	}
	type placed struct {
		seg   int
		t     float64
		order int
		lng   float64
		lat   float64
	}
	placedWps := make([]placed, 0, len(wps))
	nseg := len(stops) - 1
	for i, w := range wps {
		bestSeg, bestT, bestD := 0, 0.0, math.MaxFloat64
		if nseg <= 0 {
			placedWps = append(placedWps, placed{seg: 0, t: 0, order: i, lng: w.Lng, lat: w.Lat})
			continue
		}
		if forced, ok := segmentAfterStop(stops, w.AfterStopID); ok {
			t, _ := projectOnSegment(stops[forced].Lng, stops[forced].Lat, stops[forced+1].Lng, stops[forced+1].Lat, w.Lng, w.Lat)
			placedWps = append(placedWps, placed{seg: forced, t: t, order: i, lng: w.Lng, lat: w.Lat})
			continue
		}
		for s := 0; s < nseg; s++ {
			t, d2 := projectOnSegment(stops[s].Lng, stops[s].Lat, stops[s+1].Lng, stops[s+1].Lat, w.Lng, w.Lat)
			if d2 < bestD {
				bestD = d2
				bestSeg = s
				bestT = t
			}
		}
		placedWps = append(placedWps, placed{seg: bestSeg, t: bestT, order: i, lng: w.Lng, lat: w.Lat})
	}
	sort.SliceStable(placedWps, func(i, j int) bool {
		if placedWps[i].seg != placedWps[j].seg {
			return placedWps[i].seg < placedWps[j].seg
		}
		if placedWps[i].t != placedWps[j].t {
			return placedWps[i].t < placedWps[j].t
		}
		return placedWps[i].order < placedWps[j].order
	})
	out := make([][2]float64, 0, len(stops)+len(wps))
	pi := 0
	for s, st := range stops {
		out = append(out, [2]float64{st.Lng, st.Lat})
		for pi < len(placedWps) && placedWps[pi].seg == s {
			out = append(out, [2]float64{placedWps[pi].lng, placedWps[pi].lat})
			pi++
		}
	}
	return out
}

func segmentAfterStop(stops []dto.EditorStop, afterStopID string) (int, bool) {
	if afterStopID == "" || len(stops) < 2 {
		return 0, false
	}
	lastSeg := len(stops) - 2
	for i, st := range stops {
		if st.StopID != afterStopID {
			continue
		}
		if i > lastSeg {
			return lastSeg, true
		}
		return i, true
	}
	return 0, false
}

func projectOnSegment(ax, ay, bx, by, x, y float64) (t, d2 float64) {
	abx := bx - ax
	aby := by - ay
	len2 := abx*abx + aby*aby
	if len2 == 0 {
		dx := x - ax
		dy := y - ay
		return 0, dx*dx + dy*dy
	}
	t = ((x-ax)*abx + (y-ay)*aby) / len2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	px := ax + t*abx
	py := ay + t*aby
	dx := x - px
	dy := y - py
	return t, dx*dx + dy*dy
}

func uniqueNonEmpty(ids []string) []string {
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
