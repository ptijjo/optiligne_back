package dto_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ptijjo/optiligne_back/internal/guidance/dto"
)

func TestDecodeLineString(t *testing.T) {
	got := dto.DecodeLineString(`{"type":"LineString","coordinates":[[6.1,49.1],[6.2,49.2]]}`)
	if got.Type != "LineString" || len(got.Coordinates) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got.Coordinates[0][0] != 6.1 || got.Coordinates[0][1] != 49.1 {
		t.Fatalf("ordre lon/lat invalide %+v", got.Coordinates[0])
	}
}

func TestDecodeLineString_RefuseInvalide(t *testing.T) {
	if len(dto.DecodeLineString("").Coordinates) != 0 {
		t.Fatal("vide")
	}
	if len(dto.DecodeLineString(`{"type":"Point"}`).Coordinates) != 0 {
		t.Fatal("point")
	}
}

func TestStopPoint_JSONInclutHoraire(t *testing.T) {
	raw, err := json.Marshal(dto.StopPoint{
		Name: "Forbach", Lon: 6.9, Lat: 49.1, ArrivalSec: 26100, Sequence: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{`"arrivalSec":26100`, `"sequence":2`, `"name":"Forbach"`} {
		if !strings.Contains(s, want) {
			t.Fatalf("json=%s manque %s", s, want)
		}
	}
}
