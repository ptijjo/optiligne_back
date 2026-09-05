package admin

import (
	"strings"

	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/pkg/id"
)

// interpolateStopTimes aligne la nouvelle séquence sur les horaires connus (interpolation linéaire).
func interpolateStopTimes(old []dto.TimedStop, next []dto.EditorStop) []dto.TimedStop {
	byID := make(map[string]dto.TimedStop, len(old))
	for _, o := range old {
		byID[o.StopID] = o
	}
	out := make([]dto.TimedStop, len(next))
	knownIdx := make([]int, 0, len(next))
	for i, st := range next {
		out[i] = dto.TimedStop{
			StopID: st.StopID, StopSequence: i + 1,
			Name: st.Name, Lat: st.Lat, Lng: st.Lng,
		}
		if o, ok := byID[st.StopID]; ok {
			out[i].ArrivalSec = o.ArrivalSec
			out[i].DepartureSec = o.DepartureSec
			knownIdx = append(knownIdx, i)
		}
	}
	if len(knownIdx) == 0 {
		return out
	}
	// Avant le premier connu : recopier.
	for i := 0; i < knownIdx[0]; i++ {
		out[i].ArrivalSec = out[knownIdx[0]].ArrivalSec
		out[i].DepartureSec = out[knownIdx[0]].DepartureSec
	}
	// Entre deux connus : interpoler.
	for k := 0; k < len(knownIdx)-1; k++ {
		a, b := knownIdx[k], knownIdx[k+1]
		span := b - a
		if span <= 1 {
			continue
		}
		arrA, arrB := out[a].ArrivalSec, out[b].ArrivalSec
		depA, depB := out[a].DepartureSec, out[b].DepartureSec
		for i := a + 1; i < b; i++ {
			t := float64(i-a) / float64(span)
			out[i].ArrivalSec = arrA + int(t*float64(arrB-arrA))
			out[i].DepartureSec = depA + int(t*float64(depB-depA))
		}
	}
	// Après le dernier connu : recopier.
	last := knownIdx[len(knownIdx)-1]
	for i := last + 1; i < len(out); i++ {
		out[i].ArrivalSec = out[last].ArrivalSec
		out[i].DepartureSec = out[last].DepartureSec
	}
	return out
}

func normalizeStops(stops []dto.EditorStop) ([]dto.EditorStop, error) {
	if len(stops) < 2 {
		return nil, ErrTooFewStops
	}
	out := make([]dto.EditorStop, 0, len(stops))
	seen := make(map[string]struct{}, len(stops))
	for i, st := range stops {
		name := strings.TrimSpace(st.Name)
		stopID := strings.TrimSpace(st.StopID)
		if name == "" || !validCoords(st.Lat, st.Lng) {
			return nil, ErrInvalidCoords
		}
		// Arrêt créé dans l'éditeur sans id : CUID, jamais une ligne GTFS vide.
		if stopID == "" {
			stopID = "ol-" + id.New()
		}
		if _, dup := seen[stopID]; dup {
			continue
		}
		seen[stopID] = struct{}{}
		out = append(out, dto.EditorStop{
			StopID: stopID, Name: name, Sequence: i + 1, Lat: st.Lat, Lng: st.Lng,
		})
	}
	if len(out) < 2 {
		return nil, ErrTooFewStops
	}
	for i := range out {
		out[i].Sequence = i + 1
	}
	return out, nil
}
