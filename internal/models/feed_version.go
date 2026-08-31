package models

import "time"

// FeedVersion identifie un import GTFS.
type FeedVersion struct {
	ID          string    `gorm:"primaryKey;size:32"`
	Checksum    string    `gorm:"uniqueIndex;size:128"`
	Publisher   string    `gorm:"size:255"`
	FeedVersion string    `gorm:"size:64"`
	StartDate   string    `gorm:"size:8"`
	EndDate     string    `gorm:"size:8"`
	Active      bool      `gorm:"index"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (FeedVersion) TableName() string { return "feed_versions" }
