package admin_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/admin"
	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

func TestGTFSFiles_PatchStop(t *testing.T) {
	dir := t.TempDir()
	body := "stop_id,stop_name,stop_lat,stop_lon\nA,Depart,49.100000,6.900000\nB,Ecole,49.120000,6.910000\n"
	if err := os.WriteFile(filepath.Join(dir, "stops.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := admin.NewGTFSFiles(dir)
	if err := files.PatchStop("A", 49.201344, 6.928857); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "stops.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "49.201344") || !strings.Contains(string(raw), "6.928857") {
		t.Fatalf("stops.txt = %s", raw)
	}
	if !strings.Contains(string(raw), "Ecole") {
		t.Fatal("l'autre arrêt a disparu")
	}
}

func TestGTFSFiles_ReplaceShape(t *testing.T) {
	dir := t.TempDir()
	body := "shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence\nSH1,49.1,6.9,1\nSH1,49.11,6.91,2\nSH2,49.0,6.8,1\n"
	if err := os.WriteFile(filepath.Join(dir, "shapes.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := admin.NewGTFSFiles(dir)
	err := files.ReplaceShape("SH1", []gtfs.ShapePoint{
		{Lat: 49.2, Lon: 6.92, Sequence: 1},
		{Lat: 49.21, Lon: 6.93, Sequence: 2},
		{Lat: 49.22, Lon: 6.94, Sequence: 3},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "shapes.txt"))
	s := string(raw)
	if strings.Count(s, "SH1") != 3 {
		t.Fatalf("SH1 count: %s", s)
	}
	if !strings.Contains(s, "SH2") {
		t.Fatal("SH2 disparu")
	}
}

func TestGTFSFiles_ReplaceShapes_UneSeulePasse(t *testing.T) {
	dir := t.TempDir()
	body := "shape_id,shape_pt_lat,shape_pt_lon,shape_pt_sequence\nSH1,49.1,6.9,1\nSH1,49.11,6.91,2\nSH2,49.0,6.8,1\nSH3,48.9,6.7,1\n"
	if err := os.WriteFile(filepath.Join(dir, "shapes.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := admin.NewGTFSFiles(dir)
	pts := []gtfs.ShapePoint{
		{Lat: 49.2, Lon: 6.92, Sequence: 1},
		{Lat: 49.21, Lon: 6.93, Sequence: 2},
	}
	if err := files.ReplaceShapes([]string{"SH1", "SH2"}, pts); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "shapes.txt"))
	s := string(raw)
	if strings.Count(s, "SH1,") != 2 || strings.Count(s, "SH2,") != 2 {
		t.Fatalf("SH1/SH2 mal remplacés: %s", s)
	}
	if !strings.Contains(s, "SH3") {
		t.Fatal("SH3 disparu")
	}
}

func TestGTFSFiles_UpsertStop_Ajoute(t *testing.T) {
	dir := t.TempDir()
	body := "stop_id,stop_name,stop_lat,stop_lon\nA,Depart,49.100000,6.900000\n"
	if err := os.WriteFile(filepath.Join(dir, "stops.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := admin.NewGTFSFiles(dir)
	if err := files.UpsertStop("ol-x", "Nouveau", 49.2, 6.92); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "stops.txt"))
	if !strings.Contains(string(raw), "ol-x") || !strings.Contains(string(raw), "Nouveau") {
		t.Fatalf("%s", raw)
	}
}

func TestGTFSFiles_ReplaceStopTimes_UneSeulePasse(t *testing.T) {
	dir := t.TempDir()
	body := "trip_id,arrival_time,departure_time,stop_id,stop_sequence\nT1,07:00:00,07:00:00,A,1\nT1,07:10:00,07:10:00,B,2\nT2,08:00:00,08:00:00,A,1\n"
	if err := os.WriteFile(filepath.Join(dir, "stop_times.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := admin.NewGTFSFiles(dir)
	err := files.ReplaceStopTimes([]string{"T1"}, []admin.StopTimeFileRow{
		{TripID: "T1", StopID: "A", StopSequence: 1, ArrivalSec: 7 * 3600, DepartureSec: 7 * 3600},
		{TripID: "T1", StopID: "X", StopSequence: 2, ArrivalSec: 7*3600 + 300, DepartureSec: 7*3600 + 300},
		{TripID: "T1", StopID: "B", StopSequence: 3, ArrivalSec: 7*3600 + 600, DepartureSec: 7*3600 + 600},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(dir, "stop_times.txt"))
	s := string(raw)
	if !strings.Contains(s, "X") || !strings.Contains(s, "T2") {
		t.Fatalf("%s", s)
	}
	if strings.Count(s, "T1,") != 3 {
		t.Fatalf("T1 rows: %s", s)
	}
}
