package admin

import (
	"context"

	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	guidancedto "github.com/ptijjo/optiligne_back/internal/guidance/dto"
	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"gorm.io/gorm"
)

type draftRow struct {
	RouteID     string
	ShortName   string
	LongName    string
	RouteType   int
	TripID      string
	ShapeID     string
	FeedID      string
	FeedVersion string
}

// Store lit / écrit le brouillon éditeur.
type Store interface {
	LoadDraft(ctx context.Context, operatorCode, depotCode, routeID, tripID string) (*dto.Draft, error)
	SearchStops(ctx context.Context, query string, limit int) ([]dto.StopHit, error)
	UpdateStop(ctx context.Context, feedID, stopID string, lat, lng float64) error
	UpdateRouteType(ctx context.Context, feedID, routeID string, routeType int) error
	UpsertStop(ctx context.Context, feedID string, stop dto.EditorStop) error
	UpdateShape(ctx context.Context, feedID, shapeID string, pts []gtfs.ShapePoint) error
	ListSiblingShapeIDs(ctx context.Context, feedID, routeID, tripID string) ([]string, error)
	ListSiblingTripIDs(ctx context.Context, feedID, routeID, tripID string) ([]string, error)
	ListTripStopTimes(ctx context.Context, feedID, tripID string) ([]dto.TimedStop, error)
	ReplaceTripStopTimes(ctx context.Context, feedID, tripID string, stops []dto.TimedStop) error
	RebuildFracsForTrips(ctx context.Context, feedID string, tripIDs []string) error
	RecomputeFracs(ctx context.Context, feedID string, shapeIDs []string) error
}

// Repository GORM + PostGIS.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) LoadDraft(ctx context.Context, operatorCode, depotCode, routeID, tripID string) (*dto.Draft, error) {
	var row draftRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.route_id, r.short_name, r.long_name, r.route_type, t.trip_id, t.shape_id,
		       fv.id AS feed_id, fv.feed_version
		FROM routes r
		JOIN route_assignments a ON a.route_id = r.route_id AND a.feed_version_id = r.feed_version_id
		JOIN operators o ON o.id = a.operator_id
		JOIN depots d ON d.id = a.depot_id
		JOIN feed_versions fv ON fv.id = r.feed_version_id AND fv.active = true
		JOIN trips t ON t.route_id = r.route_id AND t.feed_version_id = r.feed_version_id
		WHERE o.code = ? AND d.code = ? AND r.route_id = ?
		  AND (? = '' OR t.trip_id = ?)
		ORDER BY t.trip_id
		LIMIT 1
	`, operatorCode, depotCode, routeID, tripID, tripID).Scan(&row).Error
	if err != nil {
		return nil, err
	}
	if row.TripID == "" {
		return nil, catalog.ErrRouteNotFound
	}

	var raw string
	_ = r.db.WithContext(ctx).Raw(
		`SELECT ST_AsGeoJSON(geom) FROM shapes WHERE feed_version_id = ? AND shape_id = ?`,
		row.FeedID, row.ShapeID,
	).Scan(&raw).Error

	type stopRow struct {
		StopID       string
		Name         string
		StopSequence int
		Lat          float64
		Lon          float64
	}
	var stops []stopRow
	if err := r.db.WithContext(ctx).Raw(`
		SELECT st.stop_id, s.name, st.stop_sequence, s.lat, s.lon
		FROM stop_times st
		JOIN stops s ON s.stop_id = st.stop_id AND s.feed_version_id = st.feed_version_id
		WHERE st.feed_version_id = ? AND st.trip_id = ?
		ORDER BY st.stop_sequence ASC
	`, row.FeedID, row.TripID).Scan(&stops).Error; err != nil {
		return nil, err
	}
	outStops := make([]dto.EditorStop, 0, len(stops))
	for _, s := range stops {
		outStops = append(outStops, dto.EditorStop{
			StopID: s.StopID, Name: s.Name, Sequence: s.StopSequence, Lat: s.Lat, Lng: s.Lon,
		})
	}
	return &dto.Draft{
		RouteID: row.RouteID, ShortName: row.ShortName, LongName: row.LongName, RouteType: row.RouteType,
		TripID: row.TripID, ShapeID: row.ShapeID, FeedID: row.FeedID, FeedVersion: row.FeedVersion,
		Shape: guidancedto.DecodeLineString(raw), Stops: outStops,
	}, nil
}

func (r *Repository) UpdateRouteType(ctx context.Context, feedID, routeID string, routeType int) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE routes SET route_type = ? WHERE feed_version_id = ? AND route_id = ?`,
		routeType, feedID, routeID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return catalog.ErrRouteNotFound
	}
	return nil
}

func (r *Repository) SearchStops(ctx context.Context, query string, limit int) ([]dto.StopHit, error) {
	if limit <= 0 || limit > 20 {
		limit = 20
	}
	q := "%" + query + "%"
	var rows []dto.StopHit
	err := r.db.WithContext(ctx).Raw(`
		SELECT s.stop_id, s.name, s.lat, s.lon AS lng
		FROM stops s
		JOIN feed_versions fv ON fv.id = s.feed_version_id AND fv.active = true
		WHERE s.name ILIKE ?
		ORDER BY s.name ASC
		LIMIT ?
	`, q, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	if rows == nil {
		rows = []dto.StopHit{}
	}
	return rows, nil
}

func (r *Repository) UpdateStop(ctx context.Context, feedID, stopID string, lat, lng float64) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE stops SET lat = ?, lon = ? WHERE feed_version_id = ? AND stop_id = ?`,
		lat, lng, feedID, stopID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return catalog.ErrRouteNotFound
	}
	return nil
}

func (r *Repository) UpsertStop(ctx context.Context, feedID string, stop dto.EditorStop) error {
	res := r.db.WithContext(ctx).Exec(
		`UPDATE stops SET lat = ?, lon = ?, name = ? WHERE feed_version_id = ? AND stop_id = ?`,
		stop.Lat, stop.Lng, stop.Name, feedID, stop.StopID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected > 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(&models.Stop{
		ID: id.New(), FeedVersionID: feedID, StopID: stop.StopID,
		Name: stop.Name, Lat: stop.Lat, Lon: stop.Lng,
	}).Error
}

func (r *Repository) UpdateShape(ctx context.Context, feedID, shapeID string, pts []gtfs.ShapePoint) error {
	if len(pts) < 2 {
		return ErrShapeTooShort
	}
	wkt := gtfs.LineStringWKT(pts)
	return r.db.WithContext(ctx).Exec(
		`UPDATE shapes SET geom = ST_SetSRID(ST_GeomFromText(?), 4326) WHERE feed_version_id = ? AND shape_id = ?`,
		wkt, feedID, shapeID,
	).Error
}

func (r *Repository) ListSiblingShapeIDs(ctx context.Context, feedID, routeID, tripID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
		WITH pattern AS (
			SELECT string_agg(stop_id, ',' ORDER BY stop_sequence) AS seq
			FROM stop_times
			WHERE feed_version_id = ? AND trip_id = ?
		),
		siblings AS (
			SELECT st.trip_id
			FROM stop_times st
			JOIN trips t ON t.trip_id = st.trip_id AND t.feed_version_id = st.feed_version_id
			WHERE st.feed_version_id = ? AND t.route_id = ?
			GROUP BY st.trip_id
			HAVING string_agg(st.stop_id, ',' ORDER BY st.stop_sequence) = (SELECT seq FROM pattern)
		)
		SELECT DISTINCT t.shape_id
		FROM trips t
		JOIN siblings s ON s.trip_id = t.trip_id
		WHERE t.feed_version_id = ?
		  AND t.shape_id IS NOT NULL AND t.shape_id <> ''
		ORDER BY t.shape_id
	`, feedID, tripID, feedID, routeID, feedID).Scan(&ids).Error
	return ids, err
}

func (r *Repository) ListSiblingTripIDs(ctx context.Context, feedID, routeID, tripID string) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).Raw(`
		WITH pattern AS (
			SELECT string_agg(stop_id, ',' ORDER BY stop_sequence) AS seq
			FROM stop_times
			WHERE feed_version_id = ? AND trip_id = ?
		)
		SELECT st.trip_id
		FROM stop_times st
		JOIN trips t ON t.trip_id = st.trip_id AND t.feed_version_id = st.feed_version_id
		WHERE st.feed_version_id = ? AND t.route_id = ?
		GROUP BY st.trip_id
		HAVING string_agg(st.stop_id, ',' ORDER BY st.stop_sequence) = (SELECT seq FROM pattern)
		ORDER BY st.trip_id
	`, feedID, tripID, feedID, routeID).Scan(&ids).Error
	return ids, err
}

func (r *Repository) ListTripStopTimes(ctx context.Context, feedID, tripID string) ([]dto.TimedStop, error) {
	var rows []dto.TimedStop
	err := r.db.WithContext(ctx).Raw(`
		SELECT st.stop_id, st.stop_sequence, st.arrival_sec, st.departure_sec,
		       COALESCE(s.name, '') AS name, COALESCE(s.lat, 0) AS lat, COALESCE(s.lon, 0) AS lng
		FROM stop_times st
		LEFT JOIN stops s ON s.stop_id = st.stop_id AND s.feed_version_id = st.feed_version_id
		WHERE st.feed_version_id = ? AND st.trip_id = ?
		ORDER BY st.stop_sequence ASC
	`, feedID, tripID).Scan(&rows).Error
	return rows, err
}

func (r *Repository) ReplaceTripStopTimes(ctx context.Context, feedID, tripID string, stops []dto.TimedStop) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`DELETE FROM stop_times WHERE feed_version_id = ? AND trip_id = ?`, feedID, tripID).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DELETE FROM stop_fracs WHERE feed_version_id = ? AND trip_id = ?`, feedID, tripID).Error; err != nil {
			return err
		}
		for _, st := range stops {
			row := models.StopTime{
				ID: id.New(), FeedVersionID: feedID, TripID: tripID,
				StopID: st.StopID, StopSequence: st.StopSequence,
				ArrivalSec: st.ArrivalSec, DepartureSec: st.DepartureSec,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
			frac := models.StopFrac{
				ID: id.New(), FeedVersionID: feedID, TripID: tripID,
				StopID: st.StopID, StopSequence: st.StopSequence,
				Frac: 0, ArrivalSec: st.ArrivalSec, StopName: st.Name,
			}
			if err := tx.Create(&frac).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) RebuildFracsForTrips(ctx context.Context, feedID string, tripIDs []string) error {
	if len(tripIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE stop_fracs AS sf
		SET
			frac = COALESCE(ST_LineLocatePoint(sh.geom, ST_SetSRID(ST_MakePoint(s.lon, s.lat), 4326)), sf.frac),
			stop_name = COALESCE(s.name, sf.stop_name)
		FROM trips t
		JOIN shapes sh ON sh.shape_id = t.shape_id AND sh.feed_version_id = t.feed_version_id
		JOIN stops s ON s.stop_id = sf.stop_id AND s.feed_version_id = sf.feed_version_id
		WHERE sf.feed_version_id = ?
		  AND sf.trip_id = t.trip_id AND t.feed_version_id = sf.feed_version_id
		  AND sf.trip_id IN ?
	`, feedID, tripIDs).Error
}

func (r *Repository) RecomputeFracs(ctx context.Context, feedID string, shapeIDs []string) error {
	if len(shapeIDs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Exec(`
		UPDATE stop_fracs AS sf
		SET frac = computed.frac
		FROM (
			SELECT
				sf2.id AS id,
				COALESCE(
					ST_LineLocatePoint(sh.geom, ST_SetSRID(ST_MakePoint(s.lon, s.lat), 4326)),
					sf2.frac
				) AS frac
			FROM stop_fracs sf2
			JOIN trips t ON t.trip_id = sf2.trip_id AND t.feed_version_id = sf2.feed_version_id
			JOIN shapes sh ON sh.shape_id = t.shape_id AND sh.feed_version_id = t.feed_version_id
			JOIN stops s ON s.stop_id = sf2.stop_id AND s.feed_version_id = sf2.feed_version_id
			WHERE sf2.feed_version_id = ? AND t.shape_id IN ?
		) AS computed
		WHERE sf.id = computed.id
	`, feedID, shapeIDs).Error
}
