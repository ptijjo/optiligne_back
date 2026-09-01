package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
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
