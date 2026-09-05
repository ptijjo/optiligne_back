package admin

import (
	"strings"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/admin/dto"
)

func TestInterpolateStopTimes_ConserveConnus(t *testing.T) {
	old := []dto.TimedStop{
		{StopID: "A", StopSequence: 1, ArrivalSec: 3600, DepartureSec: 3600},
		{StopID: "B", StopSequence: 2, ArrivalSec: 3700, DepartureSec: 3700},
		{StopID: "C", StopSequence: 3, ArrivalSec: 3800, DepartureSec: 3800},
	}
	next := []dto.EditorStop{
		{StopID: "A", Name: "A", Sequence: 1, Lat: 49.1, Lng: 6.9},
		{StopID: "X", Name: "X", Sequence: 2, Lat: 49.11, Lng: 6.91},
		{StopID: "C", Name: "C", Sequence: 3, Lat: 49.12, Lng: 6.92},
	}
	got := interpolateStopTimes(old, next)
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].ArrivalSec != 3600 || got[2].ArrivalSec != 3800 {
		t.Fatalf("%+v", got)
	}
	if got[1].ArrivalSec != 3700 {
		t.Fatalf("interp X = %d (attendu 3700)", got[1].ArrivalSec)
	}
}

func TestInterpolateStopTimes_AjouteEnFin(t *testing.T) {
	old := []dto.TimedStop{
		{StopID: "A", StopSequence: 1, ArrivalSec: 3600, DepartureSec: 3600},
		{StopID: "B", StopSequence: 2, ArrivalSec: 3700, DepartureSec: 3700},
	}
	next := []dto.EditorStop{
		{StopID: "A", Name: "A", Sequence: 1, Lat: 49.1, Lng: 6.9},
		{StopID: "B", Name: "B", Sequence: 2, Lat: 49.11, Lng: 6.91},
		{StopID: "Z", Name: "Z", Sequence: 3, Lat: 49.12, Lng: 6.92},
	}
	got := interpolateStopTimes(old, next)
	if got[2].ArrivalSec != 3700 {
		t.Fatalf("fin = %d", got[2].ArrivalSec)
	}
}

func TestNormalizeStops_RefuseMoinsDeDeux(t *testing.T) {
	_, err := normalizeStops([]dto.EditorStop{
		{StopID: "A", Name: "A", Sequence: 1, Lat: 49.1, Lng: 6.9},
	})
	if err != ErrTooFewStops {
		t.Fatalf("err = %v", err)
	}
}

func TestNormalizeStops_AssigneIDSiVide(t *testing.T) {
	got, err := normalizeStops([]dto.EditorStop{
		{StopID: "A", Name: "Depart", Sequence: 1, Lat: 49.1, Lng: 6.9},
		{StopID: "", Name: "Nouveau", Sequence: 2, Lat: 49.11, Lng: 6.91},
		{StopID: "B", Name: "Ecole", Sequence: 3, Lat: 49.12, Lng: 6.92},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[1].StopID == "" || !strings.HasPrefix(got[1].StopID, "ol-") {
		t.Fatalf("stop_id généré = %q", got[1].StopID)
	}
	if got[0].StopID != "A" || got[2].StopID != "B" {
		t.Fatalf("%+v", got)
	}
}
