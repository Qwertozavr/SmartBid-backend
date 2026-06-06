package router

import (
	"log/slog"
	"net/http"

	"smartbid-backend/internal/http/handler"
	"smartbid-backend/internal/http/middleware"
)

type Dependencies struct {
	Logger      *slog.Logger
	PingHandler *handler.PingHandler
	AdHandler   *handler.AdHandler
}

func New(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /ping", deps.PingHandler.Ping)
	mux.HandleFunc("POST /api/v1/ads", deps.AdHandler.Create)
	mux.HandleFunc("GET /api/v1/ads/{id}", deps.AdHandler.FindByID)
	mux.HandleFunc("POST /api/v1/ads/{id}/increase", deps.AdHandler.IncreasePrice)
	mux.HandleFunc("POST /api/v1/ads/{id}/publish", deps.AdHandler.Publish)

	return middleware.Logging(deps.Logger)(
		middleware.Recovery(deps.Logger)(
			middleware.SecurityHeaders(mux),
		),
	)
}
