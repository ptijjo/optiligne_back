package scope_test

import (
	"testing"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
	"github.com/ptijjo/optiligne_back/internal/scope"
)

func TestNormalizeSector(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"rge", "rge"},
		{"RGE", "rge"},
		{"fluo57", "rge"},
		{"casas", "casas"},
		{"transavold", "casas"},
		{"transchool", "casas"},
		{"casc", "casc"},
		{"forbus", "forbus"},
		{"hombourg-haut", "hombourg-haut"},
	}
	for _, tt := range tests {
		if got := scope.NormalizeSector(tt.in); got != tt.want {
			t.Fatalf("NormalizeSector(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestDepotCodes_CasasUnion(t *testing.T) {
	got := scope.DepotCodes("casas")
	if len(got) != 2 || got[0] != "transavold" || got[1] != "transchool" {
		t.Fatalf("casas = %v", got)
	}
}

func TestDepotCodes_RGE(t *testing.T) {
	got := scope.DepotCodes("rge")
	if len(got) != 1 || got[0] != "fluo57" {
		t.Fatalf("rge = %v", got)
	}
}

func TestDepotCodes_HombourgHaut(t *testing.T) {
	got := scope.DepotCodes("hombourg-haut")
	if len(got) != 1 || got[0] != "hombourg-haut" {
		t.Fatalf("hombourg-haut = %v", got)
	}
}

func TestAssignUnlistedRoutes_AjouteLesRestantesAuRGE(t *testing.T) {
	p := &scope.Perimeter{
		Assigns: []scope.Assignment{
			{OperatorCode: "transavold", DepotCode: "fluo57", RouteID: "R1"},
			{OperatorCode: "transavold", DepotCode: "casc", RouteID: "R-CASC"},
		},
	}
	routes := []gtfs.Route{
		{RouteID: "R1", ShortName: "57SAV34"},
		{RouteID: "R2", ShortName: "57R099"},
		{RouteID: "R-CASC", ShortName: "57EKV00"},
		{RouteID: "R3", ShortName: "57R200"},
	}
	scope.AssignUnlistedRoutes(p, routes, "transavold", "fluo57")

	ids := map[string]string{}
	for _, a := range p.Assigns {
		ids[a.RouteID] = a.DepotCode
	}
	if ids["R2"] != "fluo57" || ids["R3"] != "fluo57" {
		t.Fatalf("restantes absentes: %+v", p.Assigns)
	}
	if ids["R-CASC"] != "casc" {
		t.Fatalf("affectation explicite écrasée: %+v", p.Assigns)
	}
	n := 0
	for _, a := range p.Assigns {
		if a.RouteID == "R1" {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("R1 dupliqué %d fois", n)
	}
}
