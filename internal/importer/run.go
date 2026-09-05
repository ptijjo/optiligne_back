package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path/filepath"

	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/database"
	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/internal/scope"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"gorm.io/gorm"
)

// Run charge la config, connecte la DB et importe GTFS + périmètres.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	n, err := Import(context.Background(), db, cfg.GTFSDataDir, cfg.PerimetresDir)
	if err != nil {
		return err
	}
	log.Printf("import GTFS terminé (%d routes)", n)
	return nil
}

// shouldSkipMissingGTFS : pas de fichiers dans l'image, mais un feed déjà en base (prod).
func shouldSkipMissingGTFS(gtfsDir string, hasActiveFeed bool) (bool, error) {
	if gtfs.DirComplete(gtfsDir) {
		return false, nil
	}
	if hasActiveFeed {
		return true, nil
	}
	return false, fmt.Errorf("gtfs: dossier incomplet (%s) et aucun feed actif en base — définir GTFS_DATA_DIR (volume)", gtfsDir)
}

// Import parse le feed et les affectations, persist en base.
func Import(ctx context.Context, db *gorm.DB, gtfsDir, perimDir string) (int, error) {
	// 0. Feed déjà en Postgres, fichiers absents de l'image → démarrer l'API.
	var active models.FeedVersion
	activeErr := db.WithContext(ctx).Where("active = ?", true).First(&active).Error
	if activeErr != nil && !errors.Is(activeErr, gorm.ErrRecordNotFound) {
		return 0, activeErr
	}
	skip, err := shouldSkipMissingGTFS(gtfsDir, activeErr == nil)
	if err != nil {
		return 0, err
	}
	if skip {
		log.Printf("GTFS absent de l'image, conservation du feed %s déjà en base", active.FeedVersion)
		var n int64
		if err := db.WithContext(ctx).Model(&models.Route{}).Where("feed_version_id = ?", active.ID).Count(&n).Error; err != nil {
			return 0, err
		}
		return int(n), nil
	}

	// 1. Parser le feed sans charger shapes.txt en mémoire.
	feed, err := gtfs.ParseDirWithoutShapes(gtfsDir)
	if err != nil {
		return 0, err
	}
	if n := len(feed.Anomalies); n > 0 {
		shown := n
		if shown > 20 {
			shown = 20
		}
		log.Printf("%d anomalie(s) GTFS (flag, pas de correction): %v", n, feed.Anomalies[:shown])
	}
	// 2. Charger le périmètre déclaré (échec si ligne inconnue).
	perim, err := scope.Load(perimDir, feed.Routes)
	if err != nil {
		return 0, err
	}
	// 3. Checksum de version : même feed = no-op.
	sum := sha256.Sum256([]byte(feed.Version + "|" + feed.StartDate + "|" + feed.EndDate))
	checksum := hex.EncodeToString(sum[:])

	var existing models.FeedVersion
	err = db.WithContext(ctx).Where("checksum = ?", checksum).First(&existing).Error
	if err == nil {
		log.Printf("feed %s déjà importé, no-op", feed.Version)
		return len(feed.Routes), nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, err
	}

	neededShapes := assignedShapeIDs(feed.Trips, perim)

	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 4. Désactiver l'ancien feed, créer la nouvelle version.
		if err := tx.Model(&models.FeedVersion{}).Where("active = ?", true).Update("active", false).Error; err != nil {
			return err
		}
		fv := models.FeedVersion{
			ID: id.New(), Checksum: checksum, Publisher: feed.Publisher,
			FeedVersion: feed.Version, StartDate: feed.StartDate, EndDate: feed.EndDate, Active: true,
		}
		if err := tx.Create(&fv).Error; err != nil {
			return err
		}
		// 5. Persister tables GTFS (hors shapes).
		if err := persistFeed(tx, fv.ID, feed); err != nil {
			return err
		}
		// 6. Streamer shapes.txt : uniquement les tracés des lignes affectées.
		if err := persistShapes(tx, fv.ID, filepath.Join(gtfsDir, "shapes.txt"), neededShapes); err != nil {
			return err
		}
		// 7. Persister opérateurs / dépôts / affectations.
		if err := persistPerimeter(tx, fv.ID, perim); err != nil {
			return err
		}
		// 8. Précalculer les fractions d'arrêts sur le shape.
		return computeFracs(tx, fv.ID)
	})
	if err != nil {
		return 0, err
	}
	return len(feed.Routes), nil
}

func assignedShapeIDs(trips []gtfs.Trip, perim *scope.Perimeter) map[string]struct{} {
	routes := make(map[string]struct{}, len(perim.Assigns))
	for _, a := range perim.Assigns {
		routes[a.RouteID] = struct{}{}
	}
	out := map[string]struct{}{}
	for _, t := range trips {
		if t.ShapeID == "" {
			continue
		}
		if _, ok := routes[t.RouteID]; ok {
			out[t.ShapeID] = struct{}{}
		}
	}
	return out
}

func persistFeed(tx *gorm.DB, feedID string, feed *gtfs.Feed) error {
	agencies := make([]models.Agency, 0, len(feed.Agencies))
	for _, a := range feed.Agencies {
		agencies = append(agencies, models.Agency{ID: id.New(), FeedVersionID: feedID, AgencyID: a.AgencyID, Name: a.Name, Timezone: a.Timezone})
	}
	if err := createBatches(tx, agencies, 200); err != nil {
		return err
	}
	routes := make([]models.Route, 0, len(feed.Routes))
	for _, r := range feed.Routes {
		routes = append(routes, models.Route{ID: id.New(), FeedVersionID: feedID, RouteID: r.RouteID, AgencyID: r.AgencyID, ShortName: r.ShortName, LongName: r.LongName, RouteType: r.RouteType})
	}
	if err := createBatches(tx, routes, 200); err != nil {
		return err
	}
	trips := make([]models.Trip, 0, len(feed.Trips))
	for _, t := range feed.Trips {
		trips = append(trips, models.Trip{ID: id.New(), FeedVersionID: feedID, TripID: t.TripID, RouteID: t.RouteID, ServiceID: t.ServiceID, ShapeID: t.ShapeID, Headsign: t.Headsign, DirectionID: t.DirectionID})
	}
	if err := createBatches(tx, trips, 200); err != nil {
		return err
	}
	stops := modelStops(feedID, feed.Stops)
	if err := createBatches(tx, stops, 500); err != nil {
		return err
	}
	sts := make([]models.StopTime, 0, len(feed.StopTimes))
	for _, st := range feed.StopTimes {
		sts = append(sts, models.StopTime{ID: id.New(), FeedVersionID: feedID, TripID: st.TripID, StopID: st.StopID, StopSequence: st.StopSequence, ArrivalSec: st.ArrivalSec, DepartureSec: st.DepartureSec, ShapeDist: st.ShapeDist})
	}
	if err := createBatches(tx, sts, 500); err != nil {
		return err
	}
	cals := make([]models.Calendar, 0, len(feed.Calendars))
	for _, c := range feed.Calendars {
		cals = append(cals, models.Calendar{
			ID: id.New(), FeedVersionID: feedID, ServiceID: c.ServiceID,
			Monday: c.Monday, Tuesday: c.Tuesday, Wednesday: c.Wednesday, Thursday: c.Thursday,
			Friday: c.Friday, Saturday: c.Saturday, Sunday: c.Sunday,
			StartDate: c.StartDate, EndDate: c.EndDate,
		})
	}
	if err := createBatches(tx, cals, 200); err != nil {
		return err
	}
	dates := make([]models.CalendarDate, 0, len(feed.CalendarDates))
	for _, d := range feed.CalendarDates {
		dates = append(dates, models.CalendarDate{ID: id.New(), FeedVersionID: feedID, ServiceID: d.ServiceID, Date: d.Date, ExceptionType: d.ExceptionType})
	}
	return createBatches(tx, dates, 500)
}

// modelStops mappe les arrêts GTFS vers les modèles, en ignorant stop_id vide
// et en dédupliquant (ux_stop_feed) — flag au parse, défense ici.
func modelStops(feedID string, in []gtfs.Stop) []models.Stop {
	seen := make(map[string]struct{}, len(in))
	out := make([]models.Stop, 0, len(in))
	for _, s := range in {
		if s.StopID == "" {
			continue
		}
		if _, ok := seen[s.StopID]; ok {
			continue
		}
		seen[s.StopID] = struct{}{}
		out = append(out, models.Stop{
			ID:            id.New(),
			FeedVersionID: feedID,
			StopID:        s.StopID,
			Name:          s.Name,
			Lat:           s.Lat,
			Lon:           s.Lon,
		})
	}
	return out
}

func persistShapes(tx *gorm.DB, feedID, shapesPath string, wanted map[string]struct{}) error {
	return gtfs.StreamShapes(shapesPath, wanted, func(shapeID string, pts []gtfs.ShapePoint) error {
		sh := models.Shape{ID: id.New(), FeedVersionID: feedID, ShapeID: shapeID}
		if err := tx.Create(&sh).Error; err != nil {
			return err
		}
		wkt := gtfs.LineStringWKT(pts)
		return tx.Exec(`UPDATE shapes SET geom = ST_SetSRID(ST_GeomFromText(?), 4326) WHERE id = ?`, wkt, sh.ID).Error
	})
}

func persistPerimeter(tx *gorm.DB, feedID string, p *scope.Perimeter) error {
	opIDs := map[string]string{}
	for _, o := range p.Operators {
		var existing models.Operator
		err := tx.Where("code = ?", o.Code).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			existing = models.Operator{ID: id.New(), Code: o.Code, Name: o.Name}
			if err := tx.Create(&existing).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		opIDs[o.Code] = existing.ID
	}
	depIDs := map[string]string{}
	for _, d := range p.Depots {
		oid := opIDs[d.OperatorCode]
		var existing models.Depot
		err := tx.Where("code = ? AND operator_id = ?", d.Code, oid).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			existing = models.Depot{ID: id.New(), Code: d.Code, OperatorID: oid, Name: d.Name}
			if err := tx.Create(&existing).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		depIDs[d.OperatorCode+"|"+d.Code] = existing.ID
	}
	for _, a := range p.Assigns {
		row := models.RouteAssignment{
			ID: id.New(), FeedVersionID: feedID,
			OperatorID: opIDs[a.OperatorCode],
			DepotID:    depIDs[a.OperatorCode+"|"+a.DepotCode],
			RouteID:    a.RouteID,
		}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func computeFracs(tx *gorm.DB, feedID string) error {
	type row struct {
		TripID       string
		StopID       string
		StopSequence int
		Frac         float64
		ArrivalSec   int
		StopName     string
	}
	var rows []row
	if err := tx.Raw(`
		SELECT
			t.trip_id,
			st.stop_id,
			st.stop_sequence,
			COALESCE(ST_LineLocatePoint(sh.geom, ST_SetSRID(ST_MakePoint(s.lon, s.lat), 4326)), 0) AS frac,
			st.arrival_sec,
			s.name AS stop_name
		FROM stop_times st
		JOIN trips t ON t.trip_id = st.trip_id AND t.feed_version_id = st.feed_version_id
		JOIN stops s ON s.stop_id = st.stop_id AND s.feed_version_id = st.feed_version_id
		JOIN shapes sh ON sh.shape_id = t.shape_id AND sh.feed_version_id = t.feed_version_id
		WHERE st.feed_version_id = ?
	`, feedID).Scan(&rows).Error; err != nil {
		return err
	}
	fracs := make([]models.StopFrac, 0, len(rows))
	for _, r := range rows {
		fracs = append(fracs, models.StopFrac{
			ID: id.New(), FeedVersionID: feedID, TripID: r.TripID, StopID: r.StopID,
			StopSequence: r.StopSequence, Frac: r.Frac, ArrivalSec: r.ArrivalSec, StopName: r.StopName,
		})
	}
	return createBatches(tx, fracs, 500)
}

func createBatches[T any](tx *gorm.DB, rows []T, size int) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.CreateInBatches(rows, size).Error
}
