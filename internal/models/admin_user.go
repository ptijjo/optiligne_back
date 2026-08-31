package models

import "time"

// AdminUser est un compte exploitant (console admin uniquement).
type AdminUser struct {
	ID           string    `gorm:"primaryKey;size:32"`
	Email        string    `gorm:"uniqueIndex;size:255"`
	PasswordHash string    `gorm:"size:255"`
	OperatorCode string    `gorm:"size:64;index"`
	DepotCode    string    `gorm:"size:64;index"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime"`
}

func (AdminUser) TableName() string { return "admin_users" }
