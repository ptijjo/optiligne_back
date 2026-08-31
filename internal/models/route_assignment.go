package models

// RouteAssignment lie une route GTFS à un transporteur + dépôt.
type RouteAssignment struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_assign"`
	OperatorID    string `gorm:"size:32;index"`
	DepotID       string `gorm:"size:32;uniqueIndex:ux_assign"`
	RouteID       string `gorm:"size:64;uniqueIndex:ux_assign"`
}

func (RouteAssignment) TableName() string { return "route_assignments" }
