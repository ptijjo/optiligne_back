package models

// Calendar est un calendrier de service GTFS.
type Calendar struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_cal_feed"`
	ServiceID     string `gorm:"size:128;uniqueIndex:ux_cal_feed"`
	Monday        bool
	Tuesday       bool
	Wednesday     bool
	Thursday      bool
	Friday        bool
	Saturday      bool
	Sunday        bool
	StartDate     string `gorm:"size:8"`
	EndDate       string `gorm:"size:8"`
}

func (Calendar) TableName() string { return "calendars" }
