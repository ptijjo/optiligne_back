package models

import "time"

// RefreshToken est un jeton de rafraîchissement (hashé, révocable).
type RefreshToken struct {
	ID        string    `gorm:"primaryKey;size:32"`
	UserID    string    `gorm:"size:32;index"`
	TokenHash string    `gorm:"uniqueIndex;size:128"`
	ExpiresAt time.Time `gorm:"index"`
	Revoked   bool      `gorm:"index"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

func (RefreshToken) TableName() string { return "refresh_tokens" }
