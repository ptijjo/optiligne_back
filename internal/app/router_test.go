package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/config"
)

func TestHealth_RetourneOK(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := newRouter(&App{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu %d", w.Code, http.StatusOK)
	}

	var body struct {
		Data struct {
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("json: %v body=%s", err, w.Body.String())
	}
	if body.Data.Status != "ok" {
		t.Fatalf("status = %q, attendu ok", body.Data.Status)
	}
}

func TestHealth_HEAD_RetourneOKSansCorps(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := newRouter(&App{})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodHead, "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, attendu %d", w.Code, http.StatusOK)
	}
	if w.Body.Len() != 0 {
		t.Fatalf("corps HEAD non vide: %q", w.Body.String())
	}
}

func TestCORS_PreflightOrigineAutorisee(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newRouter(&App{cfg: &config.Config{
		CORSOrigins: "https://admin.example.com,http://localhost:3000",
	}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/auth/login", nil)
	req.Header.Set("Origin", "https://admin.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization, Content-Type")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://admin.example.com" {
		t.Fatalf("Allow-Origin = %q", got)
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("Allow-Credentials manquant")
	}
	allow := w.Header().Get("Access-Control-Allow-Headers")
	if !strings.Contains(allow, "Authorization") || !strings.Contains(allow, "Content-Type") {
		t.Fatalf("Allow-Headers = %q", allow)
	}
}

func TestCORS_RefuseOrigineInconnue(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := newRouter(&App{cfg: &config.Config{CORSOrigins: "http://localhost:3000"}})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example")
	r.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("origine inconnue acceptée: %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
}
