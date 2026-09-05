package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

// Config regroupe les variables typées lues depuis .env / l'environnement.
type Config struct {
	AppEnv            string
	Port              string
	DatabaseURL       string
	WSAllowedOrigins  string
	GuidanceOffRouteM float64
	GTFSDataDir       string
	PerimetresDir     string
	ScopeOperatorID   string
	JWTAccessSecret   string
	JWTRefreshSecret  string
	CORSOrigins       string
	AdminCORSOrigins  string
	AdminEmail        string
	AdminPassword     string
	AdminOperatorCode string
	AdminDepotCode    string
	OSRMURL           string
}

var loadEnvOnce sync.Once

// LoadEnv charge le fichier .env une seule fois.
func LoadEnv() {
	loadEnvOnce.Do(func() {
		for _, p := range envFileCandidates() {
			if err := godotenv.Load(p); err == nil {
				return
			}
		}
	})
}

// Get charge le .env si besoin puis retourne la variable d'environnement.
func Get(key string) string {
	LoadEnv()
	return os.Getenv(key)
}

// GetOr comme Get, avec valeur par défaut si la clé est vide / absente.
func GetOr(key, fallback string) string {
	if v := Get(key); v != "" {
		return v
	}
	return fallback
}

// Load charge le .env puis retourne la config typée validée.
func Load() (*Config, error) {
	LoadEnv()
	return LoadFromEnv()
}

// LoadFromEnv lit uniquement os.Environ (utile pour les tests).
func LoadFromEnv() (*Config, error) {
	offRoute := 80.0
	if raw := os.Getenv("GUIDANCE_OFF_ROUTE_M"); raw != "" {
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil || v <= 0 {
			return nil, fmt.Errorf("GUIDANCE_OFF_ROUTE_M invalide")
		}
		offRoute = v
	}

	corsOrigins := firstNonEmpty(os.Getenv("CORS_ORIGINS"), os.Getenv("ADMIN_CORS_ORIGINS"), "http://localhost:3000")

	cfg := &Config{
		AppEnv:            firstNonEmpty(os.Getenv("APP_ENV"), "development"),
		Port:              firstNonEmpty(os.Getenv("HTTP_PORT"), os.Getenv("PORT"), "8080"),
		DatabaseURL:       trimQuotes(os.Getenv("DATABASE_URL")),
		WSAllowedOrigins:  os.Getenv("WS_ALLOWED_ORIGINS"),
		GuidanceOffRouteM: offRoute,
		GTFSDataDir:       firstNonEmpty(os.Getenv("GTFS_DATA_DIR"), "GTFS"),
		PerimetresDir:     firstNonEmpty(os.Getenv("PERIMETRES_DIR"), filepath.Join("data", "perimetres")),
		ScopeOperatorID:   os.Getenv("SCOPE_OPERATOR_ID"),
		JWTAccessSecret:   os.Getenv("JWT_ACCESS_SECRET"),
		JWTRefreshSecret:  os.Getenv("JWT_REFRESH_SECRET"),
		CORSOrigins:       corsOrigins,
		AdminCORSOrigins:  corsOrigins,
		AdminEmail:        firstNonEmpty(os.Getenv("ADMIN_EMAIL"), os.Getenv("SEEDER_EMAIL")),
		AdminPassword:     firstNonEmpty(os.Getenv("ADMIN_PASSWORD"), os.Getenv("SEEDER_PASSWORD")),
		AdminOperatorCode: firstNonEmpty(os.Getenv("ADMIN_OPERATOR_CODE"), os.Getenv("SEEDER_OPERATOR_CODE"), os.Getenv("SCOPE_OPERATOR_ID")),
		AdminDepotCode:    firstNonEmpty(os.Getenv("ADMIN_DEPOT_CODE"), os.Getenv("SEEDER_DEPOT_CODE")),
		OSRMURL:           firstNonEmpty(os.Getenv("OSRM_URL"), "https://router.project-osrm.org"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL manquante dans l'environnement / .env")
	}

	return cfg, nil
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func trimQuotes(s string) string {
	return strings.Trim(strings.TrimSpace(s), `"'`)
}

func envFileCandidates() []string {
	var paths []string
	dir, err := os.Getwd()
	if err != nil {
		return []string{".env"}
	}
	for range 6 {
		paths = append(paths, filepath.Join(dir, ".env"))
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return paths
}
