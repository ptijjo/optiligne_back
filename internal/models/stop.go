package models

// Stop est un arrêt GTFS.
type Stop struct {
	ID            string  `gorm:"primaryKey;size:32"`
	FeedVersionID string  `gorm:"size:32;uniqueIndex:ux_stop_feed"`
	StopID        string  `gorm:"size:64;uniqueIndex:ux_stop_feed"`
	Name          string  `gorm:"size:255"`
	Lat           float64 `gorm:"not null"`
	Lon           float64 `gorm:"not null"`
}

func (Stop) TableName() string { return "stops" }
