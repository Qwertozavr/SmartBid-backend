package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"smartbid-backend/internal/config"
	"smartbid-backend/internal/event"
	eventkafka "smartbid-backend/internal/event/kafka"
	"smartbid-backend/internal/http/handler"
	"smartbid-backend/internal/http/router"
	"smartbid-backend/internal/price/openrouter"
	"smartbid-backend/internal/repository/postgres"
	"smartbid-backend/internal/service"
	"smartbid-backend/pkg/database"
)

type App struct {
	db        *database.Postgres
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	publisher *eventkafka.Publisher
	handler   http.Handler
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	priceEstimator, err := openrouter.NewClient(openrouter.Config{
		APIKey:  cfg.OpenRouterAPIKey,
		BaseURL: cfg.OpenRouterBaseURL,
		Model:   cfg.OpenRouterModel,
		Timeout: cfg.OpenRouterTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create openrouter client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	transactor := postgres.NewTransactor(db.Pool())
	adRepository := postgres.NewAdRepository(db.Pool())
	outboxRepository := postgres.NewOutboxRepository(db.Pool())
	publisher := eventkafka.NewPublisher(cfg.KafkaBrokers, cfg.KafkaDLQTopic)
	adService := service.NewAdService(
		adRepository,
		priceEstimator,
		outboxRepository,
		transactor,
		cfg.KafkaAdCreatedTopic,
		cfg.KafkaAdFinishedTopic,
	)

	appCtx, cancel := context.WithCancel(context.Background())
	outboxDispatcher := event.NewOutboxDispatcher(
		outboxRepository,
		publisher,
		logger,
		event.OutboxDispatcherConfig{},
	)
	expirationWorker := service.NewAdExpirationWorker(
		adService,
		logger,
		service.AdExpirationWorkerConfig{},
	)

	pingHandler := handler.NewPingHandler()
	adHandler := handler.NewAdHandler(adService, config.CreateAdTimeout(cfg.OpenRouterTimeout))

	httpHandler := router.New(router.Dependencies{
		Logger:      logger,
		PingHandler: pingHandler,
		AdHandler:   adHandler,
	})

	application := &App{
		db:        db,
		cancel:    cancel,
		publisher: publisher,
		handler:   httpHandler,
	}

	application.wg.Add(1)
	go func() {
		defer application.wg.Done()
		outboxDispatcher.Run(appCtx)
	}()
	application.wg.Add(1)
	go func() {
		defer application.wg.Done()
		expirationWorker.Run(appCtx)
	}()

	return application, nil
}

func (a *App) Handler() http.Handler {
	return a.handler
}

func (a *App) Close() {
	if a.cancel != nil {
		a.cancel()
	}
	a.wg.Wait()
	if a.publisher != nil {
		_ = a.publisher.Close()
	}
	if a.db != nil {
		a.db.Close()
	}
}
