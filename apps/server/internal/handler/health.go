package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
)

const checkTimeoutDefault = 5 * time.Second

type checkResult struct {
	Status       string `json:"status"`
	ResponseTime string `json:"response_time"`
	Error        string `json:"error,omitempty"`
}

type healthResponse struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Environment string                 `json:"environment"`
	Checks      map[string]checkResult `json:"checks"`
}

type HealthHandler struct {
	Handler
}

func NewHealthHandler(s *server.Server) *HealthHandler {
	return &HealthHandler{
		Handler: NewHandler(s),
	}
}

func (h *HealthHandler) CheckHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	logger := h.server.Logger.With(slog.String("operation", "health_check"))

	resp := healthResponse{
		Status:      "healthy",
		Timestamp:   time.Now().UTC(),
		Environment: h.server.Config.Primary.Env,
		Checks:      make(map[string]checkResult),
	}

	healthy := true
	healthChecksEnabled := true
	checkTimeout := checkTimeoutDefault
	checks := []string{"database", "redis"}
	if h.server.Config.Observability != nil {
		healthChecks := h.server.Config.Observability.HealthChecks
		healthChecksEnabled = healthChecks.Enabled
		if healthChecks.Timeout > 0 {
			checkTimeout = healthChecks.Timeout
		}
		if len(healthChecks.Checks) > 0 {
			checks = healthChecks.Checks
		}
	}

	if healthChecksEnabled {

		for _, name := range checks {
			var checkOK bool
			switch name {
			case "database":
				if h.server.DB == nil || h.server.DB.Pool == nil {
					checkOK = h.recordUnavailableCheck(logger, &resp, name)
				} else {
					checkOK = h.runCheck(r.Context(), logger, &resp, name, checkTimeout, h.server.DB.Pool.Ping)
				}
			case "redis":
				if h.server.Redis == nil {
					checkOK = h.recordUnavailableCheck(logger, &resp, name)
				} else {
					checkOK = h.runCheck(r.Context(), logger, &resp, name, checkTimeout, func(ctx context.Context) error {
						return h.server.Redis.Ping(ctx).Err()
					})
				}
			default:
				checkOK = h.recordUnsupportedCheck(logger, &resp, name)
			}
			healthy = healthy && checkOK
		}
	}

	status := http.StatusOK
	if !healthy {
		resp.Status = "unhealthy"
		status = http.StatusServiceUnavailable
		logger.Warn("health check failed", slog.Duration("total_duration", time.Since(start)))
		h.recordEvent(map[string]any{
			"check_type":        "overall",
			"operation":         "health_check",
			"error_type":        "overall_unhealthy",
			"total_duration_ms": time.Since(start).Milliseconds(),
		})
	} else {
		logger.Info("health check passed", slog.Duration("total_duration", time.Since(start)))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		// Status line is already sent, so just log it.
		logger.Error("write health response", slog.Any("error", err))
		h.recordEvent(map[string]any{
			"check_type":    "response",
			"operation":     "health_check",
			"error_type":    "json_response_error",
			"error_message": err.Error(),
		})
	}
}

// runCheck runs one dependency check, records its result, and reports whether it passed.
func (h *HealthHandler) runCheck(
	parent context.Context,
	logger *slog.Logger,
	resp *healthResponse,
	name string,
	timeout time.Duration,
	ping func(context.Context) error,
) bool {
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()

	start := time.Now()
	err := ping(ctx)
	elapsed := time.Since(start)

	if err != nil {
		resp.Checks[name] = checkResult{
			Status:       "unhealthy",
			ResponseTime: elapsed.String(),
			Error:        "dependency unavailable",
		}
		logger.Error(name+" health check failed",
			slog.Duration("response_time", elapsed),
			slog.Any("error", err),
		)
		h.recordEvent(map[string]any{
			"check_type":       name,
			"operation":        "health_check",
			"error_type":       name + "_unhealthy",
			"response_time_ms": elapsed.Milliseconds(),
			"error_message":    err.Error(),
		})
		return false
	}

	resp.Checks[name] = checkResult{Status: "healthy", ResponseTime: elapsed.String()}
	logger.Info(name+" health check passed", slog.Duration("response_time", elapsed))
	return true
}

func (h *HealthHandler) recordUnavailableCheck(logger *slog.Logger, resp *healthResponse, name string) bool {
	resp.Checks[name] = checkResult{
		Status: "unhealthy",
		Error:  "dependency unavailable",
	}
	logger.Error(name + " health check unavailable")
	h.recordEvent(map[string]any{
		"check_type": name,
		"operation":  "health_check",
		"error_type": name + "_unavailable",
	})
	return false
}

func (h *HealthHandler) recordUnsupportedCheck(logger *slog.Logger, resp *healthResponse, name string) bool {
	resp.Checks[name] = checkResult{
		Status: "unhealthy",
		Error:  "unsupported health check",
	}
	logger.Error("unsupported health check", slog.String("check", name))
	return false
}

func (h *HealthHandler) recordEvent(attrs map[string]any) {
	ls := h.server.LoggerService
	if ls == nil || ls.GetApplication() == nil {
		return
	}
	ls.GetApplication().RecordCustomEvent("HealthCheckError", attrs)
}
