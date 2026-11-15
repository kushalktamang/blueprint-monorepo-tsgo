package middleware

import (
	"context"
	"log/slog"
	"net"
	"net/http"

	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/logger"
	"github.com/kushalktamang/blueprint-monorepo-tsgo/internal/server"
	"github.com/newrelic/go-agent/v3/newrelic"
)

type logContextKey struct{}

type ContextEnhancer struct {
	server *server.Server
}

func NewContextEnhancer(server *server.Server) *ContextEnhancer {
	return &ContextEnhancer{server: server}
}

func (contextEnhancer *ContextEnhancer) EnhanceContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		log := contextEnhancer.server.Logger.With(
			slog.String("request_id", GetRequestID(ctx)),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("ip", clientIp(r)),
		)

		if txn := newrelic.FromContext(ctx); txn != nil {
			log = logger.WithTraceContext(log, txn)
		}

		// Set by RequireAuth, so this middleware must run after it
		if userID, ok := UserID(ctx); ok && userID != "" {
			log = log.With(slog.String("user_id", userID))
		}
		if role, ok := UserRole(ctx); ok && role != "" {
			log = log.With(slog.String("user_role", role))
		}

		ctx = context.WithValue(ctx, logContextKey{}, log)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func Getlog(ctx context.Context) *slog.Logger {
	if log, ok := ctx.Value(logContextKey{}).(*slog.Logger); ok {
		return log
	}
	return slog.Default()
}

func clientIp(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
