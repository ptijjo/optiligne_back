package database

import (
	"fmt"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Connect ouvre une connexion PostgreSQL via GORM et vérifie PostGIS.
func Connect(databaseURL string) (*gorm.DB, error) {
	// 1. Valider l'URL fournie (issue du .env via config).
	if strings.TrimSpace(databaseURL) == "" {
		return nil, fmt.Errorf("DATABASE_URL vide")
	}

	// 2. Ouvrir GORM avec le driver Postgres.
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("connexion base de données: %w", err)
	}

	// 3. Vérifier que la connexion répond (ping).
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("accès sql.DB: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	// 4. Fail fast si PostGIS est absent.
	if err := requirePostGIS(db); err != nil {
		return nil, err
	}

	return db, nil
}

func requirePostGIS(db *gorm.DB) error {
	var version string
	if err := db.Raw("SELECT PostGIS_Version()").Scan(&version).Error; err != nil {
		return fmt.Errorf("PostGIS requis: %w", err)
	}
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("PostGIS requis: version vide")
	}
	return nil
}
