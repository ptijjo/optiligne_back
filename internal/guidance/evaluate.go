package guidance

import "time"

// Clock injectable (interdit time.Now dans le métier).
type Clock interface {
	Now() time.Time
}

// SystemClock utilise l'horloge système.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// StopProg est un arrêt le long du shape.
type StopProg struct {
	Name       string
	Frac       float64
	ArrivalSec int
	Sequence   int
	Lon        float64
	Lat        float64
}

// Vitesse bus urbaine estimée (25 km/h) — pas un routage trafic.
const AvgBusSpeedMps = 25.0 / 3.6

// Plafond ETA (2 h).
const MaxTravelS = 2 * 3600

// Input est le résultat d'un snap PostGIS + course.
type Input struct {
	Frac            float64
	OffsetM         float64
	PrevFrac        float64
	Stops           []StopProg
	Now             time.Time
	ServiceMidnight time.Time
	OffRouteM       float64
	RemainingM      float64
}

// Result est l'état de guidage déterministe.
type Result struct {
	Frac     float64
	OffsetM  float64
	NextStop string
	DelayS   int
	TravelS  int
	State    string
}

// TravelSeconds convertit une distance restante en secondes de route estimées.
func TravelSeconds(remainingM float64) int {
	if remainingM <= 0 || !isFinite(remainingM) {
		return 0
	}
	s := int(remainingM / AvgBusSpeedMps)
	if s > MaxTravelS {
		return MaxTravelS
	}
	return s
}

func isFinite(v float64) bool {
	return !((v != v) || v > 1e15 || v < -1e15)
}

// Evaluate applique l'algo de guidage (sans HTTP / SQL).
func Evaluate(in Input) Result {
	const antiRecul = 0.02
	res := Result{Frac: in.Frac, OffsetM: in.OffsetM, State: "on_route"}

	if in.PrevFrac > 0 && in.Frac+antiRecul < in.PrevFrac &&
		!(in.OffRouteM > 0 && in.OffsetM > in.OffRouteM) {
		res.State = "ambiguous"
		res.Frac = in.PrevFrac
		return res
	}

	offRoute := in.OffRouteM > 0 && in.OffsetM > in.OffRouteM
	if offRoute {
		res.State = "off_route"
	}

	var target *StopProg
	if offRoute {
		if n := len(in.Stops); n > 0 {
			target = &in.Stops[0]
		}
	} else {
		for i := range in.Stops {
			st := &in.Stops[i]
			if st.Frac > in.Frac {
				target = st
				break
			}
		}
	}

	if target == nil {
		if !offRoute {
			res.State = "arrived"
		}
		if n := len(in.Stops); n > 0 {
			res.NextStop = in.Stops[n-1].Name
		}
		return res
	}

	res.NextStop = target.Name
	res.TravelS = TravelSeconds(in.RemainingM)
	arrival := in.ServiceMidnight.Add(time.Duration(target.ArrivalSec) * time.Second)
	eta := in.Now.Add(time.Duration(res.TravelS) * time.Second)
	res.DelayS = int(eta.Sub(arrival).Seconds())
	return res
}
