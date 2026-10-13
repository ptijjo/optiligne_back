package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGTFSSector_DepuisPrefixe(t *testing.T) {
	if got := gtfsSector("ol-x", "rge:123", "casas:9"); got != "rge" {
		t.Fatalf("got %q", got)
	}
	if got := gtfsSector("casc:R1"); got != "casc" {
		t.Fatalf("got %q", got)
	}
	if got := gtfsSector("forbus:T1", "ol-y"); got != "forbus" {
		t.Fatalf("got %q", got)
	}
	if got := gtfsSector("ol-x", "plain"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestWithGTFSSector_PrefixeOl(t *testing.T) {
	if got := withGTFSSector("rge", "ol-new"); got != "rge:ol-new" {
		t.Fatalf("got %q", got)
	}
	if got := withGTFSSector("rge", "rge:266"); got != "rge:266" {
		t.Fatalf("ne doit pas double-préfixer: %q", got)
	}
	if got := withGTFSSector("", "ol-new"); got != "ol-new" {
		t.Fatalf("got %q", got)
	}
}

func TestGTFSFiles_UpsertStop_DossierSecteur(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "rge")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "stop_id,stop_name,stop_lat,stop_lon\nA,Depart,49.100000,6.900000\n"
	if err := os.WriteFile(filepath.Join(dir, "stops.txt"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	files := NewGTFSFiles(root)
	if err := files.UpsertStop(withGTFSSector("rge", "ol-x"), "Nouveau", 49.2, 6.92); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, "stops.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "ol-x") || !strings.Contains(string(raw), "Nouveau") {
		t.Fatalf("%s", raw)
	}
	if _, err := os.Stat(filepath.Join(root, "stops.txt")); !os.IsNotExist(err) {
		t.Fatal("ne doit pas créer stops.txt à la racine")
	}
}
