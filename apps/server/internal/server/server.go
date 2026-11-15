package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/database"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/lib/job"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/lib/redisconfig"
	loggerPkg "github.com/kushalktamang/blueprint-monorepo-tsgo/internal/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrredis-v9"
	"github.com/redis/go-redis/v9"
)

type Server struct {
	Config        *config.Config
	Logger        *slog.Logger
	LoggerService *loggerPkg.LoggerService
	DB            *database.Database
	Redis         *redis.Client
	httpServer    *http.Server
	Job           *job.JobService
	cleanupMu     sync.Mutex
	cleanups      []func()
	cleanupOnce   sync.Once
}

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger, loggerService *loggerPkg.LoggerService) (*Server, error) {
	db, err := database.New(ctx, cfg, logger, loggerService)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	redisOptions, err := redisconfig.Parse(cfg.Redis.Address)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("parse redis address: %w", err)
	}

	// Redis client with New Relic integration
	redisClient := redis.NewClient(redisOptions.GoRedis())

	// Add New Relic Redis hooks if available
	if loggerService != nil && loggerService.GetApplication() != nil {
		redisClient.AddHook(nrredis.NewHook(redisClient.Options()))
	}

	// Test Redis connection. Redis is required by the background job service,
	// so startup fails fast when it is unavailable.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := redisClient.Ping(pingCtx).Err(); err != nil {
		_ = redisClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	// job service
	jobService := job.NewJobService(logger, cfg, redisOptions)
	if err := jobService.Start(); err != nil {
		jobService.Stop()
		_ = redisClient.Close()
		_ = db.Close()
		return nil, fmt.Errorf("start job service: %w", err)
	}

	server := &Server{
		Config:        cfg,
		Logger:        logger,
		LoggerService: loggerService,
		DB:            db,
		Redis:         redisClient,
		Job:           jobService,
	}

	// Start metrics collection
	// Runtime metrics are automatically collected by New Relic Go agent

	return server, nil
}

// RegisterCleanup registers a function that runs once during Shutdown.
// It lets packages that build server-owned resources participate in the
// lifecycle without creating an import cycle with the server package.
func (s *Server) RegisterCleanup(cleanup func()) {
	if cleanup == nil {
		return
	}
	s.cleanupMu.Lock()
	defer s.cleanupMu.Unlock()
	s.cleanups = append(s.cleanups, cleanup)
}

func (s *Server) runCleanups() {
	s.cleanupOnce.Do(func() {
		s.cleanupMu.Lock()
		cleanups := append([]func(){}, s.cleanups...)
		s.cleanupMu.Unlock()

		for _, cleanup := range cleanups {
			cleanup()
		}
	})
}

func (s *Server) SetupHTTPServer(handler http.Handler) {
	s.httpServer = &http.Server{
		Addr:         ":" + s.Config.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(s.Config.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(s.Config.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(s.Config.Server.IdleTimeout) * time.Second,
	}
}

func (s *Server) Start() error {
	if s.httpServer == nil {
		return errors.New("HTTP server not initialized")
	}

	s.Logger.Info("starting server",
		slog.String("port", s.Config.Server.Port),
		slog.String("env", s.Config.Primary.Env),
	)

	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErrs []error

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("shutdown http server: %w", err))
		}
	}

	s.runCleanups()

	// Drain background jobs.
	if s.Job != nil {
		s.Job.Stop()
	}

	// Close dependencies. Keep going on failure so nothing leaks.
	if s.Redis != nil {
		if err := s.Redis.Close(); err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("close redis: %w", err))
		}
	}
	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("close database: %w", err))
		}
	}

	return errors.Join(shutdownErrs...)
}
