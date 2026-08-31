package scope_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/scope"
)

func perimDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "perimetres")
}

func gtfsDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("caller")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "test", "fixtures", "gtfs")
}

func TestLoad_ResoutShortName(t *testing.T) {
	feed, err := gtfs.ParseDir(gtfsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := scope.Load(perimDir(t), feed.Routes)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	ids := p.RouteIDs("transavold", "fluo57")
	if len(ids) != 2 {
		t.Fatalf("route ids = %v", ids)
	}
}

func TestLoad_RefuseLigneInconnue(t *testing.T) {
	feed, err := gtfs.ParseDir(gtfsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	mustWrite(t, dir, "operators.csv", "code,name\ntransavold,T\n")
	mustWrite(t, dir, "depots.csv", "code,operator_code,name\nfluo57,transavold,F\n")
	mustWrite(t, dir, "assignments.csv", "operator_code,depot_code,ligne\ntransavold,fluo57,INEXISTANTE\n")
	_, err = scope.Load(dir, feed.Routes)
	if err == nil {
		t.Fatal("attendu échec import ligne inconnue")
	}
}

func TestPerimeter_IsoleAutreTransporteur(t *testing.T) {
	feed, err := gtfs.ParseDir(gtfsDir(t))
	if err != nil {
		t.Fatal(err)
	}
	p, err := scope.Load(perimDir(t), feed.Routes)
	if err != nil {
		t.Fatal(err)
	}
	if ids := p.RouteIDs("autre", "fluo57"); len(ids) != 0 {
		t.Fatalf("fuite: %v", ids)
	}
}

func mustWrite(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
