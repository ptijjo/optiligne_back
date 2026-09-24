package catalog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/internal/catalog/dto"
)

func TestHandler_ListRoutes_SansDepot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := catalog.NewHandler(catalog.NewService(fakeStore{}, ""))
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/catalog/routes", nil))
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}

func TestHandler_ListRoutes_AccepteSector(t *testing.T) {
	gin.SetMode(gin.TestMode)
	h := catalog.NewHandler(catalog.NewService(fakeStore{routes: []dto.Route{
		{ID: "R1", ShortName: "57R004", LongName: "METZ", RouteType: 204},
	}}, ""))
	r := gin.New()
	h.RegisterRoutes(r)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/catalog/routes?operator_code=transavold&sector=rge", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", w.Code, w.Body.String())
	}
}
