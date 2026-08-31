package admin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/admin"
	"github.com/ptijjo/optiligne_back/internal/admin/dto"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authMem struct {
	mu      sync.Mutex
	users   map[string]*models.AdminUser
	byEmail map[string]string
	refresh map[string]*models.RefreshToken
}

func newAuthMem() *authMem {
	return &authMem{users: map[string]*models.AdminUser{}, byEmail: map[string]string{}, refresh: map[string]*models.RefreshToken{}}
}

func (m *authMem) FindByEmail(_ context.Context, email string) (*models.AdminUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	u := *m.users[id]
	return &u, nil
}

func (m *authMem) FindByID(_ context.Context, userID string) (*models.AdminUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *authMem) CreateUser(_ context.Context, user *models.AdminUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	m.byEmail[user.Email] = user.ID
	return nil
}

func (m *authMem) UpdateUser(_ context.Context, user *models.AdminUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	cp := *user
	m.users[user.ID] = &cp
	return nil
}

func (m *authMem) SaveRefresh(_ context.Context, tok *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh[tok.TokenHash] = tok
	return nil
}

func (m *authMem) FindRefresh(_ context.Context, hash string) (*models.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.refresh[hash]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *tok
	return &cp, nil
}

func (m *authMem) RevokeRefresh(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tok, ok := m.refresh[hash]; ok {
		tok.Revoked = true
	}
	return nil
}

func TestHandler_Draft_ExigeJWT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	authSvc := auth.NewService(newAuthMem(), &config.Config{JWTAccessSecret: "a", JWTRefreshSecret: "b"})
	h := admin.NewHandler(admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, nil, nil, ""), authSvc)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/routes/R1?operator_code=transavold&depot_code=fluo57", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandler_Draft_OK(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newAuthMem()
	hash, _ := bcrypt.GenerateFromPassword([]byte("motdepasse1"), bcrypt.MinCost)
	_ = store.CreateUser(context.Background(), &models.AdminUser{
		ID: id.New(), Email: "e@optiligne.test", PasswordHash: string(hash),
		OperatorCode: "transavold", DepotCode: "fluo57",
	})
	authSvc := auth.NewService(store, &config.Config{JWTAccessSecret: "a", JWTRefreshSecret: "b"})
	pair, err := authSvc.Login(context.Background(), "e@optiligne.test", "motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	h := admin.NewHandler(admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, nil, nil, ""), authSvc)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/routes/R1?operator_code=transavold&depot_code=fluo57", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var wrap struct {
		Data dto.Draft `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrap); err != nil {
		t.Fatal(err)
	}
	if wrap.Data.ShortName != "57S012" {
		t.Fatalf("%+v", wrap.Data)
	}
}

func TestHandler_Draft_TripQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newAuthMem()
	hash, _ := bcrypt.GenerateFromPassword([]byte("motdepasse1"), bcrypt.MinCost)
	_ = store.CreateUser(context.Background(), &models.AdminUser{
		ID: id.New(), Email: "e@optiligne.test", PasswordHash: string(hash),
		OperatorCode: "transavold", DepotCode: "fluo57",
	})
	authSvc := auth.NewService(store, &config.Config{JWTAccessSecret: "a", JWTRefreshSecret: "b"})
	pair, err := authSvc.Login(context.Background(), "e@optiligne.test", "motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	drafts := &fakeStore{draft: sampleDraft()}
	h := admin.NewHandler(admin.NewService(drafts, fakeRouter{}, nil, nil, ""), authSvc)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/admin/routes/R1?operator_code=transavold&depot_code=fluo57&trip_id=T9", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	if drafts.lastTripID != "T9" {
		t.Fatalf("tripID = %q", drafts.lastTripID)
	}
}

func TestHandler_Save_JSONInvalide(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store := newAuthMem()
	hash, _ := bcrypt.GenerateFromPassword([]byte("motdepasse1"), bcrypt.MinCost)
	_ = store.CreateUser(context.Background(), &models.AdminUser{
		ID: id.New(), Email: "e@optiligne.test", PasswordHash: string(hash),
		OperatorCode: "transavold", DepotCode: "fluo57",
	})
	authSvc := auth.NewService(store, &config.Config{JWTAccessSecret: "a", JWTRefreshSecret: "b"})
	pair, _ := authSvc.Login(context.Background(), "e@optiligne.test", "motdepasse1")
	h := admin.NewHandler(admin.NewService(&fakeStore{draft: sampleDraft()}, fakeRouter{}, nil, nil, ""), authSvc)
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/routes/R1/save", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}
