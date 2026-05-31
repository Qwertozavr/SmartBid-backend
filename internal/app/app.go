package app

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"smartbid-backend/internal/config"
	"smartbid-backend/internal/http/handler"
	"smartbid-backend/internal/http/router"
	"smartbid-backend/internal/repository/postgres"
	"smartbid-backend/internal/service"
	"smartbid-backend/pkg/database"
)

type App struct {
	db      *database.Postgres
	handler http.Handler
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	adRepository := postgres.NewAdRepository(db.Pool())
	adService := service.NewAdService(adRepository)

	pingHandler := handler.NewPingHandler()
	adHandler := handler.NewAdHandler(adService)

	httpHandler := router.New(router.Dependencies{
		Logger:      logger,
		PingHandler: pingHandler,
		AdHandler:   adHandler,
	})

	return &App{
		db:      db,
		handler: httpHandler,
	}, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() {
	if a.db != nil {
		a.db.Close()
	}
}
