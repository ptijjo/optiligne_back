package geo

import (
	"context"

	"gorm.io/gorm"
)

// Snap projette un point WGS84 sur un shape (PostGIS).
func Snap(ctx context.Context, db *gorm.DB, feedVersionID, shapeID string, lon, lat float64) (frac, offsetM float64, err error) {
	row := db.WithContext(ctx).Raw(`
		SELECT
			ST_LineLocatePoint(s.geom, ST_SetSRID(ST_MakePoint(?, ?), 4326)) AS frac,
			ST_Distance(
				s.geom::geography,
				ST_SetSRID(ST_MakePoint(?, ?), 4326)::geography
			) AS offset_m
		FROM shapes s
		WHERE s.feed_version_id = ? AND s.shape_id = ?
	`, lon, lat, lon, lat, feedVersionID, shapeID).Row()
	err = row.Scan(&frac, &offsetM)
	return frac, offsetM, err
}
