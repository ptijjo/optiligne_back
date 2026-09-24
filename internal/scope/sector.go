package scope

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

const (
	SectorRGE          = "rge"
	SectorCASAS        = "casas"
	SectorCASC         = "casc"
	SectorForbus       = "forbus"
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

// PrimaryDepot retourne le dépôt d'affectation auto pour un secteur.
func PrimaryDepot(sectorOrDepot string) string {
	codes := DepotCodes(sectorOrDepot)
	if len(codes) == 0 {
		return ""
	}
	return codes[0]
}

// LoadBase lit opérateurs + dépôts sans affectations (import multi-secteurs).
func LoadBase(dir string) (*Perimeter, error) {
	ops, err := readOperators(filepath.Join(dir, "operators.csv"))
	if err != nil {
		return nil, err
	}
	deps, err := readDepots(filepath.Join(dir, "depots.csv"))
	if err != nil {
		return nil, err
	}
	return &Perimeter{Operators: ops, Depots: deps}, nil
}

// LoadAssignmentsSoft résout assignments.csv et ignore les lignes absentes du GTFS.
func LoadAssignmentsSoft(p *Perimeter, dir string, routes []gtfs.Route) error {
	if p == nil {
		return nil
	}
	raw, err := readAssignments(filepath.Join(dir, "assignments.csv"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	index := routeIndex(routes)
	seen := make(map[string]struct{}, len(p.Assigns))
	for _, a := range p.Assigns {
		seen[a.RouteID] = struct{}{}
	}
	for _, a := range raw {
		id, ok := index[normalize(a.Ligne)]
		if !ok {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		a.RouteID = id
		p.Assigns = append(p.Assigns, a)
		seen[id] = struct{}{}
	}
	return nil
}

// AssignUnlistedRoutes rattache au dépôt indiqué toute ligne GTFS sans affectation explicite.
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
