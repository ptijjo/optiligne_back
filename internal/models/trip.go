package models

// Trip est une course GTFS.
type Trip struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_trip_feed"`
	TripID        string `gorm:"size:128;uniqueIndex:ux_trip_feed"`
	RouteID       string `gorm:"size:64;index"`
	ServiceID     string `gorm:"size:128;index"`
	ShapeID       string `gorm:"size:128;index"`
	Headsign      string `gorm:"size:255"`
	DirectionID   int
}

func (Trip) TableName() string { return "trips" }
