package models

// StopTime est un horaire d'arrêt (secondes depuis minuit service).
type StopTime struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_st_trip_seq"`
	TripID        string `gorm:"size:128;uniqueIndex:ux_st_trip_seq"`
	StopID        string `gorm:"size:64;index"`
	StopSequence  int    `gorm:"uniqueIndex:ux_st_trip_seq"`
	ArrivalSec    int
	DepartureSec  int
	ShapeDist     float64
}

func (StopTime) TableName() string { return "stop_times" }
