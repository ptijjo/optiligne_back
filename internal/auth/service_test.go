package auth_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/pkg/id"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type memStore struct {
	mu       sync.Mutex
	users    map[string]*models.AdminUser
	byEmail  map[string]string
	refresh  map[string]*models.RefreshToken
}

func newMem() *memStore {
	return &memStore{
		users:   map[string]*models.AdminUser{},
		byEmail: map[string]string{},
		refresh: map[string]*models.RefreshToken{},
	}
}

func (m *memStore) FindByEmail(_ context.Context, email string) (*models.AdminUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id, ok := m.byEmail[email]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	u := *m.users[id]
	return &u, nil
}

func (m *memStore) FindByID(_ context.Context, userID string) (*models.AdminUser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	u, ok := m.users[userID]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *u
	return &cp, nil
}

func (m *memStore) CreateUser(_ context.Context, user *models.AdminUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.ID] = user
	m.byEmail[user.Email] = user.ID
	return nil
}

func (m *memStore) UpdateUser(_ context.Context, user *models.AdminUser) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	cp := *user
	m.users[user.ID] = &cp
	return nil
}

func (m *memStore) SaveRefresh(_ context.Context, tok *models.RefreshToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refresh[tok.TokenHash] = tok
	return nil
}

func (m *memStore) FindRefresh(_ context.Context, hash string) (*models.RefreshToken, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	tok, ok := m.refresh[hash]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *tok
	return &cp, nil
}

func (m *memStore) RevokeRefresh(_ context.Context, hash string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tok, ok := m.refresh[hash]; ok {
		tok.Revoked = true
	}
	return nil
}

func seedUser(t *testing.T, m *memStore) *models.AdminUser {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("motdepasse1"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	u := &models.AdminUser{
		ID: id.New(), Email: "exploitant@optiligne.test",
		PasswordHash: string(hash), OperatorCode: "transavold", DepotCode: "fluo57",
	}
	if err := m.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
	return u
}

func testCfg() *config.Config {
	return &config.Config{
		JWTAccessSecret:  "access-test",
		JWTRefreshSecret: "refresh-test",
	}
}

func TestLogin_MauvaisMotDePasse(t *testing.T) {
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	_, err := svc.Login(context.Background(), "exploitant@optiligne.test", "wrongpass")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("err = %v", err)
	}
}

func TestLogin_EmailInconnu(t *testing.T) {
	svc := auth.NewService(newMem(), testCfg())
	_, err := svc.Login(context.Background(), "inconnu@optiligne.test", "motdepasse1")
	if err != auth.ErrInvalidCredentials {
		t.Fatalf("err = %v", err)
	}
}

func TestLogin_OK_PuisRefresh(t *testing.T) {
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	pair, err := svc.Login(context.Background(), "exploitant@optiligne.test", "motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" || pair.User.OperatorCode != "transavold" {
		t.Fatalf("pair %+v", pair)
	}
	cl, err := svc.ParseAccess(pair.AccessToken)
	if err != nil || cl.Email != "exploitant@optiligne.test" {
		t.Fatalf("parse %v %+v", err, cl)
	}
	next, err := svc.Refresh(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if next.AccessToken == pair.AccessToken {
		t.Fatal("le refresh doit émettre un nouvel access")
	}
	_, err = svc.Refresh(context.Background(), pair.RefreshToken)
	if err != auth.ErrInvalidToken {
		t.Fatalf("ancien refresh encore valide: %v", err)
	}
}

func TestParseAccess_RefuseRefresh(t *testing.T) {
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	pair, err := svc.Login(context.Background(), "exploitant@optiligne.test", "motdepasse1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ParseAccess(pair.RefreshToken); err != auth.ErrUnauthorized {
		t.Fatalf("err = %v", err)
	}
}

func TestSeedAdmin_CreeUneFois(t *testing.T) {
	m := newMem()
	svc := auth.NewService(m, testCfg())
	cfg := &config.Config{
		AdminEmail: "seed@optiligne.test", AdminPassword: "motdepasse1",
		AdminOperatorCode: "transavold", AdminDepotCode: "fluo57",
	}
	if err := svc.SeedAdmin(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	if err := svc.SeedAdmin(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	u, err := m.FindByEmail(context.Background(), "seed@optiligne.test")
	if err != nil || u.DepotCode != "fluo57" {
		t.Fatalf("%v %+v", err, u)
	}
}

func TestSeedAdmin_CompleteDepotManquant(t *testing.T) {
	m := newMem()
	hash, err := bcrypt.GenerateFromPassword([]byte("motdepasse1"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.CreateUser(context.Background(), &models.AdminUser{
		ID: id.New(), Email: "seed@optiligne.test", PasswordHash: string(hash),
		OperatorCode: "transavold", DepotCode: "",
	}); err != nil {
		t.Fatal(err)
	}
	svc := auth.NewService(m, testCfg())
	cfg := &config.Config{
		AdminEmail: "seed@optiligne.test", AdminPassword: "motdepasse1",
		AdminOperatorCode: "transavold", AdminDepotCode: "fluo57",
	}
	if err := svc.SeedAdmin(context.Background(), cfg); err != nil {
		t.Fatal(err)
	}
	u, err := m.FindByEmail(context.Background(), "seed@optiligne.test")
	if err != nil || u.DepotCode != "fluo57" {
		t.Fatalf("dépôt non complété: %v %+v", err, u)
	}
}

func TestLogout_Revoque(t *testing.T) {
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	pair, _ := svc.Login(context.Background(), "exploitant@optiligne.test", "motdepasse1")
	if err := svc.Logout(context.Background(), pair.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Refresh(context.Background(), pair.RefreshToken); err != auth.ErrInvalidToken {
		t.Fatalf("err = %v", err)
	}
}

func TestParseAccess_Expire(t *testing.T) {
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	pair, _ := svc.Login(context.Background(), "exploitant@optiligne.test", "motdepasse1")
	time.Sleep(10 * time.Millisecond)
	if _, err := svc.ParseAccess(pair.AccessToken); err != nil {
		t.Fatal(err)
	}
}
