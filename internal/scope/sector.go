package scope

import (
	"strings"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

const (
	SectorRGE         = "rge"
	SectorCASAS       = "casas"
	SectorCASC        = "casc"
	SectorForbus      = "forbus"
	SectorHombourgHaut = "hombourg-haut"
)

// NormalizeSector mappe alias dépôt / type vers un code secteur.
func NormalizeSector(code string) string {
	switch strings.ToLower(strings.TrimSpace(code)) {
	case "rge", "fluo57", "rgefluo57":
		return SectorRGE
	case "casas", "transavold", "transchool":
		return SectorCASAS
	case "casc":
		return SectorCASC
	case "forbus":
		return SectorForbus
	case "hombourg-haut", "hombourghaut":
		return SectorHombourgHaut
	default:
		return strings.ToLower(strings.TrimSpace(code))
	}
}

// DepotCodes retourne les dépôts GTFS d'un secteur (ou du code dépôt tel quel).
func DepotCodes(sectorOrDepot string) []string {
	switch NormalizeSector(sectorOrDepot) {
	case SectorRGE:
		return []string{"fluo57"}
	case SectorCASAS:
		return []string{"transavold", "transchool"}
	case SectorCASC:
		return []string{"casc"}
	case SectorForbus:
		return []string{"forbus"}
	case SectorHombourgHaut:
		return []string{"hombourg-haut"}
	default:
		if sectorOrDepot == "" {
			return nil
		}
		return []string{strings.TrimSpace(sectorOrDepot)}
	}
}

// AssignUnlistedRoutes rattache au dépôt RGE toute ligne GTFS sans affectation explicite.
func AssignUnlistedRoutes(p *Perimeter, routes []gtfs.Route, operatorCode, depotCode string) {
	if p == nil || operatorCode == "" || depotCode == "" {
		return
	}
	seen := make(map[string]struct{}, len(p.Assigns)+len(routes))
	for _, a := range p.Assigns {
		if a.RouteID != "" {
			seen[a.RouteID] = struct{}{}
		}
	}
	for _, r := range routes {
		if r.RouteID == "" {
			continue
		}
		if _, ok := seen[r.RouteID]; ok {
			continue
		}
		p.Assigns = append(p.Assigns, Assignment{
			OperatorCode: operatorCode,
			DepotCode:    depotCode,
			Ligne:        r.ShortName,
			RouteID:      r.RouteID,
		})
		seen[r.RouteID] = struct{}{}
	}
}
