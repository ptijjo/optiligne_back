package catalog

import (
	"context"
	"sort"
	"time"

	"github.com/ptijjo/optiligne_back/internal/catalog/dto"
	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/models"
	"gorm.io/gorm"
)

// Repository lit le catalogue en base.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) SchoolRoutes(ctx context.Context, operatorCode, depotCode string) ([]dto.Route, error) {
	var rows []models.Route
	err := r.db.WithContext(ctx).Raw(`
		SELECT r.id, r.route_id, r.short_name, r.long_name, r.route_type
		FROM routes r
		JOIN route_assignments a ON a.route_id = r.route_id AND a.feed_version_id = r.feed_version_id
		JOIN operators o ON o.id = a.operator_id
		JOIN depots d ON d.id = a.depot_id
		JOIN feed_versions fv ON fv.id = r.feed_version_id AND fv.active = true
		WHERE o.code = ? AND d.code = ?
		ORDER BY r.short_name
	`, operatorCode, depotCode).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]dto.Route, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.Route{
			ID:        row.RouteID,
			ShortName: row.ShortName,
			LongName:  row.LongName,
			RouteType: row.RouteType,
		})
	}
	return out, nil
}

func (r *Repository) TripsOnDate(ctx context.Context, operatorCode, depotCode, routeID string, day time.Time) ([]dto.Trip, error) {
	var routes []models.Route
	if err := r.db.WithContext(ctx).Raw(`
		SELECT r.*
		FROM routes r
		JOIN route_assignments a ON a.route_id = r.route_id AND a.feed_version_id = r.feed_version_id
		JOIN operators o ON o.id = a.operator_id
		JOIN depots d ON d.id = a.depot_id
		JOIN feed_versions fv ON fv.id = r.feed_version_id AND fv.active = true
		WHERE o.code = ? AND d.code = ? AND r.route_id = ?
	`, operatorCode, depotCode, routeID).Scan(&routes).Error; err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, ErrRouteNotFound
	}
	rt := routes[0]

	var trips []models.Trip
	if err := r.db.WithContext(ctx).Where("feed_version_id = ? AND route_id = ?", rt.FeedVersionID, routeID).Find(&trips).Error; err != nil {
		return nil, err
	}
	var cals []models.Calendar
	_ = r.db.WithContext(ctx).Where("feed_version_id = ?", rt.FeedVersionID).Find(&cals).Error
	var dates []models.CalendarDate
	_ = r.db.WithContext(ctx).Where("feed_version_id = ?", rt.FeedVersionID).Find(&dates).Error

	deps, err := r.firstDepartures(ctx, rt.FeedVersionID, routeID)
	if err != nil {
		return nil, err
	}

	gc := toGTFSCalendars(cals)
	gd := toGTFSDates(dates)
	out := make([]dto.Trip, 0)
	for _, tr := range trips {
		if gtfs.ServiceActive(gc, gd, tr.ServiceID, day) {
			out = append(out, dto.Trip{
				ID:           tr.TripID,
				Headsign:     tr.Headsign,
				RouteID:      tr.RouteID,
				DepartureSec: deps[tr.TripID],
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].DepartureSec == out[j].DepartureSec {
			return out[i].ID < out[j].ID
		}
		return out[i].DepartureSec < out[j].DepartureSec
	})
	return out, nil
}

func (r *Repository) TripStops(ctx context.Context, operatorCode, depotCode, tripID string) ([]dto.Stop, error) {
	var scoped int
	if err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM trips t
		JOIN routes rt ON rt.route_id = t.route_id AND rt.feed_version_id = t.feed_version_id
		JOIN route_assignments a ON a.route_id = t.route_id AND a.feed_version_id = t.feed_version_id
		JOIN operators o ON o.id = a.operator_id
		JOIN depots d ON d.id = a.depot_id
		JOIN feed_versions fv ON fv.id = t.feed_version_id AND fv.active = true
		WHERE o.code = ? AND d.code = ? AND t.trip_id = ?
	`, operatorCode, depotCode, tripID).Scan(&scoped).Error; err != nil {
		return nil, err
	}
	if scoped == 0 {
		return nil, ErrTripNotFound
	}

	type row struct {
		StopID       string
		Name         string
		StopSequence int
		ArrivalSec   int
		DepartureSec int
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT st.stop_id, s.name, st.stop_sequence, st.arrival_sec, st.departure_sec
		FROM stop_times st
		JOIN trips t ON t.trip_id = st.trip_id AND t.feed_version_id = st.feed_version_id
		JOIN stops s ON s.stop_id = st.stop_id AND s.feed_version_id = st.feed_version_id
		JOIN feed_versions fv ON fv.id = st.feed_version_id AND fv.active = true
		WHERE st.trip_id = ?
		ORDER BY st.stop_sequence ASC
	`, tripID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]dto.Stop, 0, len(rows))
	for _, row := range rows {
		out = append(out, dto.Stop{
			StopID:       row.StopID,
			Name:         row.Name,
			Sequence:     row.StopSequence,
			ArrivalSec:   row.ArrivalSec,
			DepartureSec: row.DepartureSec,
		})
	}
	return out, nil
}

func (r *Repository) firstDepartures(ctx context.Context, feedVersionID, routeID string) (map[string]int, error) {
	type row struct {
		TripID       string
		DepartureSec int
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT DISTINCT ON (st.trip_id) st.trip_id, st.departure_sec
		FROM stop_times st
		JOIN trips t ON t.trip_id = st.trip_id AND t.feed_version_id = st.feed_version_id
		WHERE st.feed_version_id = ? AND t.route_id = ?
		ORDER BY st.trip_id, st.stop_sequence ASC
	`, feedVersionID, routeID).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.TripID] = row.DepartureSec
	}
	return out, nil
}

func toGTFSCalendars(in []models.Calendar) []gtfs.Calendar {
	out := make([]gtfs.Calendar, 0, len(in))
	for _, c := range in {
		out = append(out, gtfs.Calendar{
			ServiceID: c.ServiceID, Monday: c.Monday, Tuesday: c.Tuesday,
			Wednesday: c.Wednesday, Thursday: c.Thursday, Friday: c.Friday,
			Saturday: c.Saturday, Sunday: c.Sunday,
			StartDate: c.StartDate, EndDate: c.EndDate,
		})
	}
	return out
}

func toGTFSDates(in []models.CalendarDate) []gtfs.CalendarDate {
	out := make([]gtfs.CalendarDate, 0, len(in))
	for _, d := range in {
		out = append(out, gtfs.CalendarDate{ServiceID: d.ServiceID, Date: d.Date, ExceptionType: d.ExceptionType})
	}
	return out
}
