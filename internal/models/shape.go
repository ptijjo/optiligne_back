package models

// Shape est le tracé LineString (geom ajouté en SQL PostGIS).
type Shape struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_shape_feed"`
	ShapeID       string `gorm:"size:128;uniqueIndex:ux_shape_feed"`
}

func (Shape) TableName() string { return "shapes" }
