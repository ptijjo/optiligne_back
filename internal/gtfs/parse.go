package gtfs

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Feed est le GTFS parsé en mémoire (sans GORM).
type Feed struct {
	Publisher    string
	Version      string
	StartDate    string
	EndDate      string
	Agencies     []Agency
	Routes       []Route
	Trips        []Trip
	Stops        []Stop
	StopTimes    []StopTime
	Calendars    []Calendar
	CalendarDates []CalendarDate
	ShapePoints  map[string][]ShapePoint
}

type Agency struct {
	AgencyID, Name, Timezone string
}

type Route struct {
	RouteID, AgencyID, ShortName, LongName string
	RouteType                              int
}

type Trip struct {
	TripID, RouteID, ServiceID, ShapeID, Headsign string
	DirectionID                                   int
}

type Stop struct {
	StopID, Name string
	Lat, Lon     float64
}

type StopTime struct {
	TripID, StopID           string
	StopSequence             int
	ArrivalSec, DepartureSec int
	ShapeDist                float64
}

type Calendar struct {
	ServiceID, StartDate, EndDate string
	Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, Sunday bool
}

type CalendarDate struct {
	ServiceID, Date string
	ExceptionType   int
}

type ShapePoint struct {
	Lat, Lon  float64
	Sequence  int
	Dist      float64
}

var requiredFiles = []string{
	"agency.txt", "routes.txt", "trips.txt", "stops.txt",
	"stop_times.txt", "shapes.txt", "calendar.txt", "feed_info.txt",
}

// ParseDir lit un dossier GTFS de fichiers .txt (shapes inclus, pour fixtures).
func ParseDir(dir string) (*Feed, error) {
	return parseDir(dir, true)
}

// ParseDirWithoutShapes lit le feed sans charger shapes.txt en mémoire.
func ParseDirWithoutShapes(dir string) (*Feed, error) {
	return parseDir(dir, false)
}

func parseDir(dir string, withShapes bool) (*Feed, error) {
	for _, name := range requiredFiles {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return nil, fmt.Errorf("gtfs: fichier manquant %s: %w", name, err)
		}
	}
	feed := &Feed{ShapePoints: map[string][]ShapePoint{}}
	if err := parseFeedInfo(filepath.Join(dir, "feed_info.txt"), feed); err != nil {
		return nil, err
	}
	if err := parseAgencies(filepath.Join(dir, "agency.txt"), feed); err != nil {
		return nil, err
	}
	if err := parseRoutes(filepath.Join(dir, "routes.txt"), feed); err != nil {
		return nil, err
	}
	if err := parseTrips(filepath.Join(dir, "trips.txt"), feed); err != nil {
		return nil, err
	}
	if err := parseStops(filepath.Join(dir, "stops.txt"), feed); err != nil {
		return nil, err
	}
	if err := parseStopTimes(filepath.Join(dir, "stop_times.txt"), feed); err != nil {
		return nil, err
	}
	if withShapes {
		if err := StreamShapes(filepath.Join(dir, "shapes.txt"), nil, func(id string, pts []ShapePoint) error {
			feed.ShapePoints[id] = pts
			return nil
		}); err != nil {
			return nil, err
		}
	}
	if err := parseCalendar(filepath.Join(dir, "calendar.txt"), feed); err != nil {
		return nil, err
	}
	calDates := filepath.Join(dir, "calendar_dates.txt")
	if _, err := os.Stat(calDates); err == nil {
		if err := parseCalendarDates(calDates, feed); err != nil {
			return nil, err
		}
	}
	return feed, nil
}

// StreamShapes lit shapes.txt en flux, un shape à la fois (fichier souvent groupé par shape_id).
// Si wanted est non nil, seuls ces shape_id sont transmis au callback.
func StreamShapes(path string, wanted map[string]struct{}, fn func(shapeID string, pts []ShapePoint) error) error {
	header, r, f, err := openCSV(path)
	if err != nil {
		return fmt.Errorf("gtfs: %s: %w", filepath.Base(path), err)
	}
	defer f.Close()

	var curID string
	var pts []ShapePoint
	flush := func() error {
		if curID == "" || len(pts) < 2 {
			pts = pts[:0]
			curID = ""
			return nil
		}
		sort.Slice(pts, func(i, j int) bool { return pts[i].Sequence < pts[j].Sequence })
		cp := make([]ShapePoint, len(pts))
		copy(cp, pts)
		err := fn(curID, cp)
		pts = pts[:0]
		curID = ""
		return err
	}

	for {
		row, err := r.Read()
		if err == io.EOF {
			return flush()
		}
		if err != nil {
			return fmt.Errorf("gtfs: %s: %w", filepath.Base(path), err)
		}
		id := col(header, row, "shape_id")
		if wanted != nil {
			if _, ok := wanted[id]; !ok {
				if curID != "" && curID != id {
					if err := flush(); err != nil {
						return err
					}
				}
				continue
			}
		}
		if id != curID {
			if err := flush(); err != nil {
				return err
			}
			curID = id
		}
		lat, _ := strconv.ParseFloat(col(header, row, "shape_pt_lat"), 64)
		lon, _ := strconv.ParseFloat(col(header, row, "shape_pt_lon"), 64)
		seq, _ := strconv.Atoi(col(header, row, "shape_pt_sequence"))
		dist, _ := strconv.ParseFloat(col(header, row, "shape_dist_traveled"), 64)
		pts = append(pts, ShapePoint{Lat: lat, Lon: lon, Sequence: seq, Dist: dist})
	}
}

// ParseGTFSTime convertit HH:MM:SS (peut dépasser 24h) en secondes depuis minuit service.
func ParseGTFSTime(s string) (int, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("horaire GTFS invalide %q", s)
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	sec, err3 := strconv.Atoi(parts[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, fmt.Errorf("horaire GTFS invalide %q", s)
	}
	return h*3600 + m*60 + sec, nil
}

// FormatGTFSClock formate des secondes depuis minuit service en HH:MM (peut dépasser 24h).
func FormatGTFSClock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// LineStringWKT construit un WKT lon/lat ordonné.
func LineStringWKT(pts []ShapePoint) string {
	var b strings.Builder
	b.WriteString("LINESTRING(")
	for i, p := range pts {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(strconv.FormatFloat(p.Lon, 'f', -1, 64))
		b.WriteByte(' ')
		b.WriteString(strconv.FormatFloat(p.Lat, 'f', -1, 64))
	}
	b.WriteByte(')')
	return b.String()
}

func openCSV(path string) ([]string, *csv.Reader, *os.File, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, nil, err
	}
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		_ = f.Close()
		return nil, nil, nil, err
	}
	idx := make([]string, len(header))
	for i, h := range header {
		idx[i] = cleanHeader(h, i == 0)
	}
	return idx, r, f, nil
}

func cleanHeader(h string, first bool) string {
	h = strings.TrimSpace(h)
	if first {
		h = strings.TrimPrefix(h, "\ufeff")
	}
	return h
}

func col(header []string, row []string, name string) string {
	for i, h := range header {
		if h == name {
			if i < len(row) {
				return strings.TrimSpace(row[i])
			}
			return ""
		}
	}
	return ""
}

func parseFeedInfo(path string, feed *Feed) error {
	header, r, f, err := openCSV(path)
	if err != nil {
		return err
	}
	defer f.Close()
	row, err := r.Read()
	if err != nil && err != io.EOF {
		return err
	}
	if row != nil {
		feed.Publisher = col(header, row, "feed_publisher_name")
		feed.Version = col(header, row, "feed_version")
		feed.StartDate = col(header, row, "feed_start_date")
		feed.EndDate = col(header, row, "feed_end_date")
	}
	return nil
}

func parseAgencies(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		feed.Agencies = append(feed.Agencies, Agency{
			AgencyID: col(h, row, "agency_id"),
			Name:     col(h, row, "agency_name"),
			Timezone: col(h, row, "agency_timezone"),
		})
		return nil
	})
}

func parseRoutes(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		rt, _ := strconv.Atoi(col(h, row, "route_type"))
		feed.Routes = append(feed.Routes, Route{
			RouteID:   col(h, row, "route_id"),
			AgencyID:  col(h, row, "agency_id"),
			ShortName: col(h, row, "route_short_name"),
			LongName:  col(h, row, "route_long_name"),
			RouteType: rt,
		})
		return nil
	})
}

func parseTrips(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		dir, _ := strconv.Atoi(col(h, row, "direction_id"))
		feed.Trips = append(feed.Trips, Trip{
			TripID:      col(h, row, "trip_id"),
			RouteID:     col(h, row, "route_id"),
			ServiceID:   col(h, row, "service_id"),
			ShapeID:     col(h, row, "shape_id"),
			Headsign:    col(h, row, "trip_headsign"),
			DirectionID: dir,
		})
		return nil
	})
}

func parseStops(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		lat, _ := strconv.ParseFloat(col(h, row, "stop_lat"), 64)
		lon, _ := strconv.ParseFloat(col(h, row, "stop_lon"), 64)
		feed.Stops = append(feed.Stops, Stop{
			StopID: col(h, row, "stop_id"),
			Name:   col(h, row, "stop_name"),
			Lat:    lat,
			Lon:    lon,
		})
		return nil
	})
}

func parseStopTimes(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		arr, err := ParseGTFSTime(col(h, row, "arrival_time"))
		if err != nil {
			return err
		}
		dep, err := ParseGTFSTime(col(h, row, "departure_time"))
		if err != nil {
			return err
		}
		seq, _ := strconv.Atoi(col(h, row, "stop_sequence"))
		dist, _ := strconv.ParseFloat(col(h, row, "shape_dist_traveled"), 64)
		feed.StopTimes = append(feed.StopTimes, StopTime{
			TripID:       col(h, row, "trip_id"),
			StopID:       col(h, row, "stop_id"),
			StopSequence: seq,
			ArrivalSec:   arr,
			DepartureSec: dep,
			ShapeDist:    dist,
		})
		return nil
	})
}

func parseCalendar(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		feed.Calendars = append(feed.Calendars, Calendar{
			ServiceID: col(h, row, "service_id"),
			Monday:    col(h, row, "monday") == "1",
			Tuesday:   col(h, row, "tuesday") == "1",
			Wednesday: col(h, row, "wednesday") == "1",
			Thursday:  col(h, row, "thursday") == "1",
			Friday:    col(h, row, "friday") == "1",
			Saturday:  col(h, row, "saturday") == "1",
			Sunday:    col(h, row, "sunday") == "1",
			StartDate: col(h, row, "start_date"),
			EndDate:   col(h, row, "end_date"),
		})
		return nil
	})
}

func parseCalendarDates(path string, feed *Feed) error {
	return eachRow(path, func(h, row []string) error {
		et, _ := strconv.Atoi(col(h, row, "exception_type"))
		feed.CalendarDates = append(feed.CalendarDates, CalendarDate{
			ServiceID:     col(h, row, "service_id"),
			Date:          col(h, row, "date"),
			ExceptionType: et,
		})
		return nil
	})
}

func eachRow(path string, fn func(header, row []string) error) error {
	header, r, f, err := openCSV(path)
	if err != nil {
		return fmt.Errorf("gtfs: %s: %w", filepath.Base(path), err)
	}
	defer f.Close()
	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("gtfs: %s: %w", filepath.Base(path), err)
		}
		if err := fn(header, row); err != nil {
			return err
		}
	}
}
