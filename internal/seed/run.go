package seed

import (
	"context"
	"fmt"
	"log"

	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/database"
	"github.com/ptijjo/optiligne_back/internal/models"
)

// Run charge la config, migre la DB et crée le compte admin depuis .env s'il n'existe pas.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		return fmt.Errorf("SEEDER_EMAIL/SEEDER_PASSWORD ou ADMIN_EMAIL/ADMIN_PASSWORD requis dans .env")
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	svc := auth.NewService(auth.NewRepository(db), cfg)
	if err := svc.SeedAdmin(context.Background(), cfg); err != nil {
		return err
	}
	log.Printf("compte admin prêt : %s (opérateur=%s, dépôt=%s)", cfg.AdminEmail, cfg.AdminOperatorCode, cfg.AdminDepotCode)
	return nil
}
