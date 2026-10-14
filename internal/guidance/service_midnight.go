package guidance

import (
	"time"
	// Embed IANA tzdb : obligatoire en conteneur Alpine/scratch sans paquet tzdata,
	// sinon LoadLocation("Europe/Paris") échoue et le start guidage renvoie trip_not_found.
	_ "time/tzdata"
)

// ServiceMidnightParis retourne minuit local Europe/Paris pour un jour de service GTFS (YYYY-MM-DD).
// Les arrival_sec / departure_sec du feed sont relatifs à ce minuit (fuseau exploitation France).
func ServiceMidnightParis(date string) (time.Time, error) {
	paris, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return time.Time{}, err
	}
	return time.ParseInLocation("2006-01-02", date, paris)
}
