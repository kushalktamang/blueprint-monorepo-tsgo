package config

import (
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/v2"

	// auto-loads .env into the process environment
	_ "github.com/joho/godotenv/autoload"
)

// Root/Parent application configuration
// Built from the BLUEPRINT_* environment variables
type Config struct {
	Primary       Primary              `koanf:"primary" validate:"required"`
	Server        ServerConfig         `koanf:"server" validate:"required"`
	Database      DatabaseConfig       `koanf:"database" validate:"required"`
	Auth          AuthConfig           `koanf:"auth" validate:"required"`
	Redis         RedisConfig          `koanf:"redis" validate:"required"`
	Integration   IntegrationConfig    `koanf:"integration" validate:"required"`
	Observability *ObservabilityConfig `koanf:"observability"`
}

type Primary struct {
	Env string `koanf:"env" validate:"required"`
}

type ServerConfig struct {
	Port               string   `koanf:"port" validate:"required"`
	ReadTimeout        int      `koanf:"read_timeout" validate:"required"`
	WriteTimeout       int      `koanf:"write_timeout" validate:"required"`
	IdleTimeout        int      `koanf:"idle_timeout" validate:"required"`
	CORSAllowedOrigins []string `koanf:"cors_allowed_origins" validate:"required"`
}

type DatabaseConfig struct {
	URL string `koanf:"url"`
	// Using the URL if using a full DSN service (e.g. a Neon connection string).
	// If set, it takes priority and Host/Port/User/Name/SSLMode become optional.
	Host            string `koanf:"host" validate:"required_without=URL"`
	Port            int    `koanf:"port" validate:"required_without=URL"`
	User            string `koanf:"user" validate:"required_without=URL"`
	Password        string `koanf:"password"`
	Name            string `koanf:"name" validate:"required_without=URL"`
	SSLMode         string `koanf:"ssl_mode" validate:"required_without=URL"`
	MaxOpenConns    int    `koanf:"max_open_conns" validate:"gt=0"`
	MaxIdleConns    int    `koanf:"max_idle_conns" validate:"gt=0"`
	ConnMaxLifetime int    `koanf:"conn_max_lifetime" validate:"gt=0"`
	ConnMaxIdleTime int    `koanf:"conn_max_idle_time" validate:"gt=0"`
}

type RedisConfig struct {
	Address string `koanf:"address" validate:"required"`
}

type IntegrationConfig struct {
	ResendAPIKey     string `koanf:"resend_api_key" validate:"required"`
	EmailFromName    string `koanf:"email_from_name" validate:"required"`
	EmailFromAddress string `koanf:"email_from_address" validate:"required,email"`
}

type AuthConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required"`
}

var environmentKeyPaths = map[string]string{
	"primary_env": "primary.env",

	"server_port":                 "server.port",
	"server_read_timeout":         "server.read_timeout",
	"server_write_timeout":        "server.write_timeout",
	"server_idle_timeout":         "server.idle_timeout",
	"server_cors_allowed_origins": "server.cors_allowed_origins",

	"database_url":                "database.url",
	"database_host":               "database.host",
	"database_port":               "database.port",
	"database_user":               "database.user",
	"database_password":           "database.password",
	"database_name":               "database.name",
	"database_ssl_mode":           "database.ssl_mode",
	"database_max_open_conns":     "database.max_open_conns",
	"database_max_idle_conns":     "database.max_idle_conns",
	"database_conn_max_lifetime":  "database.conn_max_lifetime",
	"database_conn_max_idle_time": "database.conn_max_idle_time",

	"auth_secret_key": "auth.secret_key",

	"integration_resend_api_key":     "integration.resend_api_key",
	"integration_email_from_name":    "integration.email_from_name",
	"integration_email_from_address": "integration.email_from_address",

	"redis_address": "redis.address",

	"observability_service_name": "observability.service_name",
	"observability_environment":  "observability.environment",

	"observability_logging_level":                "observability.logging.level",
	"observability_logging_format":               "observability.logging.format",
	"observability_logging_slow_query_threshold": "observability.logging.slow_query_threshold",

	"observability_new_relic_license_key":                 "observability.new_relic.license_key",
	"observability_new_relic_app_log_forwarding_enabled":  "observability.new_relic.app_log_forwarding_enabled",
	"observability_new_relic_distributed_tracing_enabled": "observability.new_relic.distributed_tracing_enabled",
	"observability_new_relic_debug_logging":               "observability.new_relic.debug_logging",

	"observability_health_checks_enabled":  "observability.health_checks.enabled",
	"observability_health_checks_interval": "observability.health_checks.interval",
	"observability_health_checks_timeout":  "observability.health_checks.timeout",
	"observability_health_checks_checks":   "observability.health_checks.checks",
}

func normalizeEnvironmentKey(key string) string {
	key = strings.ToLower(strings.TrimPrefix(key, "BLUEPRINT_"))
	key = strings.ReplaceAll(key, ".", "_")
	if path, ok := environmentKeyPaths[key]; ok {
		return path
	}
	return strings.ReplaceAll(key, "_", ".")
}

func parseList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func applyDatabaseDefaults(cfg *DatabaseConfig) {
	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 25
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 300
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = 300
	}
}

func validateDatabaseURL(rawURL string) error {
	if rawURL == "" {
		return nil
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}
	if parsed.Scheme != "postgres" && parsed.Scheme != "postgresql" {
		return fmt.Errorf("database URL must use postgres:// or postgresql://")
	}
	if parsed.Hostname() == "" {
		return fmt.Errorf("database URL must include a host")
	}
	if strings.Trim(parsed.Path, "/") == "" {
		return fmt.Errorf("database URL must include a database name")
	}
	return nil
}

// This function reads BLUEPRINT_* environment variables into Config and
// returns a validation error without terminating the process.
func MustLoad() (*Config, error) {
	logger := slog.New(slog.NewJSONHandler(
		os.Stderr,
		&slog.HandlerOptions{
			AddSource: true,
			Level:     slog.LevelInfo,
		}))

	// koanf instance
	koanf := koanf.New(".")

	// Load environment variables with prefix "BLUEPRINT_" and merge into config.
	// Example: BLUEPRINT_PARENT1_CHILD1_NAME becomes "parent1.child1.name"
	err := koanf.Load(env.ProviderWithValue("BLUEPRINT_", ".", func(key, value string) (string, any) {
		key = normalizeEnvironmentKey(key)

		if key == "server.cors_allowed_origins" || key == "observability.health_checks.checks" {
			return key, parseList(value)
		}
		return key, value
	}), nil)
	if err != nil {
		logger.Error("could not load initial env variables", "error", err)
		return nil, fmt.Errorf("load environment variables: %w", err)
	}

	mainConfig := &Config{
		Observability: DefaultObservabilityConfig(),
	}

	err = koanf.Unmarshal("", mainConfig)
	if err != nil {
		logger.Error("could not unMarshall main config", "error", err)
		return nil, fmt.Errorf("unmarshal configuration: %w", err)
	}

	if mainConfig.Observability == nil {
		mainConfig.Observability = DefaultObservabilityConfig()
	}
	applyDatabaseDefaults(&mainConfig.Database)
	if err := validateDatabaseURL(mainConfig.Database.URL); err != nil {
		return nil, fmt.Errorf("database.url: %w", err)
	}

	// Override service name and environment from primary config
	mainConfig.Observability.ServiceName = "blueprint"
	mainConfig.Observability.Environment = mainConfig.Primary.Env

	if mainConfig.Integration.EmailFromName == "" {
		mainConfig.Integration.EmailFromName = "Blueprint"
	}
	if mainConfig.Integration.EmailFromAddress == "" {
		mainConfig.Integration.EmailFromAddress = "onboarding@resend.dev"
	}
	if mainConfig.Primary.Env == "production" &&
		(mainConfig.Integration.EmailFromName == "Blueprint" || mainConfig.Integration.EmailFromAddress == "onboarding@resend.dev") {
		return nil, fmt.Errorf("production email sender must be configured explicitly")
	}

	validate := validator.New()
	if err = validate.Struct(mainConfig); err != nil {
		logger.Error("config validation failed", "error", err)
		return nil, fmt.Errorf("configuration validation: %w", err)
	}

	// Validate observability config
	if err := mainConfig.Observability.Validate(); err != nil {
		logger.Error("invalid observability config", "error", err)
		return nil, err
	}

	return mainConfig, nil
}
