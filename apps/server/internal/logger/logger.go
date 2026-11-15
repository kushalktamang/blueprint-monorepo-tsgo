package logger

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/config"
	"github.com/newrelic/go-agent/v3/integrations/logcontext-v2/nrslog"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type LoggerService struct {
	nrApp *newrelic.Application
}

type PgxSlogAdapter struct {
	logger *slog.Logger
}

// NewLoggerService func creates a new logger service with New Relic integration
// -> An empty license key disables New Relic entirely (returns a no-op service).
func NewLoggerService(cfg *config.ObservabilityConfig) *LoggerService {
	service := &LoggerService{}

	if cfg.NewRelic.LicenseKey == "" {
		return service
	}

	var configOptions []newrelic.ConfigOption
	configOptions = append(configOptions,
		newrelic.ConfigAppName(cfg.ServiceName),
		newrelic.ConfigLicense(cfg.NewRelic.LicenseKey),
		newrelic.ConfigAppLogForwardingEnabled(cfg.NewRelic.AppLogForwardingEnabled),
		newrelic.ConfigDistributedTracerEnabled(cfg.NewRelic.DistributedTracingEnabled),
	)

	if cfg.NewRelic.DebugLogging {
		configOptions = append(configOptions, newrelic.ConfigDebugLogger(os.Stdout))
	}

	app, err := newrelic.NewApplication(configOptions...)
	if err != nil {
		// Don't crash the app over APM
		// but surface the failure instead of silently running without New Relic.
		fmt.Fprintf(os.Stderr, "new relic: failed to start, continuing without it: %v\n", err)
		return service
	}

	service.nrApp = app
	return service
}

func (loggerService *LoggerService) Shutdown() {
	if loggerService.nrApp != nil {
		loggerService.nrApp.Shutdown(10 * time.Second)
	}
}

// GetApplication method returns the New Relic application instance
func (loggerService *LoggerService) GetApplication() *newrelic.Application {
	return loggerService.nrApp
}

// NewLoggerWithService creates a logger with full config and logger service
func NewLoggerWithService(cfg *config.ObservabilityConfig, loggerService *LoggerService) *slog.Logger {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.GetLogLevel())); err != nil {
		level = slog.LevelInfo
	}

	options := &slog.HandlerOptions{
		Level:     level,
		AddSource: !cfg.IsProduction(),
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey && a.Value.Kind() == slog.KindTime {
				a.Value = slog.StringValue(a.Value.Time().Format("2006-01-02 15:04:05"))
			}
			return a
		},
	}

	// Format decides the handler (json vs text);
	// production only decides whether it gets wrapped for New Relic forwarding.
	var handler slog.Handler
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	if cfg.IsProduction() && loggerService != nil && loggerService.nrApp != nil {
		handler = nrslog.WrapHandler(loggerService.nrApp, handler)
	}

	logger := slog.New(handler).With(
		slog.String("service", cfg.ServiceName),
		slog.String("environment", cfg.Environment),
	)

	return logger
}

// WithTraceContext adds New Relic transaction context to logger
func WithTraceContext(logger *slog.Logger, txn *newrelic.Transaction) *slog.Logger {
	if txn == nil {
		return logger
	}

	// Get trace metadata from transaction
	metadata := txn.GetTraceMetadata()

	return logger.With(
		slog.String("trace.id", metadata.TraceID),
		slog.String("span.id", metadata.SpanID),
	)
}

// NewPgxLogger func creates a database logger, matching the app's configured
// output format (json in production, console otherwise).
func NewPgxLogger(cfg *config.ObservabilityConfig, level slog.Level) *slog.Logger {
	options := &slog.HandlerOptions{
		AddSource: true,
		Level:     level,
	}

	var handler slog.Handler
	if cfg.Logging.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		options.ReplaceAttr = func(groups []string, slogAttr slog.Attr) slog.Attr {
			if len(groups) == 0 && slogAttr.Key == slog.TimeKey && slogAttr.Value.Kind() == slog.KindTime {
				slogAttr.Value = slog.StringValue(slogAttr.Value.Time().Format("2006-01-02 15:04:05"))
				return slogAttr
			}

			switch slogAttr.Value.Kind() {
			case slog.KindString:
				if s := slogAttr.Value.String(); len(s) > 200 {
					slogAttr.Value = slog.StringValue(s[:200] + "...")
				}
			case slog.KindAny:
				if b, ok := slogAttr.Value.Any().([]byte); ok {
					slogAttr.Value = slog.StringValue(string(b))
				}
			}
			return slogAttr
		}
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	return slog.New(handler).With(slog.String("component", "database"))
}

func NewPgxSlogAdapter(logger *slog.Logger) *PgxSlogAdapter {
	return &PgxSlogAdapter{
		logger: logger,
	}
}

func (slogAdpt *PgxSlogAdapter) Log(ctx context.Context, lvl tracelog.LogLevel, msg string, data map[string]any) {
	attrs := make([]slog.Attr, 0, len(data))
	for k, v := range data {
		attrs = append(attrs, slog.Any(k, v))
	}
	slogAdpt.logger.LogAttrs(ctx, toSlogLevel(lvl), msg, attrs...)
}

func ParseLevel(cfg *config.ObservabilityConfig) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.GetLogLevel())); err != nil {
		return slog.LevelInfo
	}
	return level
}

// GetPgxTraceLogLevel func converts zerolog level to pgx tracelog level
func GetPgxTraceLogLevel(level slog.Level) tracelog.LogLevel {
	switch {
	case level <= slog.LevelDebug:
		return tracelog.LogLevelDebug
	case level <= slog.LevelInfo:
		return tracelog.LogLevelInfo
	case level <= slog.LevelWarn:
		return tracelog.LogLevelWarn
	case level <= slog.LevelError:
		return tracelog.LogLevelError
	default:
		return tracelog.LogLevelNone
	}
}

func toSlogLevel(lvl tracelog.LogLevel) slog.Level {
	switch lvl {
	case tracelog.LogLevelTrace, tracelog.LogLevelDebug:
		return slog.LevelDebug
	case tracelog.LogLevelInfo:
		return slog.LevelInfo
	case tracelog.LogLevelWarn:
		return slog.LevelWarn
	case tracelog.LogLevelError:
		return slog.LevelError
	default:
		return slog.LevelError
	}
}
