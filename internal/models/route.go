package models

// Route est une ligne GTFS.
type Route struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_route_feed"`
	RouteID       string `gorm:"size:64;uniqueIndex:ux_route_feed"`
	AgencyID      string `gorm:"size:64"`
	ShortName     string `gorm:"size:64;index"`
	LongName      string `gorm:"size:255"`
	RouteType     int    `gorm:"index"`
}

func (Route) TableName() string { return "routes" }
