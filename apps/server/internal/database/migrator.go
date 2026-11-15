package database

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5"
	tern "github.com/jackc/tern/v2/migrate"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Migrate(ctx context.Context, logger *slog.Logger, cfg *config.DatabaseConfig) error {

	conn, err := pgx.Connect(ctx, buildDSN(cfg))
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	defer conn.Close(ctx)

	migrator, err := tern.NewMigrator(ctx, conn, "schema_version")
	if err != nil {
		return fmt.Errorf("constructing database migrator: %w", err)
	}

	subtree, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("retrieving database migrations subtree: %w", err)
	}
	if err := migrator.LoadMigrations(subtree); err != nil {
		return fmt.Errorf("loading database migrations: %w", err)
	}

	from, err := migrator.GetCurrentVersion(ctx)
	if err != nil {
		return fmt.Errorf("retreiving current database migration version: %w", err)
	}
	if err := migrator.Migrate(ctx); err != nil {
		return fmt.Errorf("running database migrations: %w", err)
	}

	to := len(migrator.Migrations)
	if int(from) == to {
		logger.Info("database schema up to date", "version", from)
	} else {
		logger.Info("migrated database schema", "from", from, "to", to)
	}

	return nil
}
