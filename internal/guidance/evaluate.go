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
}

// Input est le résultat d'un snap PostGIS + course.
type Input struct {
	Frac     float64
	OffsetM  float64
	PrevFrac float64
	Stops    []StopProg
	Now      time.Time
	ServiceMidnight time.Time
	OffRouteM float64
}

// Result est l'état de guidage déterministe.
type Result struct {
	Frac     float64
	OffsetM  float64
	NextStop string
	DelayS   int
	State    string
}

// Evaluate applique l'algo de guidage (sans HTTP / SQL).
func Evaluate(in Input) Result {
	const antiRecul = 0.02
	res := Result{Frac: in.Frac, OffsetM: in.OffsetM, State: "on_route"}
	if in.OffRouteM > 0 && in.OffsetM > in.OffRouteM {
		res.State = "off_route"
		return res
	}
	if in.PrevFrac > 0 && in.Frac+antiRecul < in.PrevFrac {
		res.State = "ambiguous"
		res.Frac = in.PrevFrac
		return res
	}
	var next *StopProg
	for i := range in.Stops {
		st := &in.Stops[i]
		if st.Frac > in.Frac {
			next = st
			break
		}
	}
	if next == nil {
		res.State = "arrived"
		if n := len(in.Stops); n > 0 {
			res.NextStop = in.Stops[n-1].Name
		}
		return res
	}
	res.NextStop = next.Name
	arrival := in.ServiceMidnight.Add(time.Duration(next.ArrivalSec) * time.Second)
	res.DelayS = int(in.Now.Sub(arrival).Seconds())
	return res
}
