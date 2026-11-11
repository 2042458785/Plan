package middleware_test

import (
	"Boutique/src/config"
	"Boutique/src/router"
	"Boutique/src/service"
	"bytes"
	"gorm.io/gorm"
	"net/http/httptest"
	"testing"
)

func TestAdminRoutesRejectWithoutAuthentication(t *testing.T) {
	cfg := &config.AppConfig{Server: config.ServerConfig{Mode: "release"}, JWT: config.JWTConfig{Secret: "unit-test-secret-with-32-characters", Issuer: "unit"}}
	handler := router.SetupRouter(cfg, &gorm.DB{}, &service.CatalogService{}, &service.OrderService{}, &service.AuthService{Config: cfg.JWT})
	for _, item := range []struct{ method, path string }{{"GET", "/api/v1/admin/catalog"}, {"GET", "/api/v1/admin/orders"}, {"GET", "/api/v1/admin/session"}, {"GET", "/api/v1/admin/dashboard"}, {"PATCH", "/api/v1/admin/orders/unknown"}, {"PUT", "/api/v1/admin/products/new"}, {"PUT", "/api/v1/admin/settings"}, {"POST", "/api/v1/admin/logout"}} {
		for _, token := range []string{"", "Bearer forged.invalid.token"} {
			req := httptest.NewRequest(item.method, item.path, bytes.NewBufferString(`{}`))
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", token)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)
			if w.Code != 401 {
				t.Errorf("%s %s: 期望401，实际%d", item.method, item.path, w.Code)
			}
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/admin", nil))
	if w.Code != 404 {
		t.Fatal("API服务不应提供管理页面")
	}
}
func TestCORSRejectsForeignOrigins(t *testing.T) {
	cfg := &config.AppConfig{Server: config.ServerConfig{Mode: "release", AllowOrigins: []string{"http://localhost:8082"}}}
	handler := router.SetupRouter(cfg, &gorm.DB{}, &service.CatalogService{}, &service.OrderService{}, &service.AuthService{})
	for _, test := range []struct {
		origin   string
		expected int
	}{{"https://foreign.invalid", 403}, {"http://localhost:8082", 204}} {
		req := httptest.NewRequest("OPTIONS", "/api/v1/admin/catalog", nil)
		req.Header.Set("Origin", test.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code != test.expected {
			t.Fatalf("Origin %s: %d", test.origin, w.Code)
		}
	}
}
