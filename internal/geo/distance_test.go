package geo_test

import (
	"math"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/geo"
)

func TestDistanceMeters_SamePoint(t *testing.T) {
	if d := geo.DistanceMeters(6.17, 49.11, 6.17, 49.11); d > 0.01 {
		t.Fatalf("d=%v", d)
	}
}

func TestDistanceMeters_Approx1km(t *testing.T) {
	// ~0.009° lat ≈ 1 km
	d := geo.DistanceMeters(6.17, 49.11, 6.17, 49.119)
	if math.Abs(d-1000) > 50 {
		t.Fatalf("d=%v attendu ~1000m", d)
	}
}
