package gtfs

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Source est un dossier GTFS importable (plat ou sous-dossier secteur).
type Source struct {
	Sector string // vide = layout historique à plat ; sinon rge/casas/…
	Dir    string
}

// knownSectorDir indique si le nom de dossier est un secteur métier.
func knownSectorDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "rge", "casas", "casc", "forbus", "hombourg-haut", "hombourghaut":
		return true
	default:
		return false
	}
}

// DiscoverSources détecte un feed plat ou des sous-dossiers secteur complets.
func DiscoverSources(root string) ([]Source, error) {
	if DirComplete(root) {
		return []Source{{Sector: "", Dir: root}}, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Source
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !knownSectorDir(name) {
			continue
		}
		dir := filepath.Join(root, name)
		if !DirComplete(dir) {
			continue
		}
		sector := strings.ToLower(strings.TrimSpace(name))
		if sector == "hombourghaut" {
			sector = "hombourg-haut"
		}
		out = append(out, Source{Sector: sector, Dir: dir})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Sector < out[j].Sector })
	return out, nil
}

// HasImportableGTFS est vrai s'il y a un feed plat ou au moins un secteur complet.
func HasImportableGTFS(root string) bool {
	sources, err := DiscoverSources(root)
	return err == nil && len(sources) > 0
}

// PrefixID préfixe un identifiant GTFS par le secteur (évite les collisions inter-feeds).
func PrefixID(sector, id string) string {
	if sector == "" || id == "" {
		return id
	}
	if strings.HasPrefix(id, sector+":") {
		return id
	}
	return sector + ":" + id
}

// StripSectorPrefix sépare secteur et id brut (pour écrire dans GTFS/<secteur>/).
func StripSectorPrefix(id string) (sector, raw string) {
	sector, raw, ok := strings.Cut(id, ":")
	if !ok || sector == "" || raw == "" {
		return "", id
	}
	if !knownSectorDir(sector) {
		return "", id
	}
	return sector, raw
}

// PrefixFeed réécrit tous les IDs d'un feed avec le préfixe secteur.
func PrefixFeed(sector string, feed *Feed) {
	if feed == nil || sector == "" {
		return
	}
	for i := range feed.Agencies {
		feed.Agencies[i].AgencyID = PrefixID(sector, feed.Agencies[i].AgencyID)
	}
	for i := range feed.Routes {
		feed.Routes[i].RouteID = PrefixID(sector, feed.Routes[i].RouteID)
		feed.Routes[i].AgencyID = PrefixID(sector, feed.Routes[i].AgencyID)
	}
	for i := range feed.Trips {
		feed.Trips[i].TripID = PrefixID(sector, feed.Trips[i].TripID)
		feed.Trips[i].RouteID = PrefixID(sector, feed.Trips[i].RouteID)
		feed.Trips[i].ServiceID = PrefixID(sector, feed.Trips[i].ServiceID)
		feed.Trips[i].ShapeID = PrefixID(sector, feed.Trips[i].ShapeID)
	}
	for i := range feed.Stops {
		feed.Stops[i].StopID = PrefixID(sector, feed.Stops[i].StopID)
	}
	for i := range feed.StopTimes {
		feed.StopTimes[i].TripID = PrefixID(sector, feed.StopTimes[i].TripID)
		feed.StopTimes[i].StopID = PrefixID(sector, feed.StopTimes[i].StopID)
	}
	for i := range feed.Calendars {
		feed.Calendars[i].ServiceID = PrefixID(sector, feed.Calendars[i].ServiceID)
	}
	for i := range feed.CalendarDates {
		feed.CalendarDates[i].ServiceID = PrefixID(sector, feed.CalendarDates[i].ServiceID)
	}
	if feed.ShapePoints != nil {
		next := make(map[string][]ShapePoint, len(feed.ShapePoints))
		for id, pts := range feed.ShapePoints {
			next[PrefixID(sector, id)] = pts
		}
		feed.ShapePoints = next
	}
}

// MergeFeeds concatène plusieurs feeds (IDs déjà préfixés si multi-secteurs).
func MergeFeeds(feeds []*Feed) *Feed {
	out := &Feed{ShapePoints: map[string][]ShapePoint{}}
	var versions []string
	for _, f := range feeds {
		if f == nil {
			continue
		}
		if out.Publisher == "" {
			out.Publisher = f.Publisher
			out.StartDate = f.StartDate
			out.EndDate = f.EndDate
		}
		if f.Version != "" {
			versions = append(versions, f.Version)
		}
		out.Agencies = append(out.Agencies, f.Agencies...)
		out.Routes = append(out.Routes, f.Routes...)
		out.Trips = append(out.Trips, f.Trips...)
		out.Stops = append(out.Stops, f.Stops...)
		out.StopTimes = append(out.StopTimes, f.StopTimes...)
		out.Calendars = append(out.Calendars, f.Calendars...)
		out.CalendarDates = append(out.CalendarDates, f.CalendarDates...)
		out.Anomalies = append(out.Anomalies, f.Anomalies...)
		for id, pts := range f.ShapePoints {
			out.ShapePoints[id] = pts
		}
	}
	out.Version = strings.Join(versions, "|")
	return out
}
