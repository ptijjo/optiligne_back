package models

// StopFrac est la fraction [0,1] d'un arrêt sur le shape d'une course.
type StopFrac struct {
	ID            string  `gorm:"primaryKey;size:32"`
	FeedVersionID string  `gorm:"size:32;uniqueIndex:ux_frac_trip_seq"`
	TripID        string  `gorm:"size:128;uniqueIndex:ux_frac_trip_seq"`
	StopID        string  `gorm:"size:64"`
	StopSequence  int     `gorm:"uniqueIndex:ux_frac_trip_seq"`
	Frac          float64 `gorm:"not null"`
	ArrivalSec    int
	StopName      string `gorm:"size:255"`
}

func (StopFrac) TableName() string { return "stop_fracs" }
