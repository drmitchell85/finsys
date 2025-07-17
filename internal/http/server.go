package http

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"log/slog"
	"net/http"

	"github.com/drmitchell85/finsys/internal/config"
	"github.com/drmitchell85/finsys/internal/messenger"
	"github.com/drmitchell85/finsys/internal/store"
	"github.com/go-chi/chi"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	db           *sql.DB
	httpServer   *http.Server
	queueService *messenger.QueueService // for publishing to sqs
	redis        *redis.Client           // for idempotency + distributed locks
	logger       *slog.Logger            // structured logging
	config       *config.Config          // app configuration
	httpClient   *http.Client            // for calling account service
}

func (s *Server) Start() error {
	log.Printf("listening on %s\n", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("error listening and serving: %w", err)
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) {

	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("error shutting down server: %v", err)
	}

	if err := s.db.Close(); err != nil {
		log.Printf("error closing db connection: %v", err)
	}
}

func NewServer() (*Server, error) {
	ctx := context.Background()
	server := Server{}

	config, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %s", err)
	}

	// TODO init db, redis, etc...
	db, err := store.InitDB(*config)
	if err != nil {
		return nil, fmt.Errorf("error starting db: %s", err)
	}
	server.db = db

	rds, err := store.InitCache(ctx, *config)
	if err != nil {
		return nil, fmt.Errorf("error starting cache: %s", err)
	}
	server.redis = rds

	queueService := messenger.NewQueueService(*config)
	server.queueService = queueService

	httpServer, err := initHttpServer(config)
	if err != nil {
		return nil, fmt.Errorf("error starting http server: %s", err)
	}
	server.httpServer = httpServer

	return &server, nil
}

func initHttpServer(config *config.Config) (*http.Server, error) {
	router := chi.NewRouter()
	httpServer := &http.Server{
		Addr:    ":" + fmt.Sprintf("%d", config.Server.Port),
		Handler: router,
	}

	addRoutes(router)

	return httpServer, nil
}
