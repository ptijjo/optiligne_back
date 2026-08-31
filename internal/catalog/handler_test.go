package catalog_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/internal/catalog"
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
