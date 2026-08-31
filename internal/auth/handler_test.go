package auth_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/auth/dto"
)

func TestHandler_Login_PuisMe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	m := newMem()
	seedUser(t, m)
	svc := auth.NewService(m, testCfg())
	h := auth.NewHandler(svc)
	r := gin.New()
	h.RegisterRoutes(r)

	body, _ := json.Marshal(dto.LoginRequest{Email: "exploitant@optiligne.test", Password: "motdepasse1"})
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body)))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
	var wrap struct {
		Data dto.TokenPair `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &wrap); err != nil {
		t.Fatal(err)
	}

	w2 := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+wrap.Data.AccessToken)
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusOK {
		t.Fatalf("me status = %d body=%s", w2.Code, w2.Body.String())
	}
}

func TestHandler_Login_RefuseSansJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := auth.NewHandler(auth.NewService(newMem(), testCfg()))
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(`{}`))))
	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestHandler_Me_SansBearer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := auth.NewHandler(auth.NewService(newMem(), testCfg()))
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/auth/me", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
}
