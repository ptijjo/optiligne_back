package models

// CalendarDate est une exception de calendrier GTFS.
type CalendarDate struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;index"`
	ServiceID     string `gorm:"size:128;index"`
	Date          string `gorm:"size:8;index"`
	ExceptionType int
}

func (CalendarDate) TableName() string { return "calendar_dates" }
