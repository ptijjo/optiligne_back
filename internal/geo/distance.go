package geo

import "math"

// DistanceMeters estime la distance orthodromique (WGS84) en mètres.
func DistanceMeters(lon1, lat1, lon2, lat2 float64) float64 {
	const earthR = 6371000.0
	toRad := math.Pi / 180
	φ1 := lat1 * toRad
	φ2 := lat2 * toRad
	Δφ := (lat2 - lat1) * toRad
	Δλ := (lon2 - lon1) * toRad
	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthR * c
}
