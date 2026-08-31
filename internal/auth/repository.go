package auth

import (
	"context"
	"time"

	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"gorm.io/gorm"
)

// Store persiste comptes et refresh tokens.
type Store interface {
	FindByEmail(ctx context.Context, email string) (*models.AdminUser, error)
	FindByID(ctx context.Context, userID string) (*models.AdminUser, error)
	CreateUser(ctx context.Context, user *models.AdminUser) error
	UpdateUser(ctx context.Context, user *models.AdminUser) error
	SaveRefresh(ctx context.Context, tok *models.RefreshToken) error
	FindRefresh(ctx context.Context, hash string) (*models.RefreshToken, error)
	RevokeRefresh(ctx context.Context, hash string) error
}

// Repository GORM.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*models.AdminUser, error) {
	var u models.AdminUser
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) FindByID(ctx context.Context, userID string) (*models.AdminUser, error) {
	var u models.AdminUser
	err := r.db.WithContext(ctx).Where("id = ?", userID).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(ctx context.Context, user *models.AdminUser) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *Repository) UpdateUser(ctx context.Context, user *models.AdminUser) error {
	return r.db.WithContext(ctx).Save(user).Error
}

func (r *Repository) SaveRefresh(ctx context.Context, tok *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(tok).Error
}

func (r *Repository) FindRefresh(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var tok models.RefreshToken
	err := r.db.WithContext(ctx).Where("token_hash = ? AND revoked = ?", hash, false).First(&tok).Error
	if err != nil {
		return nil, err
	}
	return &tok, nil
}

func (r *Repository) RevokeRefresh(ctx context.Context, hash string) error {
	return r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("token_hash = ?", hash).Update("revoked", true).Error
}

func newRefreshRow(userID, hash string, exp time.Time) *models.RefreshToken {
	return &models.RefreshToken{
		ID:        id.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: exp,
	}
}
