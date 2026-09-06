package geo

import (
	"context"

	"gorm.io/gorm"
)

// ShapeLengthM retourne la longueur du shape en mètres (geography).
func ShapeLengthM(ctx context.Context, db *gorm.DB, feedVersionID, shapeID string) float64 {
	if shapeID == "" {
		return 0
	}
	var length float64
	_ = db.WithContext(ctx).Raw(`
		SELECT ST_Length(geom::geography)
		FROM shapes
		WHERE feed_version_id = ? AND shape_id = ?
	`, feedVersionID, shapeID).Scan(&length).Error
	if length < 0 {
		return 0
	}
	return length
}
