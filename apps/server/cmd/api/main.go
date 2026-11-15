package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/database"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/handler"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/logger"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/repository"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/router"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/service"
)

const DefaultContextTimeout = 30

func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		fmt.Fprintln(os.Stderr, "failed to load config:", err)
		os.Exit(1)
	}

	// Initialize New Relic logger service
	loggerService := logger.NewLoggerService(cfg.Observability)
	defer loggerService.Shutdown()

	log := logger.NewLoggerWithService(cfg.Observability, loggerService)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := database.Migrate(ctx, log, &cfg.Database); err != nil {
		log.Error("failed to migrate database", "err", err)
		loggerService.Shutdown()
		os.Exit(1)
	}

	// Initialize server
	srv, err := server.New(ctx, cfg, log, loggerService)
	if err != nil {
		log.Error("failed to initialize server", "err", err)
		loggerService.Shutdown()
		os.Exit(1)
	}

	// Initialize repositories, services, and handlers
	repos := repository.NewRepositories(srv)
	services, serviceErr := service.NewServices(srv, repos)
	if serviceErr != nil {
		log.Error("could not create services", "service_error", serviceErr)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
		defer cancel()
		if shutdownErr := srv.Shutdown(shutdownCtx); shutdownErr != nil {
			log.Error("failed to clean up after service initialization", "error", shutdownErr)
		}
		loggerService.Shutdown()
		os.Exit(1)
	}
	handlers := handler.NewHandlers(srv, services)

	// Initialize router
	r := router.NewRouter(srv, handlers, services)

	// Setup HTTP server
	srv.SetupHTTPServer(r)

	// Start server
	serverErr := make(chan error, 1)
	go func() {
		serverErr <- srv.Start()
	}()

	var startErr error
	select {
	case <-ctx.Done():
		// Wait for an interrupt signal to gracefully shut down the server.
	case startErr = <-serverErr:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), DefaultContextTimeout*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("server forced to shutdown", "error", err)
	}
	if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
		log.Error("failed to start server", "error", startErr)
		loggerService.Shutdown()
		os.Exit(1)
	}

	log.Info("server exited properly")
}
