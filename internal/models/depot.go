package models

// Depot est un dépôt / régie d'un transporteur.
type Depot struct {
	ID         string `gorm:"primaryKey;size:32"`
	Code       string `gorm:"uniqueIndex:ux_depot_op;size:64"`
	OperatorID string `gorm:"size:32;uniqueIndex:ux_depot_op"`
	Name       string `gorm:"size:255"`
}

func (Depot) TableName() string { return "depots" }
