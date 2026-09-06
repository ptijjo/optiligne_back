package guidance_test

import (
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/guidance"
)

func stops() []guidance.StopProg {
	return []guidance.StopProg{
		{Name: "A", Frac: 0.0, ArrivalSec: 8 * 3600, Sequence: 1, Lon: 6.17, Lat: 49.11},
		{Name: "B", Frac: 0.5, ArrivalSec: 8*3600 + 600, Sequence: 2, Lon: 6.18, Lat: 49.12},
		{Name: "C", Frac: 1.0, ArrivalSec: 25*3600 + 15*60, Sequence: 3, Lon: 6.19, Lat: 49.13},
	}
}

func TestEvaluate_OnRoute_ProchainArret(t *testing.T) {
	midnight := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	got := guidance.Evaluate(guidance.Input{
		Frac: 0.2, OffsetM: 5, Stops: stops(),
		Now: midnight.Add(8*time.Hour + 5*time.Minute),
		ServiceMidnight: midnight, OffRouteM: 80,
	})
	if got.State != "on_route" || got.NextStop != "B" {
		t.Fatalf("%+v", got)
	}
}

func TestEvaluate_OffRouteThreshold(t *testing.T) {
	midnight := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	got := guidance.Evaluate(guidance.Input{
		Frac: 0.2, OffsetM: 90, Stops: stops(), OffRouteM: 80,
		Now: midnight.Add(7 * time.Hour), ServiceMidnight: midnight,
		RemainingM: 5000,
	})
	if got.State != "off_route" {
		t.Fatalf("%+v", got)
	}
	if got.NextStop != "A" {
		t.Fatalf("cible premier arrêt, got %+v", got)
	}
	if got.TravelS <= 0 {
		t.Fatalf("travel attendu, got %+v", got)
	}
}

func TestEvaluate_AntiRecul(t *testing.T) {
	got := guidance.Evaluate(guidance.Input{Frac: 0.1, PrevFrac: 0.4, OffsetM: 1, Stops: stops(), OffRouteM: 80})
	if got.State != "ambiguous" {
		t.Fatalf("%+v", got)
	}
}

func TestEvaluate_Terminus(t *testing.T) {
	got := guidance.Evaluate(guidance.Input{Frac: 0.99, OffsetM: 1, Stops: stops(), OffRouteM: 80})
	if got.NextStop != "C" {
		t.Fatalf("%+v", got)
	}
}

func TestEvaluate_Heure24hPlusDelay(t *testing.T) {
	midnight := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	got := guidance.Evaluate(guidance.Input{
		Frac: 0.8, OffsetM: 1, Stops: stops(),
		Now: midnight.Add(25*time.Hour + 20*time.Minute),
		ServiceMidnight: midnight, OffRouteM: 80,
	})
	if got.NextStop != "C" {
		t.Fatalf("next=%s", got.NextStop)
	}
	if got.DelayS < 0 {
		t.Fatalf("retard attendu positif, delay=%d", got.DelayS)
	}
}

func TestEvaluate_DelayPrevisionnel_TrajetLong(t *testing.T) {
	midnight := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	// 7h50, premier arrêt 8h00 → horloge seule = avance 10 min,
	// mais 15 min de route → retard prévisionnel ~5 min.
	got := guidance.Evaluate(guidance.Input{
		Frac: 0, OffsetM: 200, Stops: stops(), OffRouteM: 80,
		Now: midnight.Add(7*time.Hour + 50*time.Minute),
		ServiceMidnight: midnight,
		RemainingM:      guidance.AvgBusSpeedMps * 15 * 60, // 15 min
	})
	if got.State != "off_route" || got.NextStop != "A" {
		t.Fatalf("%+v", got)
	}
	if got.TravelS < 14*60 || got.TravelS > 16*60 {
		t.Fatalf("travel=%d", got.TravelS)
	}
	if got.DelayS < 4*60 || got.DelayS > 6*60 {
		t.Fatalf("delay prévisionnel attendu ~5 min, got %d", got.DelayS)
	}
}

func TestTravelSeconds(t *testing.T) {
	if guidance.TravelSeconds(0) != 0 {
		t.Fatal("zero")
	}
	s := guidance.TravelSeconds(guidance.AvgBusSpeedMps * 120)
	if s != 120 {
		t.Fatalf("got %d", s)
	}
	if guidance.TravelSeconds(1e12) != guidance.MaxTravelS {
		t.Fatal("plafond")
	}
}
