package config_test

import (
	"strings"
	"testing"

	"github.com/ptijjo/optiligne_back/config"
)

func TestGet_LitVariableEnvironnement(t *testing.T) {
	t.Setenv("OPTILIGNE_TEST_KEY", "valeur-test")

	got := config.Get("OPTILIGNE_TEST_KEY")
	if got != "valeur-test" {
		t.Fatalf("Get = %q, attendu %q", got, "valeur-test")
	}
}

func TestGetOr_RetourneDefautSiAbsent(t *testing.T) {
	t.Setenv("OPTILIGNE_TEST_ABSENT", "")

	got := config.GetOr("OPTILIGNE_TEST_ABSENT", "defaut")
	if got != "defaut" {
		t.Fatalf("GetOr = %q, attendu %q", got, "defaut")
	}
}

func TestLoadFromEnv_RefuseSansDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_PORT", "8080")

	_, err := config.LoadFromEnv()
	if err == nil {
		t.Fatal("attendu une erreur si DATABASE_URL est absente")
	}
}

func TestLoadFromEnv_LitPortEtDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5432/optiligne")
	t.Setenv("HTTP_PORT", "8989")
	t.Setenv("APP_ENV", "test")
	t.Setenv("GUIDANCE_OFF_ROUTE_M", "80")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.DatabaseURL != "postgresql://u:p@localhost:5432/optiligne" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
	if cfg.Port != "8989" {
		t.Fatalf("Port = %q", cfg.Port)
	}
	if cfg.AppEnv != "test" {
		t.Fatalf("AppEnv = %q", cfg.AppEnv)
	}
	if cfg.GuidanceOffRouteM != 80 {
		t.Fatalf("GuidanceOffRouteM = %v", cfg.GuidanceOffRouteM)
	}
}

func TestLoadFromEnv_LitPORTSiHTTPPortAbsent(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5437/optiligne")
	t.Setenv("HTTP_PORT", "")
	t.Setenv("PORT", "9191")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.Port != "9191" {
		t.Fatalf("Port = %q, attendu 9191", cfg.Port)
	}
}

func TestLoadFromEnv_RetireGuillemetsDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", `"postgresql://u:p@localhost:5437/optiligne?sslmode=disable"`)
	t.Setenv("PORT", "9191")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if strings.Contains(cfg.DatabaseURL, `"`) {
		t.Fatalf("DATABASE_URL encore quotée: %q", cfg.DatabaseURL)
	}
	if !strings.HasPrefix(cfg.DatabaseURL, "postgresql://") {
		t.Fatalf("DATABASE_URL = %q", cfg.DatabaseURL)
	}
}

func TestLoadFromEnv_LitSeederAdmin(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5437/optiligne")
	t.Setenv("SEEDER_EMAIL", "admin@optiligne.fr")
	t.Setenv("SEEDER_PASSWORD", "motdepasse1")
	t.Setenv("SEEDER_OPERATOR_CODE", "transavold")
	t.Setenv("SEEDER_DEPOT_CODE", "fluo57")
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.AdminEmail != "admin@optiligne.fr" {
		t.Fatalf("AdminEmail = %q", cfg.AdminEmail)
	}
	if cfg.AdminPassword != "motdepasse1" {
		t.Fatalf("AdminPassword = %q", cfg.AdminPassword)
	}
	if cfg.AdminOperatorCode != "transavold" {
		t.Fatalf("AdminOperatorCode = %q", cfg.AdminOperatorCode)
	}
	if cfg.AdminDepotCode != "fluo57" {
		t.Fatalf("AdminDepotCode = %q", cfg.AdminDepotCode)
	}
}

func TestLoadFromEnv_AdminPrimeSurSeeder(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5437/optiligne")
	t.Setenv("ADMIN_EMAIL", "prio@optiligne.fr")
	t.Setenv("ADMIN_PASSWORD", "adminpass1")
	t.Setenv("SEEDER_EMAIL", "seed@optiligne.fr")
	t.Setenv("SEEDER_PASSWORD", "seedpass1")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.AdminEmail != "prio@optiligne.fr" || cfg.AdminPassword != "adminpass1" {
		t.Fatalf("ADMIN_* doit primer: %+v", cfg)
	}
}

func TestLoadFromEnv_LitCORSOrigins(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5437/optiligne")
	t.Setenv("CORS_ORIGINS", "https://admin.example.com,http://localhost:3000")
	t.Setenv("ADMIN_CORS_ORIGINS", "http://localhost:9999")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.CORSOrigins != "https://admin.example.com,http://localhost:3000" {
		t.Fatalf("CORSOrigins = %q", cfg.CORSOrigins)
	}
}

func TestLoadFromEnv_CORSOriginsFallbackAdmin(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://u:p@localhost:5437/optiligne")
	t.Setenv("CORS_ORIGINS", "")
	t.Setenv("ADMIN_CORS_ORIGINS", "http://localhost:5173")

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv: %v", err)
	}
	if cfg.CORSOrigins != "http://localhost:5173" {
		t.Fatalf("CORSOrigins = %q", cfg.CORSOrigins)
	}
}
