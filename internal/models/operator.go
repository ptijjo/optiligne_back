package models

// Operator est un transporteur déclaré hors GTFS.
type Operator struct {
	ID   string `gorm:"primaryKey;size:32"`
	Code string `gorm:"uniqueIndex;size:64"`
	Name string `gorm:"size:255"`
}

func (Operator) TableName() string { return "operators" }
