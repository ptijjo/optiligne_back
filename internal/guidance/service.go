package guidance

import (
	"context"
	"sync"
	"time"

	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/internal/geo"
	"github.com/ptijjo/optiligne_back/internal/guidance/dto"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"gorm.io/gorm"
)

// Session est une session de guidage en mémoire.
type Session struct {
	ID              string
	TripID          string
	FeedVersionID   string
	ShapeID         string
	PrevFrac        float64
	ServiceMidnight time.Time
	Stops           []StopProg
	ShapeLengthM    float64
}

// Service orchestre périmètre + snap + Evaluate.
type Service struct {
	db       *gorm.DB
	clock    Clock
	offRoute float64
	lockOp   string
	mu       sync.Mutex
	sessions map[string]*Session
}

func NewService(db *gorm.DB, clock Clock, offRoute float64, lockOp string) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	if offRoute <= 0 {
		offRoute = 80
	}
	return &Service{db: db, clock: clock, offRoute: offRoute, lockOp: lockOp, sessions: map[string]*Session{}}
}

// Start crée une session si la course est dans le périmètre scolaire.
func (s *Service) Start(ctx context.Context, operatorCode, depotCode, tripID, date string) (*dto.StartResponse, error) {
	if s.lockOp != "" {
		operatorCode = s.lockOp
	}
	if operatorCode == "" || depotCode == "" || tripID == "" || date == "" {
		return nil, catalog.ErrScopeRequired
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, catalog.ErrTripNotFound
	}

	var row struct {
		FeedVersionID string
		ShapeID       string
		RouteType     int
		InScope       bool
	}
	err = s.db.WithContext(ctx).Raw(`
		SELECT t.feed_version_id, t.shape_id, r.route_type,
		  EXISTS (
		    SELECT 1 FROM route_assignments a
		    JOIN operators o ON o.id = a.operator_id
		    JOIN depots d ON d.id = a.depot_id
		    WHERE a.feed_version_id = t.feed_version_id AND a.route_id = t.route_id
		      AND o.code = ? AND d.code = ?
		  ) AS in_scope
		FROM trips t
		JOIN routes r ON r.route_id = t.route_id AND r.feed_version_id = t.feed_version_id
		JOIN feed_versions fv ON fv.id = t.feed_version_id AND fv.active = true
		WHERE t.trip_id = ?
	`, operatorCode, depotCode, tripID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.FeedVersionID == "" {
		return nil, catalog.ErrTripNotFound
	}
	if err := catalog.EnsureSchoolTrip(row.RouteType, row.InScope); err != nil {
		return nil, err
	}

	mapStops := s.loadStopPoints(ctx, row.FeedVersionID, tripID)
	stops := s.buildStopProg(ctx, row.FeedVersionID, tripID, mapStops)
	shapeLen := geo.ShapeLengthM(ctx, s.db, row.FeedVersionID, row.ShapeID)

	sess := &Session{
		ID:              id.New(),
		TripID:          tripID,
		FeedVersionID:   row.FeedVersionID,
		ShapeID:         row.ShapeID,
		ServiceMidnight: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location()),
		Stops:           stops,
		ShapeLengthM:    shapeLen,
	}
	s.mu.Lock()
	s.sessions[sess.ID] = sess
	s.mu.Unlock()

	shape := s.loadShape(ctx, row.FeedVersionID, row.ShapeID)
	return &dto.StartResponse{
		SessionID: sess.ID,
		TripID:    tripID,
		Shape:     shape,
		Stops:     mapStops,
	}, nil
}

func (s *Service) buildStopProg(ctx context.Context, feedVersionID, tripID string, mapStops []dto.StopPoint) []StopProg {
	var fracs []models.StopFrac
	_ = s.db.WithContext(ctx).Where("feed_version_id = ? AND trip_id = ?", feedVersionID, tripID).
		Order("stop_sequence").Find(&fracs).Error

	coordsBySeq := map[int]dto.StopPoint{}
	for _, m := range mapStops {
		coordsBySeq[m.Sequence] = m
	}

	if len(fracs) > 0 {
		out := make([]StopProg, 0, len(fracs))
		for _, f := range fracs {
			sp := StopProg{
				Name: f.StopName, Frac: f.Frac, ArrivalSec: f.ArrivalSec, Sequence: f.StopSequence,
			}
			if c, ok := coordsBySeq[f.StopSequence]; ok {
				sp.Lon, sp.Lat = c.Lon, c.Lat
				if sp.Name == "" {
					sp.Name = c.Name
				}
			}
			out = append(out, sp)
		}
		return out
	}

	// Fallback : pas de fracs → progression linéaire par index.
	out := make([]StopProg, 0, len(mapStops))
	n := len(mapStops)
	for i, m := range mapStops {
		frac := 0.0
		if n > 1 {
			frac = float64(i) / float64(n-1)
		}
		out = append(out, StopProg{
			Name: m.Name, Frac: frac, ArrivalSec: m.ArrivalSec, Sequence: m.Sequence,
			Lon: m.Lon, Lat: m.Lat,
		})
	}
	return out
}

func (s *Service) loadShape(ctx context.Context, feedVersionID, shapeID string) dto.LineString {
	if shapeID == "" {
		return dto.EmptyLineString()
	}
	var raw string
	err := s.db.WithContext(ctx).Raw(
		`SELECT ST_AsGeoJSON(geom) FROM shapes WHERE feed_version_id = ? AND shape_id = ?`,
		feedVersionID, shapeID,
	).Scan(&raw).Error
	if err != nil || raw == "" {
		return dto.EmptyLineString()
	}
	return dto.DecodeLineString(raw)
}

func (s *Service) loadStopPoints(ctx context.Context, feedVersionID, tripID string) []dto.StopPoint {
	type row struct {
		Name       string
		Lon        float64
		Lat        float64
		ArrivalSec int
		Sequence   int
	}
	var rows []row
	// 1. Préférer stop_fracs (même ordre que le guidage).
	_ = s.db.WithContext(ctx).Raw(`
		SELECT COALESCE(NULLIF(sf.stop_name, ''), s.name) AS name, s.lon, s.lat,
			sf.arrival_sec AS arrival_sec, sf.stop_sequence AS sequence
		FROM stop_fracs sf
		JOIN stops s ON s.stop_id = sf.stop_id AND s.feed_version_id = sf.feed_version_id
		WHERE sf.feed_version_id = ? AND sf.trip_id = ?
		ORDER BY sf.stop_sequence
	`, feedVersionID, tripID).Scan(&rows).Error
	if len(rows) == 0 {
		// 2. Fallback stop_times si les fractions manquent.
		_ = s.db.WithContext(ctx).Raw(`
			SELECT s.name, s.lon, s.lat,
				st.arrival_sec AS arrival_sec, st.stop_sequence AS sequence
			FROM stop_times st
			JOIN stops s ON s.stop_id = st.stop_id AND s.feed_version_id = st.feed_version_id
			WHERE st.feed_version_id = ? AND st.trip_id = ?
			ORDER BY st.stop_sequence
		`, feedVersionID, tripID).Scan(&rows).Error
	}
	out := make([]dto.StopPoint, 0, len(rows))
	for _, r := range rows {
		if r.Lat == 0 && r.Lon == 0 {
			continue
		}
		out = append(out, dto.StopPoint{
			Name: r.Name, Lon: r.Lon, Lat: r.Lat,
			ArrivalSec: r.ArrivalSec, Sequence: r.Sequence,
		})
	}
	return out
}

func remainingMeters(sess *Session, frac, offsetM, offRouteM, lon, lat float64) float64 {
	if sess == nil || len(sess.Stops) == 0 {
		return 0
	}
	offRoute := offRouteM > 0 && offsetM > offRouteM
	first := sess.Stops[0]

	if offRoute {
		if first.Lat == 0 && first.Lon == 0 {
			return 0
		}
		return geo.DistanceMeters(lon, lat, first.Lon, first.Lat)
	}

	var targetFrac float64
	found := false
	for _, st := range sess.Stops {
		if st.Frac > frac {
			targetFrac = st.Frac
			found = true
			break
		}
	}
	if !found {
		return 0
	}
	delta := targetFrac - frac
	if delta < 0 {
		delta = 0
	}
	return delta * sess.ShapeLengthM
}

// Update applique une position GPS.
func (s *Service) Update(ctx context.Context, sessionID string, lat, lon float64) (*dto.Guidance, error) {
	if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return nil, ErrInvalidPosition
	}
	s.mu.Lock()
	sess := s.sessions[sessionID]
	s.mu.Unlock()
	if sess == nil {
		return nil, catalog.ErrTripNotFound
	}
	frac, offset, err := geo.Snap(ctx, s.db, sess.FeedVersionID, sess.ShapeID, lon, lat)
	if err != nil {
		return nil, err
	}
	remaining := remainingMeters(sess, frac, offset, s.offRoute, lon, lat)
	out := Evaluate(Input{
		Frac: frac, OffsetM: offset, PrevFrac: sess.PrevFrac, Stops: sess.Stops,
		Now: s.clock.Now(), ServiceMidnight: sess.ServiceMidnight, OffRouteM: s.offRoute,
		RemainingM: remaining,
	})
	if out.State != "ambiguous" {
		s.mu.Lock()
		sess.PrevFrac = out.Frac
		s.mu.Unlock()
	}
	return &dto.Guidance{
		Frac: out.Frac, OffsetM: out.OffsetM, NextStop: out.NextStop,
		DelayS: out.DelayS, TravelS: out.TravelS, State: out.State,
	}, nil
}

// SessionExists indique si la session est connue.
func (s *Service) SessionExists(sessionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.sessions[sessionID]
	return ok
}

// HasActiveTrip est vrai si une session de guidage utilise cette course.
func (s *Service) HasActiveTrip(tripID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, sess := range s.sessions {
		if sess.TripID == tripID {
			return true
		}
	}
	return false
}
