// Patch les points shapes.txt les plus proches d'un arrêt GTFS (dev / corrections périmètre).
package main

import (
	"encoding/csv"
	"flag"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	stopID := flag.String("stop", "", "stop_id GTFS")
	lat := flag.Float64("lat", 0, "nouvelle latitude")
	lon := flag.Float64("lon", 0, "nouvelle longitude")
	dir := flag.String("dir", "GTFS", "dossier GTFS")
	flag.Parse()

	if *stopID == "" || *lat == 0 || *lon == 0 {
		log.Fatal("usage: shapefix -stop 1034178 -lat 49.201344 -lon 6.928857 [-dir GTFS]")
	}

	shapeIDs, err := shapeIDsForStop(filepath.Join(*dir, "trips.txt"), filepath.Join(*dir, "stop_times.txt"), *stopID)
	if err != nil {
		log.Fatal(err)
	}
	if len(shapeIDs) == 0 {
		log.Fatalf("aucun shape pour stop %s", *stopID)
	}
	log.Printf("%d shape(s) à corriger pour stop %s", len(shapeIDs), *stopID)

	shapesPath := filepath.Join(*dir, "shapes.txt")
	best, err := findClosestPoints(shapesPath, shapeIDs, *lat, *lon)
	if err != nil {
		log.Fatal(err)
	}
	for id, pt := range best {
		log.Printf("  %s → seq %d (%.6f, %.6f) → (%.6f, %.6f)", id, pt.seq, pt.lat, pt.lon, *lat, *lon)
	}

	n, err := patchShapes(shapesPath, best, *lat, *lon)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("shapes.txt mis à jour (%d lignes modifiées)", n)
}

type patchPoint struct {
	seq int
	lat float64
	lon float64
}

func shapeIDsForStop(tripsPath, stopTimesPath, stopID string) (map[string]struct{}, error) {
	tripShapes, err := loadTripShapes(tripsPath)
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	err = eachCSV(stopTimesPath, func(h, row []string) error {
		if col(h, row, "stop_id") != stopID {
			return nil
		}
		tripID := col(h, row, "trip_id")
		if shapeID := tripShapes[tripID]; shapeID != "" {
			out[shapeID] = struct{}{}
		}
		return nil
	})
	return out, err
}

func loadTripShapes(path string) (map[string]string, error) {
	out := map[string]string{}
	err := eachCSV(path, func(h, row []string) error {
		tripID := col(h, row, "trip_id")
		shapeID := col(h, row, "shape_id")
		if tripID != "" && shapeID != "" {
			out[tripID] = shapeID
		}
		return nil
	})
	return out, err
}

func findClosestPoints(path string, shapeIDs map[string]struct{}, lat, lon float64) (map[string]patchPoint, error) {
	best := map[string]patchPoint{}
	bestDist := map[string]float64{}

	err := eachCSV(path, func(h, row []string) error {
		id := col(h, row, "shape_id")
		if _, ok := shapeIDs[id]; !ok {
			return nil
		}
		ptLat, _ := strconv.ParseFloat(col(h, row, "shape_pt_lat"), 64)
		ptLon, _ := strconv.ParseFloat(col(h, row, "shape_pt_lon"), 64)
		seq, _ := strconv.Atoi(col(h, row, "shape_pt_sequence"))
		d := distM(ptLat, ptLon, lat, lon)
		if prev, ok := bestDist[id]; !ok || d < prev {
			bestDist[id] = d
			best[id] = patchPoint{seq: seq, lat: ptLat, lon: ptLon}
		}
		return nil
	})
	return best, err
}

func patchShapes(path string, best map[string]patchPoint, lat, lon float64) (int, error) {
	// shape_id → seq à remplacer
	targetSeq := map[string]int{}
	for id, pt := range best {
		targetSeq[id] = pt.seq
	}

	in, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer in.Close()

	tmp, err := os.CreateTemp(filepath.Dir(path), "shapes-*.txt")
	if err != nil {
		return 0, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	r := csv.NewReader(in)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	w := csv.NewWriter(tmp)

	header, err := r.Read()
	if err != nil {
		_ = tmp.Close()
		return 0, err
	}
	if err := w.Write(header); err != nil {
		_ = tmp.Close()
		return 0, err
	}
	idx := indexHeader(header)

	changed := 0
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = tmp.Close()
			return 0, err
		}
		id := rowAt(row, idx, "shape_id")
		seq, _ := strconv.Atoi(rowAt(row, idx, "shape_pt_sequence"))
		if want, ok := targetSeq[id]; ok && seq == want {
			setCol(row, idx, "shape_pt_lat", formatCoord(lat))
			setCol(row, idx, "shape_pt_lon", formatCoord(lon))
			changed++
		}
		if err := w.Write(row); err != nil {
			_ = tmp.Close()
			return 0, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		_ = tmp.Close()
		return 0, err
	}
	if err := tmp.Close(); err != nil {
		return 0, err
	}
	if err := in.Close(); err != nil {
		return 0, err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return 0, err
	}
	return changed, nil
}

func formatCoord(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func distM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthR = 6371000.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthR * math.Asin(math.Sqrt(a))
}

func indexHeader(header []string) map[string]int {
	m := make(map[string]int, len(header))
	for i, h := range header {
		m[strings.TrimSpace(strings.TrimPrefix(h, "\ufeff"))] = i
	}
	return m
}

func rowAt(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}

func setCol(row []string, idx map[string]int, name, value string) {
	if i, ok := idx[name]; ok && i < len(row) {
		row[i] = value
	}
}

func col(header, row []string, name string) string {
	for i, h := range header {
		if h == name && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func eachCSV(path string, fn func(header, row []string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	for i := range header {
		header[i] = strings.TrimSpace(strings.TrimPrefix(header[i], "\ufeff"))
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(header, row); err != nil {
			return err
		}
	}
}
