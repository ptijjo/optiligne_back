package models

import "gorm.io/gorm"

// AutoMigrate crée les tables applicatives (dev). La colonne geom PostGIS est ajoutée ensuite.
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&FeedVersion{},
		&Agency{},
		&Route{},
		&Trip{},
		&Stop{},
		&StopTime{},
		&Calendar{},
		&CalendarDate{},
		&Shape{},
		&StopFrac{},
		&Operator{},
		&Depot{},
		&RouteAssignment{},
		&AdminUser{},
		&RefreshToken{},
	); err != nil {
		return err
	}
	if err := db.Exec(`ALTER TABLE shapes ADD COLUMN IF NOT EXISTS geom geometry(LineString, 4326)`).Error; err != nil {
		return err
	}
	return db.Exec(`CREATE INDEX IF NOT EXISTS idx_shapes_geom ON shapes USING GIST (geom)`).Error
}
