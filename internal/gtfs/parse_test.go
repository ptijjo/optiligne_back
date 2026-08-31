package gtfs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

func fixtureDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "gtfs")
}

func TestParseDir_ChargeFeedMinimal(t *testing.T) {
	feed, err := gtfs.ParseDir(fixtureDir(t))
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if feed.Version != "test-v1" {
		t.Fatalf("Version = %q", feed.Version)
	}
	if len(feed.Routes) != 3 {
		t.Fatalf("Routes = %d", len(feed.Routes))
	}
	if feed.Routes[0].RouteID == "" {
		t.Fatal("route_id vide")
	}
	if len(feed.ShapePoints["SH1"]) != 4 {
		t.Fatalf("shape points = %d", len(feed.ShapePoints["SH1"]))
	}
	if feed.ShapePoints["SH1"][0].Sequence > feed.ShapePoints["SH1"][1].Sequence {
		t.Fatal("shape points non ordonnés")
	}
}

func TestParseDir_AccepteBOMUtf8(t *testing.T) {
	src := fixtureDir(t)
	dst := t.TempDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if e.Name() == "routes.txt" {
			b = append([]byte{0xEF, 0xBB, 0xBF}, b...)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	feed, err := gtfs.ParseDir(dst)
	if err != nil {
		t.Fatal(err)
	}
	if len(feed.Routes) == 0 || feed.Routes[0].RouteID == "" {
		t.Fatalf("route_id vide malgré BOM: %+v", feed.Routes)
	}
}

func TestParseGTFSTime_HeureApres24h(t *testing.T) {
	sec, err := gtfs.ParseGTFSTime("25:15:00")
	if err != nil {
		t.Fatal(err)
	}
	want := 25*3600 + 15*60
	if sec != want {
		t.Fatalf("sec = %d, want %d", sec, want)
	}
}

func TestFormatGTFSClock(t *testing.T) {
	if got := gtfs.FormatGTFSClock(7*3600 + 15*60); got != "07:15" {
		t.Fatalf("got %q", got)
	}
	if got := gtfs.FormatGTFSClock(25*3600 + 30*60); got != "25:30" {
		t.Fatalf("got %q", got)
	}
}

func TestParseDir_RefuseDossierIncomplet(t *testing.T) {
	_, err := gtfs.ParseDir(t.TempDir())
	if err == nil {
		t.Fatal("attendu une erreur si fichiers manquants")
	}
}

func TestParseDirWithoutShapes_NeChargePasLesPoints(t *testing.T) {
	feed, err := gtfs.ParseDirWithoutShapes(fixtureDir(t))
	if err != nil {
		t.Fatalf("ParseDirWithoutShapes: %v", err)
	}
	if len(feed.Routes) != 3 {
		t.Fatalf("Routes = %d", len(feed.Routes))
	}
	if len(feed.ShapePoints) != 0 {
		t.Fatalf("ShapePoints doit rester vide, got %d", len(feed.ShapePoints))
	}
}

func TestStreamShapes_GroupeEtFiltre(t *testing.T) {
	path := filepath.Join(fixtureDir(t), "shapes.txt")
	var ids []string
	var n int
	err := gtfs.StreamShapes(path, map[string]struct{}{"SH1": {}}, func(id string, pts []gtfs.ShapePoint) error {
		ids = append(ids, id)
		n = len(pts)
		if pts[0].Sequence > pts[1].Sequence {
			t.Fatal("points non ordonnés")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != "SH1" || n != 4 {
		t.Fatalf("ids=%v n=%d", ids, n)
	}
}

func TestLineStringWKT_OrdreLonLat(t *testing.T) {
	pts := []gtfs.ShapePoint{
		{Lat: 49, Lon: 6, Sequence: 0},
		{Lat: 49, Lon: 6.001, Sequence: 1},
	}
	wkt := gtfs.LineStringWKT(pts)
	if wkt != "LINESTRING(6 49,6.001 49)" {
		t.Fatalf("wkt = %q", wkt)
	}
}
