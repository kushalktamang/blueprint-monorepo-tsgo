package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/multitracer"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	loggerConfig "github.com/kushalktamang/blueprint-monorepo-tsgo/internal/logger"
	"github.com/newrelic/go-agent/v3/integrations/nrpgx5"
)

type Database struct {
	Pool *pgxpool.Pool
	log  *slog.Logger
}

const DatabasePingTimeout = 10

func New(ctx context.Context, cfg *config.Config, logger *slog.Logger, loggerService *loggerConfig.LoggerService) (*Database, error) {

	pgxPoolConfig, err := pgxpool.ParseConfig(buildDSN(&cfg.Database))
	if err != nil {
		return nil, fmt.Errorf("failed to parse pgx pool config: %w", err)
	}

	maxConns := int32(cfg.Database.MaxOpenConns)
	pgxPoolConfig.MaxConns = maxConns
	// keep a small warm pool, let the rest scale with load
	pgxPoolConfig.MinConns = min(2, maxConns)
	pgxPoolConfig.MaxConnLifetime = time.Duration(cfg.Database.ConnMaxLifetime) * time.Second
	pgxPoolConfig.MaxConnIdleTime = time.Duration(cfg.Database.ConnMaxIdleTime) * time.Second

	// New Relic implements
	//  QueryTracer, BatchTracer, PrepareTracer, and  ConnectTracer
	// keeps a typed handle so we can register it against
	//  each interface below rather than just QueryTracer.
	var nrTracer *nrpgx5.Tracer
	if loggerService != nil && loggerService.GetApplication() != nil {
		nrTracer = nrpgx5.NewTracer()
	}

	var localTracer *tracelog.TraceLog
	if cfg.Primary.Env == "local" {
		level := loggerConfig.ParseLevel(cfg.Observability)
		pgxLogger := loggerConfig.NewPgxLogger(cfg.Observability, level)
		localTracer = &tracelog.TraceLog{
			Logger:   loggerConfig.NewPgxSlogAdapter(pgxLogger),
			LogLevel: loggerConfig.GetPgxTraceLogLevel(level),
		}
	}

	// Combine whichever tracers are active.
	// multitracer dispatches each event
	// (query/batch/prepare/connect) only to tracers that
	// implement the matching interface,
	// so New Relic's batch/prepare/connect instrumentation
	// isn't silently dropped like it would be with a
	// hand-rolled QueryTracer-only wrapper.
	switch {
	case nrTracer != nil && localTracer != nil:
		pgxPoolConfig.ConnConfig.Tracer = multitracer.New(nrTracer, localTracer)
	case nrTracer != nil:
		pgxPoolConfig.ConnConfig.Tracer = nrTracer
	case localTracer != nil:
		pgxPoolConfig.ConnConfig.Tracer = localTracer
	}

	pool, err := pgxpool.NewWithConfig(ctx, pgxPoolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pgx pool: %w", err)
	}

	db := &Database{
		Pool: pool,
		log:  logger,
	}

	ctx, cancel := context.WithTimeout(ctx, DatabasePingTimeout*time.Second)
	defer cancel()
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("connected to the database")

	return db, nil
}

func (db *Database) Close() error {
	db.log.Info("closing database connection pool")
	db.Pool.Close()
	return nil
}
