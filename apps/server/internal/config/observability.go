package config

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

type ObservabilityConfig struct {
	ServiceName  string             `koanf:"service_name" validate:"required"`
	Environment  string             `koanf:"environment" validate:"required"`
	Logging      LoggingConfig      `koanf:"logging" validate:"required"`
	NewRelic     NewRelicConfig     `koanf:"new_relic" validate:"required"`
	HealthChecks HealthChecksConfig `koanf:"health_checks" validate:"required"`
}

type LoggingConfig struct {
	Level              string        `koanf:"level" validate:"required"`
	Format             string        `koanf:"format" validate:"required"`
	SlowQueryThreshold time.Duration `koanf:"slow_query_threshold"`
}

type NewRelicConfig struct {
	LicenseKey                string `koanf:"license_key"`
	AppLogForwardingEnabled   bool   `koanf:"app_log_forwarding_enabled"`
	DistributedTracingEnabled bool   `koanf:"distributed_tracing_enabled"`
	DebugLogging              bool   `koanf:"debug_logging"`
}

type HealthChecksConfig struct {
	Enabled bool          `koanf:"enabled"`
	Timeout time.Duration `koanf:"timeout" validate:"min=1s"`
	Checks  []string      `koanf:"checks"`
}

// If value is not provided, we use this default observability config to load
func DefaultObservabilityConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		ServiceName: "blueprint",
		Environment: "development",
		Logging: LoggingConfig{
			Level:              "info",
			Format:             "json",
			SlowQueryThreshold: 100 * time.Millisecond,
		},
		NewRelic: NewRelicConfig{
			LicenseKey:                "",
			AppLogForwardingEnabled:   true,
			DistributedTracingEnabled: true,
			// DebugLogging Disabled by default to avoid mixed log formats
			DebugLogging: false,
		},
		HealthChecks: HealthChecksConfig{
			Enabled: true,
			Timeout: 5 * time.Second,
			Checks:  []string{"database", "redis"},
		},
	}
}

// Validate method runs the struct tags above
// (required fields, oneof, min duration, etc.)
// through validator, so every koanf/validate
// tag on this struct is actually enforced.
func (observabilityConfig *ObservabilityConfig) Validate() error {
	if err := validator.New().Struct(observabilityConfig); err != nil {
		return fmt.Errorf("observability config: %w", err)
	}

	return nil
}

// GetLogLevel method returns the configured log level
// falling back to an  environment-appropriate default if none was set.
func (observabilityConfig *ObservabilityConfig) GetLogLevel() string {
	if observabilityConfig.Logging.Level != "" {
		return observabilityConfig.Logging.Level
	}
	if observabilityConfig.Environment == "production" {
		return "info"
	}
	return "debug"
}

func (observabilityConfig *ObservabilityConfig) IsProduction() bool {
	return observabilityConfig.Environment == "production"
}
