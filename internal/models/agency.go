package models

// Agency est une agence GTFS.
type Agency struct {
	ID            string `gorm:"primaryKey;size:32"`
	FeedVersionID string `gorm:"size:32;uniqueIndex:ux_agency_feed"`
	AgencyID      string `gorm:"size:64;uniqueIndex:ux_agency_feed"`
	Name          string `gorm:"size:255"`
	Timezone      string `gorm:"size:64"`
}

func (Agency) TableName() string { return "agencies" }
