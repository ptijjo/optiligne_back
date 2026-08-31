package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDownsampleLngLats_GardeExtremites(t *testing.T) {
	in := make([][2]float64, 200)
	for i := range in {
		in[i] = [2]float64{6.9 + float64(i)*0.0001, 49.1}
	}
	out := downsampleLngLats(in, 80)
	if len(out) != 80 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0] != in[0] || out[len(out)-1] != in[len(in)-1] {
		t.Fatalf("extrémités perdues : %+v … %+v", out[0], out[len(out)-1])
	}
}

func TestOSRM_Match_AppelleMatchPasRoute(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "Ok",
			"matchings": []map[string]any{
				{
					"geometry": map[string]any{
						"coordinates": [][]float64{{6.927, 49.2}, {6.927, 49.198}},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewOSRM(srv.URL)
	coords, err := client.Match(context.Background(), [][2]float64{{6.929, 49.2}, {6.927, 49.198}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(gotPath, "/match/v1/driving/") {
		t.Fatalf("path = %s (attendu /match, pas /route)", gotPath)
	}
	if strings.Contains(gotPath, "/route/") {
		t.Fatalf("ne doit pas appeler /route : %s", gotPath)
	}
	if len(coords) != 2 || coords[0][0] != 6.927 {
		t.Fatalf("coords = %+v", coords)
	}
}

func TestOSRM_Match_DecoupeSiTropDePoints(t *testing.T) {
	// Le démo public OSRM refuse les traces longues (TooBig) : on découpe.
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		n := strings.Count(r.URL.Path, ",") // lng,lat → 1 virgule / point
		if n > matchChunkSize {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": "TooBig", "message": "Too many trace coordinates"})
			return
		}
		// Echo : premier et dernier point du chemin (lng,lat;…).
		path := strings.TrimPrefix(r.URL.Path, "/match/v1/driving/")
		parts := strings.Split(path, ";")
		first := strings.Split(parts[0], ",")
		last := strings.Split(parts[len(parts)-1], ",")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": "Ok",
			"matchings": []map[string]any{{
				"geometry": map[string]any{
					"coordinates": [][]float64{
						{parseF(t, first[0]), parseF(t, first[1])},
						{parseF(t, last[0]), parseF(t, last[1])},
					},
				},
			}},
		})
	}))
	t.Cleanup(srv.Close)

	in := make([][2]float64, 25)
	for i := range in {
		in[i] = [2]float64{6.9 + float64(i)*0.001, 49.2}
	}
	client := NewOSRM(srv.URL)
	out, err := client.Match(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("attendu plusieurs appels chunkés, got %d", calls)
	}
	if len(out) < 2 {
		t.Fatalf("coords trop courtes: %+v", out)
	}
	if out[0][0] != in[0][0] || out[len(out)-1][0] != in[len(in)-1][0] {
		t.Fatalf("extrémités perdues: first=%v last=%v", out[0], out[len(out)-1])
	}
}

func parseF(t *testing.T, s string) float64 {
	t.Helper()
	var v float64
	if _, err := fmt.Sscanf(s, "%f", &v); err != nil {
		t.Fatal(err)
	}
	return v
}
