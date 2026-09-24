package gtfs_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

func fixtureGTFS(t *testing.T) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "gtfs")
}

func copyFixture(t *testing.T, dst string) {
	t.Helper()
	src := fixtureGTFS(t)
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDiscoverSources_FeedPlat(t *testing.T) {
	got, err := gtfs.DiscoverSources(fixtureGTFS(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Sector != "" {
		t.Fatalf("got = %+v", got)
	}
}

func TestDiscoverSources_SousDossiersSecteur(t *testing.T) {
	root := t.TempDir()
	copyFixture(t, filepath.Join(root, "rge"))
	copyFixture(t, filepath.Join(root, "casas"))
	_ = os.Mkdir(filepath.Join(root, "empty"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644)

	got, err := gtfs.DiscoverSources(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2: %+v", len(got), got)
	}
	if got[0].Sector != "casas" || got[1].Sector != "rge" {
		t.Fatalf("order = %+v", got)
	}
	if !gtfs.HasImportableGTFS(root) {
		t.Fatal("HasImportableGTFS attendu true")
	}
}

func TestDiscoverSources_Vide(t *testing.T) {
	got, err := gtfs.DiscoverSources(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("got = %+v", got)
	}
	if gtfs.HasImportableGTFS(t.TempDir()) {
		t.Fatal("HasImportableGTFS attendu false")
	}
}

func TestPrefixFeed_EviteCollisionsStop(t *testing.T) {
	feed := &gtfs.Feed{
		Routes:    []gtfs.Route{{RouteID: "R1", AgencyID: "A1", ShortName: "L1"}},
		Trips:     []gtfs.Trip{{TripID: "T1", RouteID: "R1", ServiceID: "S1", ShapeID: "SH1"}},
		Stops:     []gtfs.Stop{{StopID: "100", Name: "Gare"}},
		StopTimes: []gtfs.StopTime{{TripID: "T1", StopID: "100"}},
		Calendars: []gtfs.Calendar{{ServiceID: "S1"}},
	}
	gtfs.PrefixFeed("casas", feed)
	if feed.Routes[0].RouteID != "casas:R1" {
		t.Fatalf("route = %q", feed.Routes[0].RouteID)
	}
	if feed.Stops[0].StopID != "casas:100" {
		t.Fatalf("stop = %q", feed.Stops[0].StopID)
	}
	if feed.Trips[0].ShapeID != "casas:SH1" {
		t.Fatalf("shape = %q", feed.Trips[0].ShapeID)
	}
	if feed.StopTimes[0].StopID != "casas:100" || feed.StopTimes[0].TripID != "casas:T1" {
		t.Fatalf("stop_times = %+v", feed.StopTimes[0])
	}
}

func TestStripSectorPrefix(t *testing.T) {
	sec, raw := gtfs.StripSectorPrefix("casas:123")
	if sec != "casas" || raw != "123" {
		t.Fatalf("%q %q", sec, raw)
	}
	sec, raw = gtfs.StripSectorPrefix("plain")
	if sec != "" || raw != "plain" {
		t.Fatalf("%q %q", sec, raw)
	}
}

func TestMergeFeeds_ConcateneRoutes(t *testing.T) {
	a := &gtfs.Feed{Version: "A", Routes: []gtfs.Route{{RouteID: "rge:1"}}}
	b := &gtfs.Feed{Version: "B", Routes: []gtfs.Route{{RouteID: "casas:2"}}}
	m := gtfs.MergeFeeds([]*gtfs.Feed{a, b})
	if len(m.Routes) != 2 {
		t.Fatalf("routes = %d", len(m.Routes))
	}
}
