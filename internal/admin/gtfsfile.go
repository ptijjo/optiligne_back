package admin

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

// Files persiste les corrections dans le dossier GTFS/.
type Files interface {
	PatchStop(stopID string, lat, lng float64) error
	PatchRouteType(routeID string, routeType int) error
	UpsertStop(stopID, name string, lat, lng float64) error
	UpsertRoute(routeID, agencyID, shortName, longName string, routeType int) error
	UpsertTrip(tripID, routeID, serviceID, shapeID, headsign string) error
	UpsertCalendar(cal CalendarFileRow) error
	ReplaceShape(shapeID string, pts []gtfs.ShapePoint) error
	ReplaceShapes(shapeIDs []string, pts []gtfs.ShapePoint) error
	ReplaceStopTimes(tripIDs []string, rows []StopTimeFileRow) error
}

// StopTimeFileRow est une ligne stop_times à écrire dans le CSV.
type StopTimeFileRow struct {
	TripID       string
	StopID       string
	StopSequence int
	ArrivalSec   int
	DepartureSec int
}

// CalendarFileRow est une ligne calendar.txt.
type CalendarFileRow struct {
	ServiceID string
	Monday    bool
	Tuesday   bool
	Wednesday bool
	Thursday  bool
	Friday    bool
	Saturday  bool
	Sunday    bool
	StartDate string
	EndDate   string
}

// GTFSFiles écrit stops.txt, shapes.txt et stop_times.txt.
type GTFSFiles struct {
	Dir string
}

func NewGTFSFiles(dir string) *GTFSFiles {
	return &GTFSFiles{Dir: dir}
}

// resolve pointe vers GTFS/<secteur>/fichier.txt si l’id est préfixé (import multi-feeds).
func (f *GTFSFiles) resolve(entityID, filename string) (path, rawID string) {
	sector, raw := gtfs.StripSectorPrefix(entityID)
	if sector != "" {
		return filepath.Join(f.Dir, sector, filename), raw
	}
	return filepath.Join(f.Dir, filename), entityID
}

func (f *GTFSFiles) PatchStop(stopID string, lat, lng float64) error {
	path, raw := f.resolve(stopID, "stops.txt")
	return rewriteCSV(path, func(header, row []string) []string {
		if col(header, row, "stop_id") == "" {
			return nil
		}
		if col(header, row, "stop_id") != raw {
			return row
		}
		out := append([]string(nil), row...)
		setCol(header, out, "stop_lat", formatCoord(lat))
		setCol(header, out, "stop_lon", formatCoord(lng))
		return out
	})
}

func (f *GTFSFiles) PatchRouteType(routeID string, routeType int) error {
	path, raw := f.resolve(routeID, "routes.txt")
	return rewriteCSV(path, func(header, row []string) []string {
		if col(header, row, "route_id") != raw {
			return row
		}
		out := append([]string(nil), row...)
		setCol(header, out, "route_type", strconv.Itoa(routeType))
		return out
	})
}

func (f *GTFSFiles) UpsertRoute(routeID, agencyID, shortName, longName string, routeType int) error {
	path, rawRoute := f.resolve(routeID, "routes.txt")
	_, rawAgency := f.resolve(agencyID, "routes.txt")
	return upsertCSVRow(path, "route_id", rawRoute, func(header []string, row []string) {
		setCol(header, row, "route_id", rawRoute)
		if agencyID != "" {
			setCol(header, row, "agency_id", rawAgency)
		}
		setCol(header, row, "route_short_name", shortName)
		setCol(header, row, "route_long_name", longName)
		setCol(header, row, "route_type", strconv.Itoa(routeType))
	})
}

func (f *GTFSFiles) UpsertTrip(tripID, routeID, serviceID, shapeID, headsign string) error {
	path, rawTrip := f.resolve(tripID, "trips.txt")
	_, rawRoute := f.resolve(routeID, "trips.txt")
	_, rawService := f.resolve(serviceID, "trips.txt")
	_, rawShape := f.resolve(shapeID, "trips.txt")
	return upsertCSVRow(path, "trip_id", rawTrip, func(header []string, row []string) {
		setCol(header, row, "trip_id", rawTrip)
		setCol(header, row, "route_id", rawRoute)
		setCol(header, row, "service_id", rawService)
		setCol(header, row, "shape_id", rawShape)
		setCol(header, row, "trip_headsign", headsign)
	})
}

func (f *GTFSFiles) UpsertCalendar(cal CalendarFileRow) error {
	path, rawService := f.resolve(cal.ServiceID, "calendar.txt")
	return upsertCSVRow(path, "service_id", rawService, func(header []string, row []string) {
		setCol(header, row, "service_id", rawService)
		setCol(header, row, "monday", bool01(cal.Monday))
		setCol(header, row, "tuesday", bool01(cal.Tuesday))
		setCol(header, row, "wednesday", bool01(cal.Wednesday))
		setCol(header, row, "thursday", bool01(cal.Thursday))
		setCol(header, row, "friday", bool01(cal.Friday))
		setCol(header, row, "saturday", bool01(cal.Saturday))
		setCol(header, row, "sunday", bool01(cal.Sunday))
		setCol(header, row, "start_date", cal.StartDate)
		setCol(header, row, "end_date", cal.EndDate)
	})
}

func bool01(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func upsertCSVRow(path, idCol, idVal string, apply func(header, row []string)) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "gtfs-*.txt")
	if err != nil {
		return err
	}
	w := csv.NewWriter(tmp)
	if err := w.Write(header); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	found := false
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		if col(header, row, idCol) == idVal {
			found = true
			out := make([]string, len(header))
			copy(out, row)
			if len(out) < len(header) {
				out = make([]string, len(header))
			}
			apply(header, out)
			if err := w.Write(out); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return err
			}
			continue
		}
		if err := w.Write(row); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	if !found {
		nr := make([]string, len(header))
		apply(header, nr)
		if err := w.Write(nr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	_ = in.Close()
	return replaceFile(tmp.Name(), path)
}

// UpsertStop met à jour ou ajoute une ligne dans stops.txt.
func (f *GTFSFiles) UpsertStop(stopID, name string, lat, lng float64) error {
	stopID = strings.TrimSpace(stopID)
	if stopID == "" {
		return ErrInvalidCoords
	}
	path, raw := f.resolve(stopID, "stops.txt")
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	header = normalizeHeader(header)
	tmp, err := os.CreateTemp(filepath.Dir(path), "stops-*.txt")
	if err != nil {
		return err
	}
	w := csv.NewWriter(tmp)
	if err := w.Write(header); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	found := false
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		existingID := col(header, row, "stop_id")
		if existingID == "" {
			continue
		}
		if existingID == raw {
			found = true
			out := make([]string, len(header))
			copy(out, row)
			if err := fillStopRow(header, out, raw, name, lat, lng); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return err
			}
			if err := w.Write(out); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return err
			}
			continue
		}
		if err := w.Write(row); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	if !found {
		nr := make([]string, len(header))
		if err := fillStopRow(header, nr, raw, name, lat, lng); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		if err := w.Write(nr); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	_ = in.Close()
	return replaceFile(tmp.Name(), path)
}

func (f *GTFSFiles) ReplaceShape(shapeID string, pts []gtfs.ShapePoint) error {
	return f.ReplaceShapes([]string{shapeID}, pts)
}

// ReplaceShapes réécrit shapes.txt en une seule passe pour tous les shape_id (évite N scans du fichier géant).
func (f *GTFSFiles) ReplaceShapes(shapeIDs []string, pts []gtfs.ShapePoint) error {
	if len(pts) < 2 {
		return ErrShapeTooShort
	}
	want := make(map[string]struct{}, len(shapeIDs))
	var path string
	for _, id := range shapeIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		p, raw := f.resolve(id, "shapes.txt")
		if path == "" {
			path = p
		}
		want[raw] = struct{}{}
	}
	if len(want) == 0 {
		return nil
	}
	if path == "" {
		path = filepath.Join(f.Dir, "shapes.txt")
	}

	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	header = normalizeHeader(header)

	tmp, err := os.CreateTemp(filepath.Dir(path), "shapes-*.txt")
	if err != nil {
		return err
	}
	w := csv.NewWriter(tmp)
	if err := w.Write(header); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	written := make(map[string]struct{}, len(want))
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		id := col(header, row, "shape_id")
		if _, ok := want[id]; ok {
			if _, done := written[id]; !done {
				if err := writeShapeRows(w, header, id, pts); err != nil {
					tmp.Close()
					os.Remove(tmp.Name())
					return err
				}
				written[id] = struct{}{}
			}
			continue
		}
		if err := w.Write(row); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	for id := range want {
		if _, done := written[id]; done {
			continue
		}
		if err := writeShapeRows(w, header, id, pts); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	_ = in.Close()
	return replaceFile(tmp.Name(), path)
}

// ReplaceStopTimes réécrit stop_times.txt en une passe : remplace les lignes des trip_id listés.
func (f *GTFSFiles) ReplaceStopTimes(tripIDs []string, rows []StopTimeFileRow) error {
	want := make(map[string]struct{}, len(tripIDs))
	var path string
	for _, id := range tripIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		p, raw := f.resolve(id, "stop_times.txt")
		if path == "" {
			path = p
		}
		want[raw] = struct{}{}
	}
	if len(want) == 0 {
		return nil
	}
	if path == "" {
		path = filepath.Join(f.Dir, "stop_times.txt")
	}
	byTrip := make(map[string][]StopTimeFileRow, len(want))
	for _, row := range rows {
		_, rawTrip := f.resolve(row.TripID, "stop_times.txt")
		_, rawStop := f.resolve(row.StopID, "stop_times.txt")
		if _, ok := want[rawTrip]; ok {
			byTrip[rawTrip] = append(byTrip[rawTrip], StopTimeFileRow{
				TripID: rawTrip, StopID: rawStop,
				StopSequence: row.StopSequence, ArrivalSec: row.ArrivalSec, DepartureSec: row.DepartureSec,
			})
		}
	}

	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	header = normalizeHeader(header)
	tmp, err := os.CreateTemp(filepath.Dir(path), "stop_times-*.txt")
	if err != nil {
		return err
	}
	w := csv.NewWriter(tmp)
	if err := w.Write(header); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	written := make(map[string]struct{}, len(want))
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		tid := col(header, row, "trip_id")
		if _, ok := want[tid]; ok {
			if _, done := written[tid]; !done {
				if err := writeStopTimeRows(w, header, byTrip[tid]); err != nil {
					tmp.Close()
					os.Remove(tmp.Name())
					return err
				}
				written[tid] = struct{}{}
			}
			continue
		}
		if err := w.Write(row); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	for tid := range want {
		if _, done := written[tid]; done {
			continue
		}
		if err := writeStopTimeRows(w, header, byTrip[tid]); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	_ = in.Close()
	return replaceFile(tmp.Name(), path)
}

func writeStopTimeRows(w *csv.Writer, header []string, rows []StopTimeFileRow) error {
	for _, st := range rows {
		if strings.TrimSpace(st.StopID) == "" {
			continue
		}
		nr := make([]string, len(header))
		setCol(header, nr, "trip_id", st.TripID)
		setCol(header, nr, "arrival_time", formatGTFSTimeHMS(st.ArrivalSec))
		setCol(header, nr, "departure_time", formatGTFSTimeHMS(st.DepartureSec))
		setCol(header, nr, "stop_id", st.StopID)
		setCol(header, nr, "stop_sequence", strconv.Itoa(st.StopSequence))
		if err := w.Write(nr); err != nil {
			return err
		}
	}
	return nil
}

func formatGTFSTimeHMS(sec int) string {
	if sec < 0 {
		sec = 0
	}
	h := sec / 3600
	m := (sec % 3600) / 60
	s := sec % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func writeShapeRows(w *csv.Writer, header []string, shapeID string, pts []gtfs.ShapePoint) error {
	for i, p := range pts {
		nr := make([]string, len(header))
		setCol(header, nr, "shape_id", shapeID)
		setCol(header, nr, "shape_pt_lat", formatCoord(p.Lat))
		setCol(header, nr, "shape_pt_lon", formatCoord(p.Lon))
		setCol(header, nr, "shape_pt_sequence", strconv.Itoa(i+1))
		if err := w.Write(nr); err != nil {
			return err
		}
	}
	return nil
}

func rewriteCSV(path string, fn func(header, row []string) []string) error {
	in, err := os.Open(path)
	if err != nil {
		return err
	}
	defer in.Close()
	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	header = normalizeHeader(header)
	tmp, err := os.CreateTemp(filepath.Dir(path), "gtfs-*.txt")
	if err != nil {
		return err
	}
	w := csv.NewWriter(tmp)
	if err := w.Write(header); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
		out := fn(header, row)
		if out == nil {
			continue
		}
		if err := w.Write(out); err != nil {
			tmp.Close()
			os.Remove(tmp.Name())
			return err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	_ = in.Close()
	return replaceFile(tmp.Name(), path)
}

func replaceFile(tmpPath, dest string) error {
	if err := os.Rename(tmpPath, dest); err == nil {
		return nil
	}
	// Fallback (ex. volumes Docker / EXDEV) : copie puis suppression.
	in, err := os.Open(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	_ = os.Remove(tmpPath)
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func col(header, row []string, name string) string {
	for i, h := range header {
		if strings.TrimSpace(h) == name && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func setCol(header, row []string, name, val string) bool {
	for i, h := range header {
		if strings.TrimSpace(h) == name && i < len(row) {
			row[i] = val
			return true
		}
	}
	return false
}

func fillStopRow(header, row []string, stopID, name string, lat, lng float64) error {
	if !setCol(header, row, "stop_id", stopID) {
		return fmt.Errorf("colonne stop_id absente de stops.txt")
	}
	setCol(header, row, "stop_name", name)
	setCol(header, row, "stop_lat", formatCoord(lat))
	setCol(header, row, "stop_lon", formatCoord(lng))
	return nil
}

func normalizeHeader(header []string) []string {
	out := make([]string, len(header))
	for i, h := range header {
		h = strings.TrimSpace(h)
		if i == 0 {
			h = strings.TrimPrefix(h, "\ufeff")
		}
		out[i] = h
	}
	return out
}

func formatCoord(v float64) string {
	return fmt.Sprintf("%.6f", v)
}

func coordsToPoints(coords [][]float64) []gtfs.ShapePoint {
	out := make([]gtfs.ShapePoint, 0, len(coords))
	for i, c := range coords {
		if len(c) < 2 {
			continue
		}
		out = append(out, gtfs.ShapePoint{Lon: c[0], Lat: c[1], Sequence: i + 1})
	}
	return out
}
