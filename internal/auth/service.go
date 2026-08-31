package auth

import (
	"context"
	"strings"
	"time"

	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/auth/dto"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const accessTTL = 15 * time.Minute
const refreshTTL = 7 * 24 * time.Hour

// Service authentifie les exploitants admin (pas les chauffeurs).
type Service struct {
	store         Store
	accessSecret  []byte
	refreshSecret []byte
	now           func() time.Time
}

func NewService(store Store, cfg *config.Config) *Service {
	access := []byte(cfg.JWTAccessSecret)
	refresh := []byte(cfg.JWTRefreshSecret)
	if len(access) == 0 {
		access = []byte("dev-access-secret-change-me")
	}
	if len(refresh) == 0 {
		refresh = []byte("dev-refresh-secret-change-me")
	}
	return &Service{
		store:         store,
		accessSecret:  access,
		refreshSecret: refresh,
		now:           time.Now,
	}
}

// SeedAdmin crée le compte .env s'il n'existe pas, et complète opérateur / dépôt s'ils sont vides.
func (s *Service) SeedAdmin(ctx context.Context, cfg *config.Config) error {
	email := strings.ToLower(strings.TrimSpace(cfg.AdminEmail))
	if email == "" || cfg.AdminPassword == "" {
		return nil
	}
	if len(cfg.AdminPassword) < 8 {
		return ErrSeedIncomplete
	}
	existing, err := s.store.FindByEmail(ctx, email)
	if err == nil {
		return s.completeAdminScope(ctx, existing, cfg)
	}
	if err != gorm.ErrRecordNotFound {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.store.CreateUser(ctx, &models.AdminUser{
		ID:           id.New(),
		Email:        email,
		PasswordHash: string(hash),
		OperatorCode: cfg.AdminOperatorCode,
		DepotCode:    cfg.AdminDepotCode,
	})
}

func (s *Service) completeAdminScope(ctx context.Context, user *models.AdminUser, cfg *config.Config) error {
	op := strings.TrimSpace(cfg.AdminOperatorCode)
	depot := strings.TrimSpace(cfg.AdminDepotCode)
	changed := false
	if user.OperatorCode == "" && op != "" {
		user.OperatorCode = op
		changed = true
	}
	if user.DepotCode == "" && depot != "" {
		user.DepotCode = depot
		changed = true
	}
	if !changed {
		return nil
	}
	return s.store.UpdateUser(ctx, user)
}

// Login vérifie email + mot de passe et émet les jetons.
func (s *Service) Login(ctx context.Context, email, password string) (*dto.TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	user, err := s.store.FindByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issue(ctx, user)
}

// Refresh tourne le refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*dto.TokenPair, error) {
	c, err := parseHS256(s.refreshSecret, refreshToken)
	if err != nil || c.Typ != "refresh" {
		return nil, ErrInvalidToken
	}
	row, err := s.store.FindRefresh(ctx, hashToken(refreshToken))
	if err != nil {
		return nil, ErrInvalidToken
	}
	if row.Revoked || s.now().After(row.ExpiresAt) {
		return nil, ErrInvalidToken
	}
	if err := s.store.RevokeRefresh(ctx, row.TokenHash); err != nil {
		return nil, err
	}
	user, err := s.store.FindByID(ctx, row.UserID)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return s.issue(ctx, user)
}

// Logout révoque le refresh.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.store.RevokeRefresh(ctx, hashToken(refreshToken))
}

// ParseAccess valide un Bearer access.
func (s *Service) ParseAccess(token string) (Claims, error) {
	c, err := parseHS256(s.accessSecret, token)
	if err != nil || c.Typ != "access" {
		return Claims{}, ErrUnauthorized
	}
	return c, nil
}

func (s *Service) issue(ctx context.Context, user *models.AdminUser) (*dto.TokenPair, error) {
	now := s.now()
	access, err := signHS256(s.accessSecret, Claims{
		Sub: user.ID, Email: user.Email, OperatorCode: user.OperatorCode,
		DepotCode: user.DepotCode, Typ: "access", JTI: id.New(),
		Iat: now.Unix(), Exp: now.Add(accessTTL).Unix(),
	})
	if err != nil {
		return nil, err
	}
	jti := id.New()
	refresh, err := signHS256(s.refreshSecret, Claims{
		Sub: user.ID, Typ: "refresh", JTI: jti,
		Iat: now.Unix(), Exp: now.Add(refreshTTL).Unix(),
	})
	if err != nil {
		return nil, err
	}
	if err := s.store.SaveRefresh(ctx, newRefreshRow(user.ID, hashToken(refresh), now.Add(refreshTTL))); err != nil {
		return nil, err
	}
	return &dto.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int(accessTTL.Seconds()),
		User: dto.User{
			ID: user.ID, Email: user.Email,
			OperatorCode: user.OperatorCode, DepotCode: user.DepotCode,
		},
	}, nil
}
