package guidance_test

import (
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/guidance"
)

func stops() []guidance.StopProg {
	return []guidance.StopProg{
		{Name: "A", Frac: 0.0, ArrivalSec: 8 * 3600, Sequence: 1},
		{Name: "B", Frac: 0.5, ArrivalSec: 8*3600 + 600, Sequence: 2},
		{Name: "C", Frac: 1.0, ArrivalSec: 25*3600 + 15*60, Sequence: 3},
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
	got := guidance.Evaluate(guidance.Input{Frac: 0.2, OffsetM: 90, Stops: stops(), OffRouteM: 80})
	if got.State != "off_route" {
		t.Fatalf("%+v", got)
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
	if got.State != "on_route" && got.NextStop != "C" {
		// 0.99 < 1.0 so next is C
	}
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
