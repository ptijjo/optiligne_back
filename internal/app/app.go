package app

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ptijjo/optiligne_back/config"
	"github.com/ptijjo/optiligne_back/internal/admin"
	"github.com/ptijjo/optiligne_back/internal/auth"
	"github.com/ptijjo/optiligne_back/internal/catalog"
	"github.com/ptijjo/optiligne_back/internal/database"
	"github.com/ptijjo/optiligne_back/internal/guidance"
	"github.com/ptijjo/optiligne_back/internal/models"
	"github.com/ptijjo/optiligne_back/internal/ws"
	"gorm.io/gorm"
)

// App agrège les dépendances runtime (wiring hors main).
type App struct {
	cfg      *config.Config
	db       *gorm.DB
	router   *gin.Engine
	auth     *auth.Handler
	catalog  *catalog.Handler
	guidance *guidance.Handler
	admin    *admin.Handler
	hub      *ws.Hub
}

// Run charge la config, monte l'appli et écoute HTTP.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	application, err := New(cfg)
	if err != nil {
		return err
	}
	return application.router.Run(":" + cfg.Port)
}

// New assemble la DB, les modules et le router.
func New(cfg *config.Config) (*App, error) {
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("database: %w", err)
	}
	if err := models.AutoMigrate(db); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}

	catSvc := catalog.NewService(catalog.NewRepository(db), cfg.ScopeOperatorID)
	guideSvc := guidance.NewService(db, guidance.SystemClock{}, cfg.GuidanceOffRouteM, cfg.ScopeOperatorID)
	authSvc := auth.NewService(auth.NewRepository(db), cfg)
	if err := authSvc.SeedAdmin(context.Background(), cfg); err != nil {
		log.Printf("seed admin: %v", err)
	}
	adminSvc := admin.NewService(
		admin.NewRepository(db),
		admin.NewOSRM(cfg.OSRMURL),
		admin.NewGTFSFiles(cfg.GTFSDataDir),
		guideSvc,
		cfg.ScopeOperatorID,
	)

	application := &App{
		cfg:      cfg,
		db:       db,
		auth:     auth.NewHandler(authSvc),
		catalog:  catalog.NewHandler(catSvc),
		guidance: guidance.NewHandler(guideSvc),
		admin:    admin.NewHandler(adminSvc, authSvc),
		hub:      ws.NewHub(guideSvc, cfg.WSAllowedOrigins),
	}
	application.router = newRouter(application)
	return application, nil
}
