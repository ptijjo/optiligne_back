package importer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/scope"
)

func TestImportResolve_RefuseLigneAbsenteDuGTFS(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	feed, err := gtfs.ParseDir(filepath.Join(root, "test", "fixtures", "gtfs"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	write(t, dir, "operators.csv", "code,name\nx,X\n")
	write(t, dir, "depots.csv", "code,operator_code,name\nd,x,D\n")
	write(t, dir, "assignments.csv", "operator_code,depot_code,ligne\nx,d,PASDANSLEFEED\n")
	_, err = scope.Load(dir, feed.Routes)
	if err == nil {
		t.Fatal("attendu erreur")
	}
}

func TestAssignedShapeIDs_UniquementRoutesAffectees(t *testing.T) {
	perim := &scope.Perimeter{
		Assigns: []scope.Assignment{{RouteID: "R-IN"}},
	}
	got := assignedShapeIDs([]gtfs.Trip{
		{RouteID: "R-IN", ShapeID: "SH-IN"},
		{RouteID: "R-OUT", ShapeID: "SH-OUT"},
	}, perim)
	if _, ok := got["SH-IN"]; !ok {
		t.Fatal("shape de la route affectée manquant")
	}
	if _, ok := got["SH-OUT"]; ok {
		t.Fatal("shape hors périmètre ne doit pas être importé")
	}
}

func TestModelStops_SkipEmptyAndDups(t *testing.T) {
	got := modelStops("feed-1", []gtfs.Stop{
		{StopID: "A", Name: "Premier"},
		{StopID: "", Name: "FORBACH - Foyer"},
		{StopID: "", Name: "FORBACH - Usine"},
		{StopID: "A", Name: "Doublon A"},
		{StopID: "B", Name: "Second"},
	})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].StopID != "A" || got[0].Name != "Premier" {
		t.Fatalf("premier = %+v", got[0])
	}
	if got[1].StopID != "B" || got[1].Name != "Second" {
		t.Fatalf("second = %+v", got[1])
	}
	for _, s := range got {
		if s.FeedVersionID != "feed-1" {
			t.Fatalf("FeedVersionID = %q", s.FeedVersionID)
		}
		if s.ID == "" {
			t.Fatal("id applicatif manquant")
		}
	}
}

func write(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
