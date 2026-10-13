package admin

import (
	"strings"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

// gtfsSector lit le dossier métier (rge, casas, casc, forbus, …) depuis des IDs préfixés.
func gtfsSector(ids ...string) string {
	for _, id := range ids {
		if s, _ := gtfs.StripSectorPrefix(id); s != "" {
			return s
		}
	}
	return ""
}

// withGTFSSector force l’écriture dans GTFS/<secteur>/ pour les IDs ol-* sans préfixe.
func withGTFSSector(sector, id string) string {
	id = strings.TrimSpace(id)
	if sector == "" || id == "" {
		return id
	}
	if s, _ := gtfs.StripSectorPrefix(id); s != "" {
		return id
	}
	return gtfs.PrefixID(sector, id)
}

func withGTFSSectorAll(sector string, ids []string) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = withGTFSSector(sector, id)
	}
	return out
}

// sectorFromFiles choisit le secteur unique du volume multi-feeds (création de ligne).
func sectorFromFiles(files Files) string {
	type sectorer interface {
		DefaultSector() string
	}
	if s, ok := files.(sectorer); ok {
		return s.DefaultSector()
	}
	return ""
}
